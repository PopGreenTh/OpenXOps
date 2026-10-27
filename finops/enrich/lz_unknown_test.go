package enrich

import (
	"testing"

	"github.com/PopGreenTh/OpenXOps/record"
)

func TestLzUnknownOncePerOrganization(t *testing.T) {
	tax, err := LoadTaxonomy("testdata/demo-taxonomy.yaml")
	if err != nil {
		t.Fatal(err)
	}
	recs := []record.Record{
		{"kind": "ou", "csp": "aws", "org_id": "o-other", "management_account": "999999999999", "id": "r-9", "name": "Root", "parent_id": "", "path": "/", "id_path": "r-9", "has_children": false},
		{"kind": "account", "csp": "aws", "org_id": "o-other", "management_account": "999999999999", "id": "900000000001", "name": "a1", "email": "a1@example.com", "status": "ACTIVE", "parent_id": "r-9", "ou_path": "/", "ou_id_path": "r-9"},
		{"kind": "account", "csp": "aws", "org_id": "o-other", "management_account": "999999999999", "id": "900000000002", "name": "a2", "email": "a2@example.com", "status": "ACTIVE", "parent_id": "r-9", "ou_path": "/", "ou_id_path": "r-9"},
	}
	out := Map(tax, recs)
	var lzUnknown int
	for _, r := range out {
		if r.Kind() == "finding" && r["code"] == "lz-unknown" {
			lzUnknown++
		}
	}
	if lzUnknown != 1 {
		t.Fatalf("want one lz-unknown finding per org, got %d", lzUnknown)
	}
}
