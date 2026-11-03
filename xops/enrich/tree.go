// Package enrich holds the Enrich-stage commands shared by every domain.
package enrich

import (
	"fmt"
	"sort"
	"strings"

	"github.com/PopGreenTh/OpenXOps/record"
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
	OUAliases         map[string]string
	AccountAliases    map[string]string
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
// ou_urn and account_urn, so the two agree. Other kinds in recs are ignored.
func LZTree(tenant string, lz LZ, recs []record.Record) []record.Record {
	lzURN := urn.Tenant(tenant).Child(typeLZ, lz.Code)
	ouURN := func(path string) urn.URN { return lzURN.Child(typeOU, urn.OUPath(path)) }

	byID := map[string]urn.URN{}
	for _, r := range recs {
		if r.Kind() == "ou" {
			byID[str(r, "id")] = ouURN(str(r, "path"))
		}
	}

	var ous, accs []record.Record
	for _, r := range recs {
		switch r.Kind() {
		case "ou":
			path := str(r, "path")
			name := str(r, "name")
			id := str(r, "id")
			parent := lzURN
			if path != "/" {
				if p, ok := byID[str(r, "parent_id")]; ok {
					parent = p
				} else {
					parent = ouURN(parentPath(path))
				}
			} else if name == "" {
				name = "Root"
			}
			alias := str(r, "alias")
			if alias == "" && lz.OUAliases != nil {
				if a, ok := lz.OUAliases[path]; ok {
					alias = a
				} else if a, ok := lz.OUAliases[id]; ok {
					alias = a
				} else if a, ok := lz.OUAliases[name]; ok {
					alias = a
				}
			}
			ous = append(ous, node(tenant, str(r, "csp"), typeOU, ouURN(path), parent, urn.Slug(name), name, id, path, "", "", "", alias))
		case "account":
			parent := ouURN(str(r, "ou_path"))
			id := str(r, "id")
			name := str(r, "name")
			alias := str(r, "alias")
			if alias == "" && lz.AccountAliases != nil {
				if a, ok := lz.AccountAliases[id]; ok {
					alias = a
				} else if a, ok := lz.AccountAliases[name]; ok {
					alias = a
				}
			}
			accs = append(accs, node(tenant, str(r, "csp"), typeAcc, parent.Child(typeAcc, id), parent, id, name, id, str(r, "ou_path"), str(r, "status"),
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
