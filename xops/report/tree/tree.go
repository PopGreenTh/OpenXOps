// Package tree turns node records into the nested YAML tree of a tenant's cloud (D-054).
package tree

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/PopGreenTh/OpenXOps/record"
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
				SourceID: str(r, "source_id"), Status: str(r, "status"),
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
