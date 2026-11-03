package tree

import (
	"strings"
	"testing"

	"github.com/PopGreenTh/OpenXOps/record"
)

func nd(typ, id, urn, parent string) record.Record {
	return record.Record{"kind": "node", "tenant": "testtnt", "csp": "aws", "type": typ, "urn": urn, "parent_urn": parent,
		"code": id, "name": "n-" + id, "ref_id": "s-" + id, "path": "", "status": ""}
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
		got = append(got, c.Type+":"+c.Code)
	}
	want := "org:alpha,org:zeta,acc:111111111111,acc:200000000002"
	if strings.Join(got, ",") != want {
		t.Fatalf("children = %v, want %s (OUs before accounts, each in code order)", got, want)
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
		"tenant: testtnt\nlz:\n  type: lzc\n  code: main\n",
		"  ref_id: s-main\n  children:\n    - type: org\n      code: root\n",
		`code: "111111111111"`, // a numeric-looking code is quoted
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

func TestJoinedOnAccountsOnly(t *testing.T) {
	recs := goodNodes()
	for _, r := range recs {
		r["joined_at"], r["joined_method"] = "", ""
		if r["type"] == "acc" && r["code"] == "111111111111" {
			r["joined_at"], r["joined_method"] = "2019-03-04T09:12:45Z", "CREATED"
		}
	}
	doc, err := Build("testtnt", "nodes.jsonl", recs)
	if err != nil {
		t.Fatal(err)
	}
	data, err := Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	// On the account that has them: after status, joined_at, then joined_method, at one indent.
	lines := strings.Split(text, "\n")
	found := false
	for i, l := range lines {
		if strings.TrimSpace(l) == `joined_at: "2019-03-04T09:12:45Z"` && i > 0 && i+1 < len(lines) {
			indent := l[:len(l)-len(strings.TrimLeft(l, " "))]
			found = lines[i-1] == indent+"status: ACTIVE" && lines[i+1] == indent+"joined_method: CREATED"
		}
	}
	if !found {
		t.Errorf("YAML lacks status, joined_at, joined_method in that order:\n%s", text)
	}
	// The other account has empty ones: no key. LZ and OUs: no key.
	if n := strings.Count(text, "joined_at:"); n != 1 {
		t.Errorf("%d joined_at keys, want 1:\n%s", n, text)
	}
	if n := strings.Count(text, "joined_method:"); n != 1 {
		t.Errorf("%d joined_method keys, want 1:\n%s", n, text)
	}
}

func TestBuildTaxonomy(t *testing.T) {
	doc, err := BuildTaxonomy("testtnt", "nodes.jsonl", goodNodes())
	if err != nil {
		t.Fatal(err)
	}
	if doc.Tenant != "testtnt" || len(doc.LZConns) != 1 {
		t.Fatalf("doc = %+v", doc)
	}
	lz := doc.LZConns[0]
	if lz.Code != "main" || lz.ManagementAccount != "s-main" {
		t.Fatalf("lz = %+v", lz)
	}
	data, err := MarshalTaxonomy(doc)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "lz_conns:") {
		t.Errorf("want lz_conns in YAML, got:\n%s", text)
	}
	if !strings.Contains(text, "ou_paths:") {
		t.Errorf("want ou_paths in YAML, got:\n%s", text)
	}
	if !strings.Contains(text, "accounts:") {
		t.Errorf("want accounts in YAML, got:\n%s", text)
	}
}

func TestBuildWithAliases(t *testing.T) {
	nodes := goodNodes()
	nodes[2]["alias"] = "main-alias"
	nodes[4]["alias"] = "root-alias"
	nodes[3]["alias"] = "acc-alias"

	doc, err := Build("testtnt", "nodes.jsonl", nodes)
	if err != nil {
		t.Fatal(err)
	}
	data, err := Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "alias: main-alias") {
		t.Errorf("want alias: main-alias in YAML, got:\n%s", text)
	}
	if !strings.Contains(text, "alias: root-alias") {
		t.Errorf("want alias: root-alias in YAML, got:\n%s", text)
	}
	if !strings.Contains(text, "alias: acc-alias") {
		t.Errorf("want alias: acc-alias in YAML, got:\n%s", text)
	}

	taxDoc, err := BuildTaxonomy("testtnt", "nodes.jsonl", nodes)
	if err != nil {
		t.Fatal(err)
	}
	taxData, err := MarshalTaxonomy(taxDoc)
	if err != nil {
		t.Fatal(err)
	}
	taxText := string(taxData)
	if !strings.Contains(taxText, "alias: main-alias") {
		t.Errorf("want alias: main-alias in taxonomy YAML, got:\n%s", taxText)
	}
	if !strings.Contains(taxText, "alias: acc-alias") {
		t.Errorf("want alias: acc-alias in taxonomy YAML, got:\n%s", taxText)
	}
}
