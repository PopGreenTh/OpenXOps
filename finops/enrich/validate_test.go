package enrich

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateErrors(t *testing.T) {
	cases := []struct {
		name     string
		tax      Taxonomy
		errCode  string
		severity string
	}{
		{
			name: "duplicate app code",
			tax: Taxonomy{
				Projects:      []Project{{Code: "PRJ-1", Name: "P", BusinessUnit: "BU-1"}},
				BusinessUnits: []BusinessUnit{{Code: "BU-1", Name: "BU"}},
				CostCenters:   []CostCenter{{Code: "CC-1", Name: "C", BusinessUnit: "BU-1"}},
				Apps: []App{
					{Code: "APP-A", Name: "A", Project: "PRJ-1", CostCenter: "CC-1", Environments: []Environment{{Code: "ENV-1", Name: "One"}}},
					{Code: "APP-A", Name: "B", Project: "PRJ-1", CostCenter: "CC-1", Environments: []Environment{{Code: "ENV-1", Name: "One"}}},
				},
			},
			errCode:  "taxonomy-duplicate-code",
			severity: "error",
		},
		{
			name: "reserved code",
			tax: Taxonomy{
				BusinessUnits: []BusinessUnit{{Code: "BU-UNALLOC", Name: "Bad"}},
			},
			errCode:  "taxonomy-reserved-code",
			severity: "error",
		},
		{
			name: "unknown project reference",
			tax: Taxonomy{
				Projects:      []Project{{Code: "PRJ-1", Name: "P", BusinessUnit: "BU-1"}},
				BusinessUnits: []BusinessUnit{{Code: "BU-1", Name: "BU"}},
				Apps:          []App{{Code: "APP-1", Name: "A", Project: "PRJ-MISSING", CostCenter: "CC-1", Environments: []Environment{{Code: "ENV-1", Name: "E"}}}},
			},
			errCode:  "taxonomy-unknown-reference",
			severity: "error",
		},
		{
			name: "duplicate ou link app",
			tax: Taxonomy{
				LZs:     []LZ{{Code: "LZ-1", Name: "L", CSP: "aws", Currency: "USD", ManagementAccount: "1"}},
				OULinks: []OULink{{LZ: "LZ-1", OU: "/a", App: "APP-1"}, {LZ: "LZ-1", OU: "/a", App: "APP-2"}},
				Apps: []App{
					{Code: "APP-1", Name: "A", Project: "PRJ-1", CostCenter: "CC-1", Environments: []Environment{{Code: "ENV-1", Name: "E"}}},
					{Code: "APP-2", Name: "B", Project: "PRJ-1", CostCenter: "CC-1", Environments: []Environment{{Code: "ENV-1", Name: "E"}}},
				},
				Projects:      []Project{{Code: "PRJ-1", Name: "P", BusinessUnit: "BU-1"}},
				BusinessUnits: []BusinessUnit{{Code: "BU-1", Name: "BU"}},
				CostCenters:   []CostCenter{{Code: "CC-1", Name: "C", BusinessUnit: "BU-1"}},
			},
			errCode:  "taxonomy-duplicate-ou-link",
			severity: "error",
		},
		{
			name: "missing required field",
			tax: Taxonomy{
				BusinessUnits: []BusinessUnit{{Code: "BU-1", Name: ""}},
			},
			errCode:  "taxonomy-missing-field",
			severity: "error",
		},
		{
			name: "environment not under app on account entry",
			tax: Taxonomy{
				BusinessUnits: []BusinessUnit{{Code: "BU-1", Name: "BU"}},
				CostCenters:   []CostCenter{{Code: "CC-1", Name: "C", BusinessUnit: "BU-1"}},
				Projects:      []Project{{Code: "PRJ-1", Name: "P", BusinessUnit: "BU-1"}},
				Apps:          []App{{Code: "APP-1", Name: "A", Project: "PRJ-1", CostCenter: "CC-1", Environments: []Environment{{Code: "ENV-PROD", Name: "Prod"}}}},
				LZs:           []LZ{{Code: "LZ-1", Name: "L", CSP: "aws", Currency: "USD", ManagementAccount: "1"}},
				Accounts:      []AccountLink{{Account: "200000000001", Apps: []string{"APP-1"}, Environment: "ENV-DEV"}},
			},
			errCode:  "taxonomy-environment-not-in-app",
			severity: "error",
		},
		{
			name: "unused business unit warning",
			tax: Taxonomy{
				BusinessUnits: []BusinessUnit{{Code: "BU-1", Name: "BU"}, {Code: "BU-ORPHAN", Name: "Orphan"}},
				CostCenters:   []CostCenter{{Code: "CC-1", Name: "C", BusinessUnit: "BU-1"}},
				Projects:      []Project{{Code: "PRJ-1", Name: "P", BusinessUnit: "BU-1"}},
				Apps:          []App{{Code: "APP-1", Name: "A", Project: "PRJ-1", CostCenter: "CC-1", Environments: []Environment{{Code: "ENV-1", Name: "E"}}}},
			},
			errCode:  "taxonomy-unused",
			severity: "warning",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			findings := ValidateTaxonomy(tc.tax)
			found := false
			for _, f := range findings {
				if f["code"] == tc.errCode && f["severity"] == tc.severity {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected %q severity %q in %v", tc.errCode, tc.severity, findings)
			}
		})
	}
}

func TestLoadTaxonomyUnknownKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	body := `business_units:
  - { code: BU-1, name: Retail, colour: red }
projects:
  - { code: PRJ-1, name: P, business_unit: BU-1 }
cost_centers:
  - { code: CC-1, name: C, business_unit: BU-1 }
apps:
  - { code: APP-1, name: A, project: PRJ-1, cost_center: CC-1, environments: [{ code: ENV-1, name: E }] }
lzs:
  - { code: LZ-1, name: L, csp: aws, currency: USD, management_account: "1" }
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadTaxonomy(path)
	if err == nil {
		t.Fatal("expected unknown field error")
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("error should name file: %v", err)
	}
}
