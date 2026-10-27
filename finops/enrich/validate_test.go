package enrich

import "testing"

func TestValidateErrors(t *testing.T) {
	cases := []struct {
		name    string
		tax     Taxonomy
		errCode string
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
			errCode: "taxonomy-duplicate-code",
		},
		{
			name: "reserved code",
			tax: Taxonomy{
				BusinessUnits: []BusinessUnit{{Code: "BU-UNALLOC", Name: "Bad"}},
			},
			errCode: "taxonomy-reserved-code",
		},
		{
			name: "unknown project reference",
			tax: Taxonomy{
				Projects: []Project{{Code: "PRJ-1", Name: "P", BusinessUnit: "BU-1"}},
				Apps:     []App{{Code: "APP-1", Name: "A", Project: "PRJ-MISSING", CostCenter: "CC-1", Environments: []Environment{{Code: "ENV-1", Name: "E"}}}},
			},
			errCode: "taxonomy-unknown-reference",
		},
		{
			name: "duplicate ou link app",
			tax: Taxonomy{
				LZs:     []LZ{{Code: "LZ-1", Name: "L", CSP: "aws", Currency: "USD", ManagementAccount: "1"}},
				OULinks: []OULink{{LZ: "LZ-1", OU: "/a", App: "APP-1"}, {LZ: "LZ-1", OU: "/a", App: "APP-2"}},
				Apps:    []App{{Code: "APP-1", Name: "A", Project: "P", CostCenter: "C", Environments: []Environment{{Code: "ENV-1", Name: "E"}}}, {Code: "APP-2", Name: "B", Project: "P", CostCenter: "C", Environments: []Environment{{Code: "ENV-1", Name: "E"}}}},
			},
			errCode: "taxonomy-duplicate-ou-link",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			findings := ValidateTaxonomy(tc.tax)
			found := false
			for _, f := range findings {
				if f["code"] == tc.errCode && f["severity"] == "error" {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected error code %q in %v", tc.errCode, findings)
			}
		})
	}
}
