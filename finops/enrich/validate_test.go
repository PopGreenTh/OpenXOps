package enrich

import (
	"github.com/PopGreenTh/OpenXOps/record"
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
				Projects:      []Project{{Code: "prj-1", Name: "P", BusinessUnit: "bu-1"}},
				BusinessUnits: []BusinessUnit{{Code: "bu-1", Name: "BU"}},
				CostCenters:   []CostCenter{{Code: "cc-1", Name: "C", BusinessUnit: "bu-1"}},
				Apps: []App{
					{Code: "app-a", Name: "A", Project: "prj-1", CostCenter: "cc-1", Environments: []Environment{{Code: "env-1", Name: "One"}}},
					{Code: "app-a", Name: "B", Project: "prj-1", CostCenter: "cc-1", Environments: []Environment{{Code: "env-1", Name: "One"}}},
				},
			},
			errCode:  "taxonomy-duplicate-code",
			severity: "error",
		},
		{
			name: "reserved code",
			tax: Taxonomy{
				BusinessUnits: []BusinessUnit{{Code: "unalloc", Name: "Bad"}},
			},
			errCode:  "taxonomy-reserved-code",
			severity: "error",
		},
		{
			name: "reserved app id",
			tax: Taxonomy{
				Apps: []App{{Code: "invalid", Name: "Bad", Project: "p", CostCenter: "c"}},
			},
			errCode:  "taxonomy-reserved-code",
			severity: "error",
		},
		{
			name: "reserved environment id",
			tax: Taxonomy{
				Apps: []App{{Code: "a", Name: "A", Project: "p", CostCenter: "c", Environments: []Environment{{Code: "unspec", Name: "Bad"}}}},
			},
			errCode:  "taxonomy-reserved-code",
			severity: "error",
		},
		{
			name: "reserved environment id on an ou link",
			tax: Taxonomy{
				LZs:     []LZ{{Code: "main", Name: "L", CSP: "aws", Currency: "USD", ManagementAccount: "1"}},
				OULinks: []OULink{{LZ: "main", OU: "/a", Environment: "unalloc"}},
			},
			errCode:  "taxonomy-reserved-code",
			severity: "error",
		},
		{
			name: "unknown project reference",
			tax: Taxonomy{
				Projects:      []Project{{Code: "prj-1", Name: "P", BusinessUnit: "bu-1"}},
				BusinessUnits: []BusinessUnit{{Code: "bu-1", Name: "BU"}},
				Apps:          []App{{Code: "app-1", Name: "A", Project: "prj-missing", CostCenter: "cc-1", Environments: []Environment{{Code: "env-1", Name: "E"}}}},
			},
			errCode:  "taxonomy-unknown-reference",
			severity: "error",
		},
		{
			name: "duplicate ou link app",
			tax: Taxonomy{
				LZs:     []LZ{{Code: "lz-1", Name: "L", CSP: "aws", Currency: "USD", ManagementAccount: "1"}},
				OULinks: []OULink{{LZ: "lz-1", OU: "/a", App: "app-1"}, {LZ: "lz-1", OU: "/a", App: "app-2"}},
				Apps: []App{
					{Code: "app-1", Name: "A", Project: "prj-1", CostCenter: "cc-1", Environments: []Environment{{Code: "env-1", Name: "E"}}},
					{Code: "app-2", Name: "B", Project: "prj-1", CostCenter: "cc-1", Environments: []Environment{{Code: "env-1", Name: "E"}}},
				},
				Projects:      []Project{{Code: "prj-1", Name: "P", BusinessUnit: "bu-1"}},
				BusinessUnits: []BusinessUnit{{Code: "bu-1", Name: "BU"}},
				CostCenters:   []CostCenter{{Code: "cc-1", Name: "C", BusinessUnit: "bu-1"}},
			},
			errCode:  "taxonomy-duplicate-ou-link",
			severity: "error",
		},
		{
			name: "missing required field",
			tax: Taxonomy{
				BusinessUnits: []BusinessUnit{{Code: "bu-1", Name: ""}},
			},
			errCode:  "taxonomy-missing-field",
			severity: "error",
		},
		{
			name: "environment not under app on account entry",
			tax: Taxonomy{
				BusinessUnits: []BusinessUnit{{Code: "bu-1", Name: "BU"}},
				CostCenters:   []CostCenter{{Code: "cc-1", Name: "C", BusinessUnit: "bu-1"}},
				Projects:      []Project{{Code: "prj-1", Name: "P", BusinessUnit: "bu-1"}},
				Apps:          []App{{Code: "app-1", Name: "A", Project: "prj-1", CostCenter: "cc-1", Environments: []Environment{{Code: "env-prod", Name: "Prod"}}}},
				LZs:           []LZ{{Code: "lz-1", Name: "L", CSP: "aws", Currency: "USD", ManagementAccount: "1"}},
				Accounts:      []AccountLink{{Account: "200000000001", Apps: []string{"app-1"}, Environment: "env-dev"}},
			},
			errCode:  "taxonomy-environment-not-in-app",
			severity: "error",
		},
		{
			name: "unused business unit warning",
			tax: Taxonomy{
				BusinessUnits: []BusinessUnit{{Code: "bu-1", Name: "BU"}, {Code: "bu-orphan", Name: "Orphan"}},
				CostCenters:   []CostCenter{{Code: "cc-1", Name: "C", BusinessUnit: "bu-1"}},
				Projects:      []Project{{Code: "prj-1", Name: "P", BusinessUnit: "bu-1"}},
				Apps:          []App{{Code: "app-1", Name: "A", Project: "prj-1", CostCenter: "cc-1", Environments: []Environment{{Code: "env-1", Name: "E"}}}},
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
  - { code: bu-1, name: Retail, colour: red }
projects:
  - { code: prj-1, name: P, business_unit: bu-1 }
cost_centers:
  - { code: cc-1, name: C, business_unit: bu-1 }
apps:
  - { code: app-1, name: A, project: prj-1, cost_center: cc-1, environments: [{ code: env-1, name: E }] }
lzs:
  - { code: lz-1, name: L, csp: aws, currency: USD, management_account: "1" }
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

func TestValidateIDRule(t *testing.T) {
	long := strings.Repeat("x", 51)
	cases := []struct {
		name string
		tax  Taxonomy
	}{
		{"business unit too short", Taxonomy{BusinessUnits: []BusinessUnit{{Code: "ab", Name: "Retail"}}}},
		{"cost center with a slash", Taxonomy{CostCenters: []CostCenter{{Code: "a/b", Name: "C", BusinessUnit: "bu"}}}},
		{"project with a space", Taxonomy{Projects: []Project{{Code: "my shop", Name: "P", BusinessUnit: "bu"}}}},
		{"app 51 characters", Taxonomy{Apps: []App{{Code: long, Name: "A", Project: "p", CostCenter: "c"}}}},
		{"environment too short", Taxonomy{Apps: []App{{Code: "app", Name: "A", Project: "p", CostCenter: "c", Environments: []Environment{{Code: "qa", Name: "P"}}}}}},
		{"lz with a colon", Taxonomy{LZs: []LZ{{Code: "main:1", Name: "L", CSP: "aws", Currency: "USD", ManagementAccount: "1"}}}},
		{"non-ASCII", Taxonomy{BusinessUnits: []BusinessUnit{{Code: "caf\u00e9", Name: "Retail"}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, f := range ValidateTaxonomy(tc.tax) {
				if f["code"] == "taxonomy-invalid-id" {
					if f["severity"] != "error" {
						t.Fatalf("severity = %v, want error", f["severity"])
					}
					got = append(got, strID(f["subject"]))
				}
			}
			if len(got) != 1 {
				t.Fatalf("taxonomy-invalid-id findings = %v, want exactly one", got)
			}
		})
	}
}

func TestValidateCodesThatPassTheRule(t *testing.T) {
	tax := Taxonomy{
		BusinessUnits: []BusinessUnit{{Code: "Retail", Name: "Retail"}, {Code: "b2c-1", Name: "B2C"}, {Code: "a.b", Name: "AB"}},
		Projects:      []Project{{Code: "PRJ_Shop", Name: "P", BusinessUnit: "Retail"}, {Code: "x@y=z+1", Name: "X", BusinessUnit: "Retail"}},
		Apps:          []App{{Code: strings.Repeat("a", 50), Name: "A", Project: "PRJ_Shop", CostCenter: "c"}},
		LZs:           []LZ{{Code: "Main", Name: "L", CSP: "aws", Currency: "USD", ManagementAccount: "1"}},
	}
	for _, f := range ValidateTaxonomy(tax) {
		if f["code"] == "taxonomy-invalid-id" || f["code"] == "taxonomy-reserved-code" || f["code"] == "taxonomy-duplicate-code" {
			t.Fatalf("unexpected finding %v", f)
		}
	}
}

func TestValidateReservedIDsInAnyCase(t *testing.T) {
	for _, code := range []string{"unalloc", "UNALLOC", "Unspec", "uNsPeC", "invalid", "INVALID"} {
		findings := ValidateTaxonomy(Taxonomy{BusinessUnits: []BusinessUnit{{Code: code, Name: "Bad"}}})
		var reserved, invalid int
		for _, f := range findings {
			switch f["code"] {
			case "taxonomy-reserved-code":
				reserved++
			case "taxonomy-invalid-id":
				invalid++
			}
		}
		if reserved != 1 || invalid != 0 {
			t.Errorf("code %q: %d taxonomy-reserved-code and %d taxonomy-invalid-id findings, want 1 and 0", code, reserved, invalid)
		}
	}
	// A longer code that merely contains one is fine.
	for _, f := range ValidateTaxonomy(Taxonomy{BusinessUnits: []BusinessUnit{{Code: "unalloc-eu", Name: "EU"}}}) {
		if f["code"] == "taxonomy-reserved-code" {
			t.Fatalf("unalloc-eu is not reserved: %v", f)
		}
	}
}

func TestValidateDuplicateCodesDifferingInCase(t *testing.T) {
	cases := []struct {
		name string
		tax  Taxonomy
		want int
	}{
		{"projects Shop and shop", Taxonomy{Projects: []Project{{Code: "shop", Name: "A", BusinessUnit: "bu"}, {Code: "Shop", Name: "B", BusinessUnit: "bu"}}}, 1},
		{"business units", Taxonomy{BusinessUnits: []BusinessUnit{{Code: "retail", Name: "A"}, {Code: "RETAIL", Name: "B"}}}, 1},
		{"cost centers", Taxonomy{CostCenters: []CostCenter{{Code: "ecom", Name: "A", BusinessUnit: "bu"}, {Code: "Ecom", Name: "B", BusinessUnit: "bu"}}}, 1},
		{"apps, exact duplicate", Taxonomy{Apps: []App{{Code: "chk", Name: "A", Project: "p", CostCenter: "c"}, {Code: "chk", Name: "B", Project: "p", CostCenter: "c"}}}, 1},
		{"lzs", Taxonomy{LZs: []LZ{{Code: "main", Name: "A", CSP: "aws", Currency: "USD", ManagementAccount: "1"}, {Code: "MAIN", Name: "B", CSP: "aws", Currency: "USD", ManagementAccount: "2"}}}, 1},
		{"environments under one app", Taxonomy{Apps: []App{{Code: "chk", Name: "A", Project: "p", CostCenter: "c", Environments: []Environment{{Code: "prod", Name: "A"}, {Code: "Prod", Name: "B"}}}}}, 1},
		{"same environment under two apps", Taxonomy{Apps: []App{
			{Code: "chk", Name: "A", Project: "p", CostCenter: "c", Environments: []Environment{{Code: "prod", Name: "A"}}},
			{Code: "pay", Name: "B", Project: "p", CostCenter: "c", Environments: []Environment{{Code: "prod", Name: "A"}}}}}, 0},
		{"same code in two different lists", Taxonomy{
			Projects: []Project{{Code: "pay", Name: "P", BusinessUnit: "bu"}},
			Apps:     []App{{Code: "pay", Name: "A", Project: "pay", CostCenter: "c"}}}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := 0
			for _, f := range ValidateTaxonomy(tc.tax) {
				if f["code"] == "taxonomy-duplicate-code" {
					if f["severity"] != "error" {
						t.Fatalf("severity = %v", f["severity"])
					}
					n++
				}
			}
			if n != tc.want {
				t.Fatalf("%d taxonomy-duplicate-code findings, want %d", n, tc.want)
			}
		})
	}
}

func TestValidateLZCount(t *testing.T) {
	lz := func(code string) LZ {
		return LZ{Code: code, Name: "L", CSP: "aws", Currency: "USD", ManagementAccount: "1"}
	}
	cases := []struct {
		name    string
		lzs     []LZ
		wantErr bool
	}{
		{"none", nil, true},
		{"one", []LZ{lz("main")}, false},
		{"two", []LZ{lz("main"), lz("other")}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var found []record.Record
			for _, f := range ValidateTaxonomy(Taxonomy{LZs: tc.lzs}) {
				if f["code"] == "taxonomy-lz-count" {
					found = append(found, f)
				}
			}
			if !tc.wantErr {
				if len(found) != 0 {
					t.Fatalf("unexpected %v", found)
				}
				return
			}
			if len(found) != 1 || found[0]["severity"] != "error" || found[0]["subject"] != "lzs" {
				t.Fatalf("want one taxonomy-lz-count error with subject lzs, got %v", found)
			}
		})
	}
}

func TestLoadNestedTaxonomy(t *testing.T) {
	tax, err := LoadTaxonomy(filepath.Join("testdata", "nested-taxonomy.yaml"))
	if err != nil {
		t.Fatalf("LoadTaxonomy failed: %v", err)
	}
	if tax.Tenant != "nesttnt" {
		t.Errorf("want tenant nesttnt, got %q", tax.Tenant)
	}
	if len(tax.BusinessUnits) != 2 {
		t.Errorf("want 2 BUs, got %d", len(tax.BusinessUnits))
	}
	if len(tax.CostCenters) != 2 {
		t.Errorf("want 2 CCs, got %d", len(tax.CostCenters))
	}
	if len(tax.Projects) != 2 {
		t.Errorf("want 2 Projects, got %d", len(tax.Projects))
	}
	if len(tax.Apps) != 3 {
		t.Errorf("want 3 Apps, got %d", len(tax.Apps))
	}
	if len(tax.LZs) != 1 {
		t.Errorf("want 1 LZ, got %d", len(tax.LZs))
	}
	findings := ValidateTaxonomy(tax)
	for _, f := range findings {
		if f["severity"] == "error" {
			t.Errorf("unexpected validation error: %v", f)
		}
	}
}

func TestValidateAliasUniqueness(t *testing.T) {
	baseTax := func() Taxonomy {
		return Taxonomy{
			Tenant: "testtnt",
			BusinessUnits: []BusinessUnit{
				{Code: "bu-a", Name: "BU A", Alias: "infra"},
				{Code: "bu-b", Name: "BU B", Alias: "platform"},
			},
			CostCenters: []CostCenter{
				{Code: "cc-a", Name: "CC A", BusinessUnit: "bu-a", Alias: "cc-1"},
				{Code: "cc-b", Name: "CC B", BusinessUnit: "bu-b", Alias: "cc-2"},
			},
			OwnerTeams: []OwnerTeam{
				{Code: "team-a", Name: "Team A", CostCenter: "cc-a", Alias: "alpha"},
				{Code: "team-b", Name: "Team B", CostCenter: "cc-b", Alias: "bravo"},
			},
			Projects: []Project{
				{Code: "prj-a", Name: "Prj A", BusinessUnit: "bu-a", Alias: "proj-1"},
				{Code: "prj-b", Name: "Prj B", BusinessUnit: "bu-b", Alias: "proj-2"},
			},
			Apps: []App{
				{
					Code: "app-a", Name: "App A", Project: "prj-a", CostCenter: "cc-a", Alias: "shop",
					Environments: []Environment{
						{Code: "prod", Name: "Prod", Alias: "p"},
						{Code: "stage", Name: "Stage", Alias: "s"},
					},
				},
				{
					Code: "app-b", Name: "App B", Project: "prj-b", CostCenter: "cc-b", Alias: "hub",
					Environments: []Environment{
						{Code: "dev", Name: "Dev", Alias: "d"},
					},
				},
			},
			LZs: []LZ{
				{Code: "lz-1", Name: "Main LZ", CSP: "aws", Currency: "USD", ManagementAccount: "111111111111", Alias: "aws-lz"},
			},
			OUs: []OU{
				{Code: "ou-1", Name: "Workloads", Path: "/workloads", Alias: "wkld"},
				{Code: "ou-2", Name: "Shared", Path: "/shared", Alias: "shrd"},
			},
			Accounts: []AccountLink{
				{Account: "111111111111", Apps: []string{"app-a"}, Environment: "prod", Alias: "acct-main"},
				{Account: "222222222222", Apps: []string{"app-b"}, Environment: "dev", Alias: "acct-sub"},
			},
		}
	}

	findings := ValidateTaxonomy(baseTax())
	for _, f := range findings {
		if f["code"] == "taxonomy-duplicate-alias" {
			t.Errorf("unexpected duplicate alias error on valid taxonomy: %v", f)
		}
	}

	taxBU := baseTax()
	taxBU.BusinessUnits[1].Alias = "infra"
	if f := findCode(ValidateTaxonomy(taxBU), "taxonomy-duplicate-alias"); len(f) == 0 {
		t.Errorf("want taxonomy-duplicate-alias for duplicate BU alias, got none")
	}

	taxApp := baseTax()
	taxApp.Apps[1].Alias = "shop"
	if f := findCode(ValidateTaxonomy(taxApp), "taxonomy-duplicate-alias"); len(f) == 0 {
		t.Errorf("want taxonomy-duplicate-alias for duplicate App alias, got none")
	}

	taxEnv := baseTax()
	taxEnv.Apps[1].Environments[0].Alias = "p"
	if f := findCode(ValidateTaxonomy(taxEnv), "taxonomy-duplicate-alias"); len(f) == 0 {
		t.Errorf("want taxonomy-duplicate-alias for duplicate Environment alias across tenant, got none")
	}

	taxOnt := baseTax()
	taxOnt.OwnerTeams[1].Alias = "alpha"
	if f := findCode(ValidateTaxonomy(taxOnt), "taxonomy-duplicate-alias"); len(f) == 0 {
		t.Errorf("want taxonomy-duplicate-alias for duplicate OwnerTeam alias, got none")
	}

	taxOU := baseTax()
	taxOU.OUs[1].Alias = "wkld"
	if f := findCode(ValidateTaxonomy(taxOU), "taxonomy-duplicate-alias"); len(f) == 0 {
		t.Errorf("want taxonomy-duplicate-alias for duplicate OU alias, got none")
	}

	taxAcc := baseTax()
	taxAcc.Accounts[1].Alias = "acct-main"
	if f := findCode(ValidateTaxonomy(taxAcc), "taxonomy-duplicate-alias"); len(f) == 0 {
		t.Errorf("want taxonomy-duplicate-alias for duplicate Account alias, got none")
	}

	taxDiff := baseTax()
	taxDiff.BusinessUnits[0].Alias = "common-alias"
	taxDiff.Apps[0].Alias = "common-alias"
	taxDiff.Projects[0].Alias = "common-alias"
	if f := findCode(ValidateTaxonomy(taxDiff), "taxonomy-duplicate-alias"); len(f) != 0 {
		t.Errorf("identical alias on different types should be allowed, got: %v", f)
	}
}

func findCode(findings []record.Record, code string) []record.Record {
	var out []record.Record
	for _, f := range findings {
		if f["code"] == code {
			out = append(out, f)
		}
	}
	return out
}

func TestCodeRulesApplyToCodesReadThroughTheNestedReader(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "nested-taxonomy.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	load := func(name, body string) Taxonomy {
		path := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		tax, err := LoadTaxonomy(path)
		if err != nil {
			t.Fatal(err)
		}
		return tax
	}
	has := func(tax Taxonomy, code, subject string) bool {
		for _, f := range ValidateTaxonomy(tax) {
			if f["code"] == code && (subject == "" || f["subject"] == subject) {
				return true
			}
		}
		return false
	}

	// The nested reader takes a node's code from its URN segment, so that is what the rules see.
	// A business unit whose URN id is ab breaks the code rule.
	short := strings.Replace(string(data), "urn: urn:tnt:nesttnt:bzu:retail\n", "urn: urn:tnt:nesttnt:bzu:ab\n", 1)
	if short == string(data) {
		t.Fatal("the fixture's first business unit URN changed; update this test")
	}
	if tax := load("short.yaml", short); !has(tax, "taxonomy-invalid-id", "ab") {
		t.Errorf("want taxonomy-invalid-id for ab, got %v", ValidateTaxonomy(tax))
	}

	// A reserved id.
	reserved := strings.Replace(string(data), "urn: urn:tnt:nesttnt:bzu:retail\n", "urn: urn:tnt:nesttnt:bzu:unalloc\n", 1)
	if tax := load("reserved.yaml", reserved); !has(tax, "taxonomy-reserved-code", "unalloc") {
		t.Errorf("want taxonomy-reserved-code for unalloc, got %v", ValidateTaxonomy(tax))
	}

	// Two business units whose URN ids differ only in case.
	dup := strings.Replace(string(data), "urn: urn:tnt:nesttnt:bzu:digital\n", "urn: urn:tnt:nesttnt:bzu:RETAIL\n", 1)
	if tax := load("dup.yaml", dup); !has(tax, "taxonomy-duplicate-code", "") {
		t.Errorf("want taxonomy-duplicate-code for retail and RETAIL, got %v", ValidateTaxonomy(tax))
	}

	// The unchanged fixture stays clean.
	if tax := load("clean.yaml", string(data)); has(tax, "taxonomy-invalid-id", "") || has(tax, "taxonomy-reserved-code", "") || has(tax, "taxonomy-duplicate-code", "") {
		t.Errorf("the fixture should pass the code rules, got %v", ValidateTaxonomy(tax))
	}
}
