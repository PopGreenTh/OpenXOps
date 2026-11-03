// Package enrich holds the Enrich-stage commands shared by every domain.
package enrich

import (
	"fmt"
	"sort"
	"strings"

	"github.com/PopGreenTh/OpenXOps/record"
	"github.com/PopGreenTh/OpenXOps/record/taxonomy"
	"github.com/PopGreenTh/OpenXOps/record/urn"
)

// Node types (D-046): the URN type of the level a node record stands for.
const (
	typeLZ  = "lzc"
	typeOU  = "org"
	typeAcc = "acc"
)

// LZ is the landing zone a tree hangs from: the taxonomy's one LZ (D-052).
type LZ struct {
	Code              string
	Name              string
	CSP               string
	ManagementAccount string
	Alias             string
	// Cloud holds the codes the taxonomy lists for the LZ's OUs and accounts (D-059); the zero
	// value lists none, and codes come from the discovered names.
	Cloud          taxonomy.Cloud
	OUAliases      map[string]string
	AccountAliases map[string]string
}

// CheckManagement returns an error naming source, the line and both accounts for the first ou or
// account record in recs whose management_account is not the LZ's. recs are one input's records
// in file order, so a record's line is its position plus one.
func CheckManagement(source string, recs []record.Record, lz LZ) error {
	for i, r := range recs {
		if k := r.Kind(); k != "ou" && k != "account" {
			continue
		}
		if got := str(r, "management_account"); got != lz.ManagementAccount {
			return fmt.Errorf("%s:%d: %s record is for management account %q, but LZ %q has %q",
				source, i+1, r.Kind(), got, lz.Code, lz.ManagementAccount)
		}
	}
	return nil
}

// LZTree turns ou and account records into node records: the LZ node, then the OU nodes sorted by
// urn, then the account nodes sorted by urn. URNs are built as finops enrich map builds lz_urn,
// ou_urn and account_urn (D-059), by the same function, so the two agree. Other kinds in recs are
// ignored.
func LZTree(tenant string, lz LZ, recs []record.Record) []record.Record {
	lzURN := urn.Tenant(tenant).Child(typeLZ, lz.Code)
	ouOf := func(r record.Record) taxonomy.DiscoveredOU {
		return taxonomy.DiscoveredOU{ID: str(r, "id"), Path: str(r, "path"), Name: str(r, "name")}
	}

	byID := map[string]taxonomy.DiscoveredOU{}
	byPath := map[string]taxonomy.DiscoveredOU{}
	for _, r := range recs {
		if r.Kind() == "ou" {
			byID[str(r, "id")] = ouOf(r)
			byPath[str(r, "path")] = ouOf(r)
		}
	}
	// ouAbove finds the OU an account sits in; one that is not in recs is made from its path.
	ouAbove := func(r record.Record) taxonomy.DiscoveredOU {
		if ou, ok := byID[str(r, "parent_id")]; ok {
			return ou
		}
		path := str(r, "ou_path")
		if ou, ok := byPath[path]; ok {
			return ou
		}
		return taxonomy.DiscoveredOU{Path: path, Name: taxonomy.PathName(path)}
	}

	var ous, accs []record.Record
	for _, r := range recs {
		switch r.Kind() {
		case "ou":
			ou := ouOf(r)
			name := ou.Name
			parent := lzURN
			if ou.Path != "/" {
				if p, ok := byID[str(r, "parent_id")]; ok {
					parent = lz.Cloud.OUURN(lzURN, p)
				} else {
					pp := parentPath(ou.Path)
					p, ok := byPath[pp]
					if !ok {
						p = taxonomy.DiscoveredOU{Path: pp, Name: taxonomy.PathName(pp)}
					}
					parent = lz.Cloud.OUURN(lzURN, p)
				}
			} else if name == "" {
				name = "Root"
			}
			alias := str(r, "alias")
			if alias == "" && lz.OUAliases != nil {
				if a, ok := lz.OUAliases[ou.ID]; ok {
					alias = a
				} else if a, ok := lz.OUAliases[ou.Path]; ok {
					alias = a
				}
			}
			ous = append(ous, node(tenant, str(r, "csp"), typeOU, lz.Cloud.OUURN(lzURN, ou), parent, lz.Cloud.OUCode(ou), name, ou.ID, ou.Path, "", "", "", alias))
		case "account":
			ou := ouAbove(r)
			parent := lz.Cloud.OUURN(lzURN, ou)
			acct := taxonomy.DiscoveredAccount{ID: str(r, "id"), Name: str(r, "name")}
			alias := str(r, "alias")
			if alias == "" && lz.AccountAliases != nil {
				if a, ok := lz.AccountAliases[acct.ID]; ok {
					alias = a
				}
			}
			accs = append(accs, node(tenant, str(r, "csp"), typeAcc, lz.Cloud.AccountURN(lzURN, ou, acct), parent, lz.Cloud.AccountCode(acct), acct.Name, acct.ID, str(r, "ou_path"), str(r, "status"),
				str(r, "joined_at"), str(r, "joined_method"), alias))
		}
	}
	sortByURN(ous)
	sortByURN(accs)

	out := []record.Record{node(tenant, lz.CSP, typeLZ, lzURN, "", lz.Code, lz.Name, lz.ManagementAccount, "", "", "", "", lz.Alias)}
	out = append(out, ous...)
	return append(out, accs...)
}

// node builds one node record. joinedAt and joinedMethod are the account's (D-055), "" elsewhere.
// code is the LZ code, the slug of an OU's own name, or the account id; refID is the id of the
// cloud resource the node stands for (D-056). alias is the optional alias of the node (D-057).
func node(tenant, csp, typ string, u, parent urn.URN, code, name, refID, path, status, joinedAt, joinedMethod, alias string) record.Record {
	r := record.Record{
		"kind":          "node",
		"tenant":        tenant,
		"csp":           csp,
		"type":          typ,
		"urn":           u.String(),
		"parent_urn":    parent.String(),
		"code":          code,
		"name":          name,
		"ref_id":        refID,
		"path":          path,
		"status":        status,
		"joined_at":     joinedAt,
		"joined_method": joinedMethod,
	}
	if alias != "" {
		r["alias"] = alias
	}
	return r
}

// CheckNodes checks the node records LZTree made (D-056). It returns an error naming the node when
// a node has an empty code, urn or name, or a urn that breaks the URN rule. It returns one warning
// per node whose code breaks the code rule (an OU named `IT` slugs to `it`); such a code is kept.
func CheckNodes(nodes []record.Record) (warnings []string, err error) {
	if err := checkCollisions(nodes); err != nil {
		return nil, err
	}
	for _, n := range nodes {
		who := fmt.Sprintf("%s node %q", str(n, "type"), str(n, "urn"))
		if str(n, "urn") == "" {
			who = fmt.Sprintf("%s node named %q", str(n, "type"), str(n, "name"))
		}
		for _, f := range []string{"code", "urn", "name"} {
			if str(n, f) == "" {
				return warnings, fmt.Errorf("%s has an empty %s", who, f)
			}
		}
		if !urn.ValidURN(str(n, "urn")) {
			return warnings, fmt.Errorf("%s: the urn breaks the URN rule (11 to 250 characters from a-z 0-9 . : + = @ _ / -)", who)
		}
		if !urn.ValidCode(str(n, "code")) {
			warnings = append(warnings, fmt.Sprintf("%s: code %q breaks the code rule (3 to 50 characters from letters, digits and . + = @ _ -); kept as it is", who, str(n, "code")))
		}
	}
	return warnings, nil
}

// checkCollisions returns an error naming both OU paths or both account ids when two nodes of one
// type have one URN (D-059). Two records for the same OU or account are not a collision.
func checkCollisions(nodes []record.Record) error {
	type seen struct{ path, id string }
	first := map[string]seen{}
	for _, n := range nodes {
		typ := str(n, "type")
		if typ != typeOU && typ != typeAcc {
			continue
		}
		u := str(n, "urn")
		cur := seen{str(n, "path"), str(n, "ref_id")}
		prev, dup := first[u]
		if !dup {
			first[u] = cur
			continue
		}
		if typ == typeOU && prev.path != cur.path {
			return fmt.Errorf("OUs %s and %s both get the URN %s; list one of them under ou_paths with a code of its own", prev.path, cur.path, u)
		}
		if typ == typeAcc && prev.id != cur.id {
			return fmt.Errorf("accounts %s and %s both get the URN %s; list one of them under accounts with a code of its own", prev.id, cur.id, u)
		}
	}
	return nil
}

// parentPath returns the path of the OU above path ("/Platform/Hub" gives "/Platform").
func parentPath(path string) string {
	i := strings.LastIndex(path, "/")
	if i <= 0 {
		return "/"
	}
	return path[:i]
}

func sortByURN(recs []record.Record) {
	sort.SliceStable(recs, func(i, j int) bool { return str(recs[i], "urn") < str(recs[j], "urn") })
}

// str returns r[key] as a string, "" when it is missing or null.
func str(r record.Record, key string) string {
	v, ok := r[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}
