package enrich

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/PopGreenTh/OpenXOps/finops/enrich/share"
	"github.com/PopGreenTh/OpenXOps/record"
	"gopkg.in/yaml.v3"
)

type shareCase struct {
	Name     string          `yaml:"name"`
	Taxonomy Taxonomy        `yaml:"taxonomy"`
	Records  []record.Record `yaml:"records"`
	Expect   shareCaseExpect `yaml:"expect"`
}

type shareCaseExpect struct {
	Costs    []record.Record  `yaml:"costs"`
	Findings []mapCaseFinding `yaml:"findings"`
}

func TestShareReferenceCases(t *testing.T) {
	dir := filepath.Join("testdata", "cases", "share")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			runShareCase(t, filepath.Join(dir, e.Name()))
		})
	}
}

func runShareCase(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var c shareCase
	if err := yaml.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	out, err := share.Split(c.Records, shareOptions(c.Taxonomy))
	if err != nil {
		t.Fatal(err)
	}
	gotCost := filterKind(out, "cost")
	gotFind := filterKind(out, "finding")
	if !matchShareCosts(gotCost, c.Expect.Costs) {
		t.Fatalf("costs mismatch\ngot: %v\nwant: %v", gotCost, c.Expect.Costs)
	}
	if !matchFindings(gotFind, c.Expect.Findings) {
		t.Fatalf("findings mismatch\ngot: %v\nwant: %v", gotFind, c.Expect.Findings)
	}
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	default:
		return 0, false
	}
}

func matchShareCosts(got, want []record.Record) bool {
	if len(got) != len(want) {
		return false
	}
	fields := append([]string{"account_id", "app", "project", "business_unit", "cost_center", "environment", "app_rule", "currency", "share_rule", "tenant"}, urnCaseFields...)
	for i := range want {
		for _, f := range fields {
			if want[i][f] == nil {
				continue
			}
			if fmtStr(got[i][f]) != fmtStr(want[i][f]) {
				return false
			}
		}
		ga, _ := toFloat(got[i]["amount"])
		wa, _ := toFloat(want[i]["amount"])
		if math.Abs(ga-wa) > 1e-6 {
			return false
		}
		if want[i]["source_amount"] != nil {
			gs, _ := toFloat(got[i]["source_amount"])
			ws, _ := toFloat(want[i]["source_amount"])
			if math.Abs(gs-ws) > 1e-6 {
				return false
			}
		}
		if want[i]["share"] != nil {
			gs, _ := toFloat(got[i]["share"])
			ws, _ := toFloat(want[i]["share"])
			if math.Abs(gs-ws) > 1e-6 {
				return false
			}
		}
	}
	return true
}

func TestShareConservationFailure(t *testing.T) {
	tax := Taxonomy{Tags: &TagKeys{App: "App"}}
	recs := []record.Record{
		{"kind": "allocation", "account_id": "1", "app": "APP-A", "shared": true, "project": "P", "business_unit": "B", "cost_center": "C", "app_rule": "account", "environment": "E"},
		{"kind": "allocation", "account_id": "1", "app": "APP-B", "shared": true, "project": "P", "business_unit": "B", "cost_center": "C", "app_rule": "account", "environment": "E"},
		{"kind": "cost", "account_id": "1", "period": "2026-09", "metric": "UnblendedCost", "currency": "USD", "amount": 10.0, "shared": true, "app_tag": ""},
	}
	_, err := share.Split(recs, shareOptions(tax))
	if err != nil {
		return
	}
	out := []record.Record{{"account_id": "1", "period": "2026-09", "metric": "UnblendedCost", "currency": "USD", "amount": 9.0}}
	in := []record.Record{{"account_id": "1", "period": "2026-09", "metric": "UnblendedCost", "currency": "USD", "amount": 10.0, "shared": true}}
	if err := share.CheckConservation(in, out); err == nil {
		t.Fatal("expected conservation error")
	}
}
