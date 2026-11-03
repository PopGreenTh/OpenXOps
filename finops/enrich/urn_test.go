package enrich

import (
	"strings"
	"testing"

	"github.com/PopGreenTh/OpenXOps/finops/enrich/share"
	"github.com/PopGreenTh/OpenXOps/record"
	"github.com/PopGreenTh/OpenXOps/record/taxonomy"
)

// nd makes a taxonomy node.
func nd(code, name string) taxonomy.Node { return taxonomy.Node{Code: code, Name: name} }

// urnTaxonomy has a cost center (pay) under another business unit than its project's (retail).
func urnTaxonomy() taxonomy.Taxonomy {
	return taxonomy.Taxonomy{
		Tenant: nd("testtnt", "Test tenant"),
		BusinessUnits: []taxonomy.BusinessUnit{
			{Node: nd("retail", "Retail")},
			{Node: nd("digital", "Digital"), CostCenters: []taxonomy.CostCenter{{Node: nd("pay", "Payments")}}},
		},
		Projects: []taxonomy.Project{{
			Node: nd("shop", "Shop"), RefURNBU: "urn:tnt:testtnt:bzu:retail",
			Apps: []taxonomy.App{{
				Node: nd("chk", "Checkout"), RefURNCCT: "urn:tnt:testtnt:bzu:digital:cct:pay",
				Environments: []taxonomy.Environment{{Node: nd("prod", "Prod")}},
			}},
		}},
		LZs:      []taxonomy.LZ{{Node: nd("main", "Main"), CSP: "aws", Currency: "USD", ManagementAccount: "111111111111"}},
		Defaults: &taxonomy.Defaults{App: "chk", Environment: "prod"},
	}
}

func urnInventory() []record.Record {
	return []record.Record{
		{"kind": "ou", "csp": "aws", "org_id": "o-main", "management_account": "111111111111", "id": "r-1", "name": "Root", "parent_id": "", "path": "/", "id_path": "r-1", "has_children": true},
		{"kind": "ou", "csp": "aws", "org_id": "o-main", "management_account": "111111111111", "id": "ou-1", "name": "Shared Services", "parent_id": "r-1", "path": "/Shared Services", "id_path": "r-1/ou-1", "has_children": false},
		{"kind": "account", "csp": "aws", "org_id": "o-main", "management_account": "111111111111", "id": "200000000001", "name": "a", "email": "a@example.com", "status": "ACTIVE", "parent_id": "ou-1", "ou_path": "/Shared Services", "ou_id_path": "r-1/ou-1"},
	}
}

func costRow(account, service string) record.Record {
	return record.Record{"kind": "cost", "csp": "aws", "management_account": "111111111111", "account_id": account, "granularity": "monthly",
		"period": "2026-09", "metric": "UnblendedCost", "service": service, "app_tag": "", "amount": 10.0, "currency": "USD", "estimated": false}
}

func byKind(recs []record.Record, kind string) []record.Record { return filterKind(recs, kind) }

func TestCostCenterURNSitsUnderItsOwnBusinessUnit(t *testing.T) {
	out := Map(urnTaxonomy(), urnInventory())
	allocs := byKind(out, "allocation")
	if len(allocs) != 1 {
		t.Fatalf("allocations = %v", allocs)
	}
	a := allocs[0]
	want := map[string]string{
		"business_unit":     "retail",
		"business_unit_urn": "urn:tnt:testtnt:bzu:retail",
		"cost_center_urn":   "urn:tnt:testtnt:bzu:digital:cct:pay",
		"ou_urn":            "urn:tnt:testtnt:lzc:main:org:shared-services",
		"account_urn":       "urn:tnt:testtnt:lzc:main:org:shared-services:acc:200000000001",
		"tenant":            "testtnt",
	}
	for k, v := range want {
		if a[k] != v {
			t.Errorf("%s = %v, want %s", k, a[k], v)
		}
	}
}

func TestServiceURNIsAccountURNPlusServiceSlug(t *testing.T) {
	recs := append(urnInventory(), costRow("200000000001", "Amazon Simple Storage Service"))
	costs := byKind(Map(urnTaxonomy(), recs), "cost")
	if len(costs) != 1 {
		t.Fatalf("costs = %v", costs)
	}
	got := costs[0]["service_urn"]
	want := "urn:tnt:testtnt:lzc:main:org:shared-services:acc:200000000001:svc:amazon-simple-storage-service"
	if got != want {
		t.Fatalf("service_urn = %v, want %s", got, want)
	}
}

func TestCostOfAccountNotInInventoryHasEveryURN(t *testing.T) {
	recs := append(urnInventory(), costRow("300000000009", "AmazonS3"))
	var cost record.Record
	for _, c := range byKind(Map(urnTaxonomy(), recs), "cost") {
		if c["account_id"] == "300000000009" {
			cost = c
		}
	}
	if cost == nil {
		t.Fatal("no cost row for the unknown account")
	}
	want := map[string]string{
		"business_unit_urn": "urn:tnt:testtnt:bzu:unalloc",
		"cost_center_urn":   "urn:tnt:testtnt:bzu:unalloc:cct:unalloc",
		"project_urn":       "urn:tnt:testtnt:prj:unalloc",
		"app_urn":           "urn:tnt:testtnt:prj:unalloc:app:unalloc",
		"environment_urn":   "urn:tnt:testtnt:prj:unalloc:app:unalloc:env:unalloc",
		"lz_urn":            "urn:tnt:testtnt:lzc:main",
		"ou_urn":            "urn:tnt:testtnt:lzc:main:org:unknown",
		"account_urn":       "urn:tnt:testtnt:lzc:main:org:unknown:acc:300000000009",
		"service_urn":       "urn:tnt:testtnt:lzc:main:org:unknown:acc:300000000009:svc:amazon-s3",
	}
	for k, v := range want {
		if cost[k] != v {
			t.Errorf("%s = %v, want %s", k, cost[k], v)
		}
	}
}

func TestEveryCostAfterShareHasAllNineURNs(t *testing.T) {
	tax := urnTaxonomy()
	tax.Projects[0].Apps = append(tax.Projects[0].Apps, taxonomy.App{
		Node: nd("pay", "Payments"), RefURNCCT: "urn:tnt:testtnt:bzu:digital:cct:pay",
		Environments: []taxonomy.Environment{{Node: nd("prod", "Prod")}},
	})
	tax.LZs[0].OUPaths = []taxonomy.OUPath{{
		Node: nd("shared-services", "Shared Services"), Path: "/Shared Services",
		Accounts: []taxonomy.Account{{
			Node: nd("shared-account", "a"), RefID: "200000000001",
			RefURNApps: []string{"urn:tnt:testtnt:prj:shop:app:chk", "urn:tnt:testtnt:prj:shop:app:pay"},
			RefURNEnv:  "urn:tnt:testtnt:prj:shop:app:chk:env:prod",
		}},
	}}
	recs := urnInventory()
	for _, tag := range []string{"", "chk", "chk,pay", "nope"} {
		c := costRow("200000000001", "AmazonEC2")
		c["app_tag"] = tag
		recs = append(recs, c)
	}
	recs = append(recs, costRow("300000000009", "AmazonS3"))
	mapped := Map(tax, recs)
	shared, err := share.Split(mapped, shareOptions(tax))
	if err != nil {
		t.Fatal(err)
	}
	nine := []string{"business_unit_urn", "cost_center_urn", "project_urn", "app_urn", "environment_urn", "lz_urn", "ou_urn", "account_urn", "service_urn"}
	costs := byKind(shared, "cost")
	if len(costs) < 5 {
		t.Fatalf("costs = %d", len(costs))
	}
	for i, c := range costs {
		for _, f := range nine {
			v, _ := c[f].(string)
			if !strings.HasPrefix(v, "urn:tnt:testtnt:") {
				t.Errorf("cost %d (%v / %v): %s = %q", i, c["account_id"], c["app"], f, v)
			}
		}
	}
}

func TestMixedCaseCodeKeepsCaseInRecordsAndLowercasesInURNs(t *testing.T) {
	tax := urnTaxonomy()
	tax.Projects = []taxonomy.Project{{
		Node: nd("PRJ_Shop", "Shop"), RefURNBU: "urn:tnt:testtnt:bzu:retail",
		Apps: []taxonomy.App{{
			Node: nd("Chk_App", "Checkout"), RefURNCCT: "urn:tnt:testtnt:bzu:digital:cct:pay",
			Environments: []taxonomy.Environment{{Node: nd("Prod", "Prod")}},
		}},
	}}
	tax.Defaults = &taxonomy.Defaults{App: "Chk_App", Environment: "Prod"}
	out := Map(tax, urnInventory())
	allocs := byKind(out, "allocation")
	if len(allocs) != 1 {
		t.Fatalf("allocations = %v", allocs)
	}
	a := allocs[0]
	want := map[string]string{
		"project":         "PRJ_Shop",
		"app":             "Chk_App",
		"environment":     "Prod",
		"project_urn":     "urn:tnt:testtnt:prj:prj_shop",
		"app_urn":         "urn:tnt:testtnt:prj:prj_shop:app:chk_app",
		"environment_urn": "urn:tnt:testtnt:prj:prj_shop:app:chk_app:env:prod",
	}
	for k, v := range want {
		if a[k] != v {
			t.Errorf("%s = %v, want %s", k, a[k], v)
		}
	}
	if problems := validateTaxonomyOK(tax); problems != "" {
		t.Errorf("the taxonomy should validate: %s", problems)
	}
}

// validateTaxonomyOK returns the error findings of ValidateTaxonomy as a string, "" for none.
func validateTaxonomyOK(tax taxonomy.Taxonomy) string {
	var out []string
	for _, f := range ValidateTaxonomy(tax) {
		if f["severity"] == "error" {
			out = append(out, strID(f["code"])+":"+strID(f["subject"]))
		}
	}
	return strings.Join(out, ", ")
}
