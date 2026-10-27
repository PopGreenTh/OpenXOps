package enrich

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/PopGreenTh/OpenXOps/record"
	"gopkg.in/yaml.v3"
)

type mapCase struct {
	Name     string          `yaml:"name"`
	Taxonomy Taxonomy        `yaml:"taxonomy"`
	Records  []record.Record `yaml:"records"`
	Expect   mapCaseExpect   `yaml:"expect"`
}

type mapCaseExpect struct {
	Allocations []record.Record  `yaml:"allocations"`
	Costs       []record.Record  `yaml:"costs"`
	Findings    []mapCaseFinding `yaml:"findings"`
}

type mapCaseFinding struct {
	Code    string `yaml:"code"`
	Subject string `yaml:"subject"`
}

func TestMapReferenceCases(t *testing.T) {
	dir := filepath.Join("testdata", "cases", "map")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			runMapCase(t, filepath.Join(dir, e.Name()))
		})
	}
}

func runMapCase(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var c mapCase
	if err := yaml.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	out := Map(c.Taxonomy, c.Records)
	gotAlloc := filterKind(out, "allocation")
	gotCost := filterKind(out, "cost")
	gotFind := filterKind(out, "finding")

	if !matchPartialRecords(gotAlloc, c.Expect.Allocations, []string{
		"account_id", "lz", "ou_path", "project", "app", "environment", "business_unit", "cost_center", "app_rule", "environment_rule", "shared",
	}) {
		t.Fatalf("allocations mismatch\ngot: %v\nwant: %v", gotAlloc, c.Expect.Allocations)
	}
	if len(c.Expect.Costs) > 0 && !matchPartialRecords(gotCost, c.Expect.Costs, []string{
		"account_id", "lz", "project", "app", "environment", "business_unit", "cost_center", "app_rule", "environment_rule", "shared", "amount", "app_tag",
	}) {
		t.Fatalf("costs mismatch\ngot: %v\nwant: %v", gotCost, c.Expect.Costs)
	}
	if !matchFindings(gotFind, c.Expect.Findings) {
		t.Fatalf("findings mismatch\ngot: %v\nwant: %v", gotFind, c.Expect.Findings)
	}
}

func filterKind(recs []record.Record, kind string) []record.Record {
	var out []record.Record
	for _, r := range recs {
		if r.Kind() == kind {
			out = append(out, r)
		}
	}
	return out
}

func matchPartialRecords(got, want []record.Record, fields []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		for _, f := range fields {
			if want[i][f] == nil {
				continue
			}
			gv, wv := got[i][f], want[i][f]
			if fmtStr(gv) != fmtStr(wv) {
				return false
			}
		}
	}
	return true
}

func fmtStr(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case bool:
		if x {
			return "true"
		}
		return "false"
	default:
		return strID(v)
	}
}

func matchFindings(got []record.Record, want []mapCaseFinding) bool {
	type pair struct{ code, subject string }
	set := map[pair]struct{}{}
	for _, f := range got {
		set[pair{strID(f["code"]), strID(f["subject"])}] = struct{}{}
	}
	if len(set) != len(want) {
		return false
	}
	for _, w := range want {
		if _, ok := set[pair{w.Code, w.Subject}]; !ok {
			return false
		}
	}
	return true
}

func TestDemoTaxonomyValidate(t *testing.T) {
	tax, err := LoadTaxonomy("testdata/demo-taxonomy.yaml")
	if err != nil {
		t.Fatal(err)
	}
	findings := ValidateTaxonomy(tax)
	if HasErrorSeverity(findings) {
		t.Fatalf("demo taxonomy errors: %v", findings)
	}
}

func TestDemoMapSecurityUnmapped(t *testing.T) {
	tax, err := LoadTaxonomy("testdata/demo-taxonomy.yaml")
	if err != nil {
		t.Fatal(err)
	}
	recs, err := record.ReadFile("testdata/demo-accounts.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	out := Map(tax, recs)
	var unmapped int
	for _, r := range out {
		if r.Kind() == "finding" && r["code"] == "account-unmapped" {
			unmapped++
		}
	}
	if unmapped < 2 {
		t.Fatalf("expected Security accounts unmapped, got %d account-unmapped findings", unmapped)
	}
	allocs := filterKind(out, "allocation")
	acctSet := map[string]struct{}{}
	for _, a := range allocs {
		acctSet[strID(a["account_id"])] = struct{}{}
	}
	for _, id := range []string{"200000000001", "200000000002", "200000000003", "200000000004", "200000000005", "200000000010", "200000000011", "200000000099", "111111111111"} {
		if _, ok := acctSet[id]; !ok {
			t.Fatalf("missing allocation for account %s", id)
		}
	}
	sort.Slice(allocs, func(i, j int) bool {
		return strID(allocs[i]["account_id"]) < strID(allocs[j]["account_id"])
	})
}
