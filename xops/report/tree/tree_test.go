package tree

import (
	"strings"
	"testing"

	"github.com/PopGreenTh/OpenXOps/record"
)

func nd(typ, id, urn, parent string) record.Record {
	return record.Record{"kind": "node", "tenant": "testtnt", "csp": "aws", "type": typ, "urn": urn, "parent_urn": parent,
		"id": id, "name": "n-" + id, "source_id": "s-" + id, "path": "", "status": ""}
}

const (
	lzURN   = "urn:tnt:testtnt:lzc:main"
	rootURN = lzURN + ":org:root"
)

// goodNodes is the LZ, the root OU, one OU under it, and two accounts, deliberately out of order.
func goodNodes() []record.Record {
	acc := func(id, parent string) record.Record {
		n := nd("acc", id, parent+":acc:"+id, parent)
		n["status"] = "ACTIVE"
		return n
	}
	return []record.Record{
		acc("200000000002", rootURN),
		nd("org", "zeta", rootURN+"x", rootURN),
		nd("lzc", "main", lzURN, ""),
		acc("111111111111", rootURN),
		nd("org", "root", rootURN, lzURN),
		nd("org", "alpha", rootURN+"y", rootURN),
	}
}

func TestBuildNestsAndOrders(t *testing.T) {
	doc, err := Build("testtnt", "nodes.jsonl", goodNodes())
	if err != nil {
		t.Fatal(err)
	}
	if doc.Tenant != "testtnt" || doc.LZ.URN != lzURN || len(doc.LZ.Children) != 1 {
		t.Fatalf("doc = %+v", doc)
	}
	root := doc.LZ.Children[0]
	var got []string
	for _, c := range root.Children {
		got = append(got, c.Type+":"+c.ID)
	}
	want := "org:alpha,org:zeta,acc:111111111111,acc:200000000002"
	if strings.Join(got, ",") != want {
		t.Fatalf("children = %v, want %s (OUs before accounts, each in id order)", got, want)
	}
}

func TestMarshalShape(t *testing.T) {
	doc, err := Build("testtnt", "nodes.jsonl", goodNodes())
	if err != nil {
		t.Fatal(err)
	}
	data, err := Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"tenant: testtnt\nlz:\n  type: lzc\n  id: main\n",
		"  source_id: s-main\n  children:\n    - type: org\n      id: root\n",
		`id: "111111111111"`, // a numeric-looking id is quoted
		"status: ACTIVE\n",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("YAML lacks %q:\n%s", want, text)
		}
	}
	// A leaf has no children key; the LZ and OUs have no status key.
	if strings.Contains(text, "children: []") || strings.Contains(text, "status: \"\"") {
		t.Errorf("empty children or status written:\n%s", text)
	}
	if n := strings.Count(text, "children:"); n != 2 {
		t.Errorf("%d children keys, want 2 (the LZ and the root OU):\n%s", n, text)
	}
}

func TestBuildErrors(t *testing.T) {
	with := func(mut func([]record.Record) []record.Record) []record.Record { return mut(goodNodes()) }
	cases := []struct {
		name string
		recs []record.Record
		want []string
	}{
		{"no lzc node", with(func(r []record.Record) []record.Record { return append(r[:2], r[3:]...) }), []string{"nodes.jsonl", "no lzc node"}},
		{"two lzc nodes", with(func(r []record.Record) []record.Record {
			return append(r, nd("lzc", "other", "urn:tnt:testtnt:lzc:other", ""))
		}), []string{"nodes.jsonl:7:", "urn:tnt:testtnt:lzc:other", "exactly one"}},
		{"dangling parent_urn", with(func(r []record.Record) []record.Record {
			r[0]["parent_urn"] = "urn:tnt:testtnt:lzc:main:org:gone"
			return r
		}), []string{"nodes.jsonl:1:", "urn:tnt:testtnt:lzc:main:org:gone"}},
		{"empty parent_urn on an OU", with(func(r []record.Record) []record.Record {
			r[4]["parent_urn"] = ""
			return r
		}), []string{"nodes.jsonl:5:", rootURN}},
		{"duplicate urn", with(func(r []record.Record) []record.Record {
			return append(r, nd("org", "root2", rootURN, lzURN))
		}), []string{"nodes.jsonl:7:", "duplicate urn " + rootURN, "line 5"}},
		{"a cycle outside the tree", append(goodNodes(),
			nd("org", "a", "urn:tnt:testtnt:lzc:main:org:a", "urn:tnt:testtnt:lzc:main:org:b"),
			nd("org", "b", "urn:tnt:testtnt:lzc:main:org:b", "urn:tnt:testtnt:lzc:main:org:a")),
			[]string{"nodes.jsonl:7:", "urn:tnt:testtnt:lzc:main:org:a", "not under the lzc node"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Build("testtnt", "nodes.jsonl", tc.recs)
			if err == nil {
				t.Fatal("expected an error")
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Fatalf("error %q should contain %q", err, w)
				}
			}
		})
	}
}

func TestBuildIgnoresOtherKinds(t *testing.T) {
	recs := append([]record.Record{{"kind": "finding", "code": "x"}}, goodNodes()...)
	if _, err := Build("testtnt", "nodes.jsonl", recs); err != nil {
		t.Fatal(err)
	}
}
