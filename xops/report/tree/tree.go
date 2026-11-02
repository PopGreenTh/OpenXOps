// Package tree turns node records into the nested YAML tree of a tenant's cloud (D-054).
package tree

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/PopGreenTh/OpenXOps/record"
	"github.com/PopGreenTh/OpenXOps/record/urn"
	"gopkg.in/yaml.v3"
)

// Doc is the YAML document: the tenant and the LZ with everything under it.
type Doc struct {
	Tenant string `yaml:"tenant"`
	LZ     *Node  `yaml:"lz"`
}

// Node is one LZ, OU or account in the tree. Field order is the order in the YAML.
type Node struct {
	Type     string `yaml:"type"`
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	URN      string `yaml:"urn"`
	SourceID string `yaml:"source_id"`
	Alias    string `yaml:"alias,omitempty"`
	Status   string `yaml:"status,omitempty"`
	// JoinedAt and JoinedMethod are an account's (D-055); they are left out when empty.
	JoinedAt     string  `yaml:"joined_at,omitempty"`
	JoinedMethod string  `yaml:"joined_method,omitempty"`
	Children     []*Node `yaml:"children,omitempty"`
}

type entry struct {
	node   *Node
	parent string
	line   int
}

// Build nests node records under the one lzc node. source is the file path or "stdin", and recs
// are that input's records in file order, so a record's line is its position plus one. Records of
// other kinds are ignored. It fails, naming source, the line and the URN, when there isn't exactly
// one lzc node, when two nodes share a urn, or when a node's parent_urn names no node or the node
// is not under the lzc node. Children are the OUs, then the accounts, each group in id order.
func Build(tenant, source string, recs []record.Record) (Doc, error) {
	byURN := map[string]*entry{}
	var all []*entry
	var lz *entry
	for i, r := range recs {
		if r.Kind() != "node" {
			continue
		}
		e := &entry{
			node: &Node{
				Type: str(r, "type"), ID: str(r, "id"), Name: str(r, "name"), URN: str(r, "urn"),
				SourceID: str(r, "source_id"), Alias: str(r, "alias"), Status: str(r, "status"),
				JoinedAt: str(r, "joined_at"), JoinedMethod: str(r, "joined_method"),
			},
			parent: str(r, "parent_urn"),
			line:   i + 1,
		}
		if prev, dup := byURN[e.node.URN]; dup {
			return Doc{}, fmt.Errorf("%s:%d: duplicate urn %s (first at line %d)", source, e.line, e.node.URN, prev.line)
		}
		byURN[e.node.URN] = e
		all = append(all, e)
		if e.node.Type == "lzc" {
			if lz != nil {
				return Doc{}, fmt.Errorf("%s:%d: second lzc node %s (want exactly one, first at line %d)", source, e.line, e.node.URN, lz.line)
			}
			lz = e
		}
	}
	if lz == nil {
		return Doc{}, fmt.Errorf("%s: no lzc node (want exactly one)", source)
	}

	for _, e := range all {
		if e == lz {
			continue
		}
		p, ok := byURN[e.parent]
		if !ok {
			return Doc{}, fmt.Errorf("%s:%d: parent_urn %q of node %s names no node", source, e.line, e.parent, e.node.URN)
		}
		p.node.Children = append(p.node.Children, e.node)
	}
	if n := count(lz.node); n != len(all) {
		for _, e := range all {
			if !reachable(lz.node, e.node) {
				return Doc{}, fmt.Errorf("%s:%d: node %s is not under the lzc node", source, e.line, e.node.URN)
			}
		}
	}
	sortChildren(lz.node)
	return Doc{Tenant: tenant, LZ: lz.node}, nil
}

// Marshal writes the document as YAML with two-space indents.
func Marshal(d Doc) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(d); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func count(n *Node) int {
	total := 1
	for _, c := range n.Children {
		total += count(c)
	}
	return total
}

func reachable(from, want *Node) bool {
	if from == want {
		return true
	}
	for _, c := range from.Children {
		if reachable(c, want) {
			return true
		}
	}
	return false
}

// sortChildren orders every node's children: OUs, then accounts, then anything else, each by id.
func sortChildren(n *Node) {
	rank := func(typ string) int {
		switch typ {
		case "org":
			return 0
		case "acc":
			return 1
		}
		return 2
	}
	sort.SliceStable(n.Children, func(i, j int) bool {
		a, b := n.Children[i], n.Children[j]
		if ra, rb := rank(a.Type), rank(b.Type); ra != rb {
			return ra < rb
		}
		return a.ID < b.ID
	})
	for _, c := range n.Children {
		sortChildren(c)
	}
}

// TaxonomyDoc is the YAML document for --format taxonomy.
type TaxonomyDoc struct {
	Tenant  string            `yaml:"tenant"`
	LZConns []*TaxonomyLZConn `yaml:"lz_conns"`
}

type TaxonomyLZConn struct {
	Code              string            `yaml:"code"`
	URN               string            `yaml:"urn"`
	Name              string            `yaml:"name"`
	Alias             string            `yaml:"alias,omitempty"`
	Tenant            string            `yaml:"tenant"`
	CSP               string            `yaml:"csp"`
	Currency          string            `yaml:"currency"`
	ManagementAccount string            `yaml:"management_account"`
	RefID             string            `yaml:"ref_id"`
	OUPaths           []*TaxonomyOUPath `yaml:"ou_paths"`
}

type TaxonomyOUPath struct {
	Code     string             `yaml:"code"`
	URN      string             `yaml:"urn"`
	Name     string             `yaml:"name"`
	Alias    string             `yaml:"alias,omitempty"`
	RefID    string             `yaml:"ref_id"`
	Path     string             `yaml:"path"`
	Accounts []*TaxonomyAccount `yaml:"accounts,omitempty"`
}

type TaxonomyAccount struct {
	Code   string `yaml:"code"`
	URN    string `yaml:"urn"`
	Name   string `yaml:"name"`
	Alias  string `yaml:"alias,omitempty"`
	RefID  string `yaml:"ref_id"`
	Status string `yaml:"status,omitempty"`
}

// BuildTaxonomy builds the taxonomy lz_conns structure from node records.
// It includes only the leaf OUs where accounts reside.
func BuildTaxonomy(tenant, source string, recs []record.Record) (TaxonomyDoc, error) {
	var lzRecord record.Record
	ouMap := map[string]record.Record{}
	var accRecords []record.Record

	for i, r := range recs {
		if r.Kind() != "node" {
			continue
		}
		typ := str(r, "type")
		urnStr := str(r, "urn")
		switch typ {
		case "lzc":
			if lzRecord != nil {
				return TaxonomyDoc{}, fmt.Errorf("%s:%d: second lzc node %s (want exactly one)", source, i+1, urnStr)
			}
			lzRecord = r
		case "org":
			ouMap[urnStr] = r
		case "acc":
			accRecords = append(accRecords, r)
		}
	}

	if lzRecord == nil {
		return TaxonomyDoc{}, fmt.Errorf("%s: no lzc node (want exactly one)", source)
	}

	lzURN := str(lzRecord, "urn")
	mgmtID := str(lzRecord, "source_id")

	accsByOU := map[string][]record.Record{}
	for _, acc := range accRecords {
		pURN := str(acc, "parent_urn")
		if (pURN == lzURN || pURN == lzURN+":org:root") && str(acc, "source_id") == mgmtID {
			continue
		}
		accsByOU[pURN] = append(accsByOU[pURN], acc)
	}

	var ouPaths []*TaxonomyOUPath
	for ouURN, accs := range accsByOU {
		ouNode, ok := ouMap[ouURN]
		if !ok {
			return TaxonomyDoc{}, fmt.Errorf("%s: parent_urn %q of accounts names no OU node", source, ouURN)
		}

		ouName := str(ouNode, "name")
		ouCode := ouName
		if ouCode == "" {
			ouCode = str(ouNode, "id")
		}

		sort.SliceStable(accs, func(i, j int) bool {
			return str(accs[i], "source_id") < str(accs[j], "source_id")
		})

		var accounts []*TaxonomyAccount
		for _, a := range accs {
			accCode := str(a, "name")
			if accCode == "" {
				accCode = str(a, "id")
			}
			accURN := str(ouNode, "urn") + ":acc:" + urn.Slug(accCode)
			accounts = append(accounts, &TaxonomyAccount{
				Code:   accCode,
				URN:    accURN,
				Name:   accCode,
				Alias:  str(a, "alias"),
				RefID:  str(a, "source_id"),
				Status: str(a, "status"),
			})
		}

		ouPaths = append(ouPaths, &TaxonomyOUPath{
			Code:     ouCode,
			URN:      str(ouNode, "urn"),
			Name:     ouName,
			Alias:    str(ouNode, "alias"),
			RefID:    str(ouNode, "source_id"),
			Path:     str(ouNode, "path"),
			Accounts: accounts,
		})
	}

	sort.SliceStable(ouPaths, func(i, j int) bool {
		return ouPaths[i].Path < ouPaths[j].Path
	})

	lzCode := str(lzRecord, "id")
	lzName := str(lzRecord, "name")
	csp := str(lzRecord, "csp")
	if csp == "" {
		csp = "aws"
	}

	lzConn := &TaxonomyLZConn{
		Code:              lzCode,
		URN:               str(lzRecord, "urn"),
		Name:              lzName,
		Alias:             str(lzRecord, "alias"),
		Tenant:            tenant,
		CSP:               csp,
		Currency:          "USD",
		ManagementAccount: mgmtID,
		RefID:             mgmtID,
		OUPaths:           ouPaths,
	}

	return TaxonomyDoc{
		Tenant:  tenant,
		LZConns: []*TaxonomyLZConn{lzConn},
	}, nil
}

// MarshalTaxonomy writes the taxonomy document as YAML with two-space indents.
func MarshalTaxonomy(d TaxonomyDoc) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(d); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

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
