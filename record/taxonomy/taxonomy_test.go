package taxonomy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// full uses every key of the file (D-060), including the keys that are read and not used (D-057).
const full = `
tenant: { code: testtnt, name: Test tenant, urn: "urn:tnt:testtnt", alias: tt }
business_units:
  - code: retail
    name: Retail
    urn: urn:tnt:testtnt:bzu:retail
    alias: ret
    cost_centers:
      - code: ecom
        name: E-commerce
        urn: urn:tnt:testtnt:bzu:retail:cct:ecom
        alias: ec
        owner_teams:
          - { code: web-team, name: Web team, urn: "urn:tnt:testtnt:ont:web-team", alias: web }
projects:
  - code: PRJ_Shop
    name: Online shop
    urn: urn:tnt:testtnt:prj:prj_shop
    alias: shop
    ref_urn_bu: urn:tnt:testtnt:bzu:retail
    apps:
      - code: chk
        name: Checkout
        urn: urn:tnt:testtnt:prj:prj_shop:app:chk
        alias: checkout
        ref_urn_cct: urn:tnt:testtnt:bzu:retail:cct:ecom
        environments:
          - { code: prod, name: Production, urn: "urn:tnt:testtnt:prj:prj_shop:app:chk:env:prod", alias: p, ref_urn_ont: "urn:tnt:testtnt:ont:web-team" }
          - { code: dev, name: Development }
lzs:
  - code: main
    name: Main
    urn: urn:tnt:testtnt:lzc:main
    alias: lz
    csp: aws
    currency: USD
    management_account: "111111111111"
    ref_id: "111111111111"
    default: { app: chk, environment: dev }
    ref_urn_prjs: [urn:tnt:testtnt:prj:prj_shop]
    ou_paths:
      - code: checkout
        name: Checkout
        urn: urn:tnt:testtnt:lzc:main:org:checkout
        alias: co
        path: /Workloads/Checkout
        ref_id: ou-1a2b-3c4d5e6f
        ref_urn_apps: [urn:tnt:testtnt:prj:prj_shop:app:chk]
        ref_urn_env: urn:tnt:testtnt:prj:prj_shop:app:chk:env:prod
        accounts:
          - code: shop-prod
            name: shop-prod
            urn: urn:tnt:testtnt:lzc:main:org:checkout:acc:shop-prod
            alias: sp
            ref_id: "200000000002"
            ref_urn_apps: [urn:tnt:testtnt:prj:prj_shop:app:chk]
            ref_urn_env: urn:tnt:testtnt:prj:prj_shop:app:chk:env:prod
            contact: owner@example.com
defaults: { app: chk, environment: dev }
tags: { app: AppTag, project: ProjectTag }
`

func mustParse(t *testing.T, body string) Taxonomy {
	t.Helper()
	tax, err := Parse([]byte(body))
	if err != nil {
		t.Fatalf("Parse: %v\n%s", err, body)
	}
	return tax
}

func TestParseEveryKey(t *testing.T) {
	tax := mustParse(t, full)
	checks := []struct {
		name      string
		got, want string
	}{
		{"tenant code", tax.Tenant.Code, "testtnt"},
		{"tenant name", tax.Tenant.Name, "Test tenant"},
		{"tenant urn", tax.Tenant.URN, "urn:tnt:testtnt"},
		{"tenant alias", tax.Tenant.Alias, "tt"},
		{"bu code", tax.BusinessUnits[0].Code, "retail"},
		{"bu name", tax.BusinessUnits[0].Name, "Retail"},
		{"bu urn", tax.BusinessUnits[0].URN, "urn:tnt:testtnt:bzu:retail"},
		{"bu alias", tax.BusinessUnits[0].Alias, "ret"},
		{"cc code", tax.BusinessUnits[0].CostCenters[0].Code, "ecom"},
		{"cc alias", tax.BusinessUnits[0].CostCenters[0].Alias, "ec"},
		{"project code kept as written", tax.Projects[0].Code, "PRJ_Shop"},
		{"project ref_urn_bu", tax.Projects[0].RefURNBU, "urn:tnt:testtnt:bzu:retail"},
		{"app code", tax.Projects[0].Apps[0].Code, "chk"},
		{"app ref_urn_cct", tax.Projects[0].Apps[0].RefURNCCT, "urn:tnt:testtnt:bzu:retail:cct:ecom"},
		{"env code", tax.Projects[0].Apps[0].Environments[0].Code, "prod"},
		{"env alias", tax.Projects[0].Apps[0].Environments[0].Alias, "p"},
		{"lz code", tax.LZs[0].Code, "main"},
		{"lz csp", tax.LZs[0].CSP, "aws"},
		{"lz currency", tax.LZs[0].Currency, "USD"},
		{"lz management_account", tax.LZs[0].ManagementAccount, "111111111111"},
		{"lz ref_id", tax.LZs[0].RefID, "111111111111"},
		{"lz default app", tax.LZs[0].Default.App, "chk"},
		{"lz default environment", tax.LZs[0].Default.Environment, "dev"},
		{"ou code", tax.LZs[0].OUPaths[0].Code, "checkout"},
		{"ou path", tax.LZs[0].OUPaths[0].Path, "/Workloads/Checkout"},
		{"ou ref_id", tax.LZs[0].OUPaths[0].RefID, "ou-1a2b-3c4d5e6f"},
		{"ou ref_urn_apps", tax.LZs[0].OUPaths[0].RefURNApps[0], "urn:tnt:testtnt:prj:prj_shop:app:chk"},
		{"ou ref_urn_env", tax.LZs[0].OUPaths[0].RefURNEnv, "urn:tnt:testtnt:prj:prj_shop:app:chk:env:prod"},
		{"account code", tax.LZs[0].OUPaths[0].Accounts[0].Code, "shop-prod"},
		{"account ref_id", tax.LZs[0].OUPaths[0].Accounts[0].RefID, "200000000002"},
		{"account alias", tax.LZs[0].OUPaths[0].Accounts[0].Alias, "sp"},
		{"account ref_urn_apps", tax.LZs[0].OUPaths[0].Accounts[0].RefURNApps[0], "urn:tnt:testtnt:prj:prj_shop:app:chk"},
		{"account ref_urn_env", tax.LZs[0].OUPaths[0].Accounts[0].RefURNEnv, "urn:tnt:testtnt:prj:prj_shop:app:chk:env:prod"},
		{"defaults app", tax.Defaults.App, "chk"},
		{"defaults environment", tax.Defaults.Environment, "dev"},
		{"tags app", tax.Tags.App, "AppTag"},
		{"tags project", tax.Tags.Project, "ProjectTag"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}
	if got := len(tax.Projects[0].Apps[0].Environments); got != 2 {
		t.Errorf("%d environments, want 2", got)
	}
}

func TestParseReadNotUsedKeys(t *testing.T) {
	tax := mustParse(t, full)
	if got := tax.BusinessUnits[0].CostCenters[0].OwnerTeams; len(got) != 1 || got[0].Code != "web-team" {
		t.Errorf("owner_teams = %+v", got)
	}
	if got := tax.Projects[0].Apps[0].Environments[0].RefURNONT; got != "urn:tnt:testtnt:ont:web-team" {
		t.Errorf("ref_urn_ont = %q", got)
	}
	if got := tax.LZs[0].RefURNPrjs; len(got) != 1 || got[0] != "urn:tnt:testtnt:prj:prj_shop" {
		t.Errorf("ref_urn_prjs = %v", got)
	}
	if got := tax.LZs[0].OUPaths[0].Accounts[0].Contact; got != "owner@example.com" {
		t.Errorf("contact = %q", got)
	}
}

func TestDefaultTagKeys(t *testing.T) {
	app, project := mustParse(t, "tenant: { code: testtnt, name: T }\n").TagKeys()
	if app != "App" || project != "Project" {
		t.Fatalf("defaults = %q %q, want App Project", app, project)
	}
	app, project = mustParse(t, full).TagKeys()
	if app != "AppTag" || project != "ProjectTag" {
		t.Fatalf("set = %q %q", app, project)
	}
}

func TestParseUnknownKeys(t *testing.T) {
	// One unknown key added at each depth of a file that is otherwise valid.
	cases := []struct{ name, old, new string }{
		{"top level", "tags: { app: AppTag, project: ProjectTag }", "tags: { app: AppTag, project: ProjectTag }\ncolour: red"},
		{"tenant", "tenant: { code: testtnt,", "tenant: { colour: red, code: testtnt,"},
		{"business unit", "    name: Retail\n", "    name: Retail\n    colour: red\n"},
		{"cost center", "        name: E-commerce\n", "        name: E-commerce\n        colour: red\n"},
		{"owner team", "name: Web team,", "name: Web team, colour: red,"},
		{"project", "    ref_urn_bu: urn:tnt:testtnt:bzu:retail\n", "    ref_urn_bu: urn:tnt:testtnt:bzu:retail\n    colour: red\n"},
		{"app", "        name: Checkout\n", "        name: Checkout\n        colour: red\n"},
		{"environment", "{ code: dev, name: Development }", "{ code: dev, name: Development, colour: red }"},
		{"lz", "    csp: aws\n", "    csp: aws\n    colour: red\n"},
		{"ou path", "        path: /Workloads/Checkout\n", "        path: /Workloads/Checkout\n        colour: red\n"},
		{"account", "            contact: owner@example.com\n", "            contact: owner@example.com\n            colour: red\n"},
		{"defaults", "defaults: { app: chk, environment: dev }", "defaults: { app: chk, environment: dev, colour: red }"},
		{"the key of the old flat ou link", "        path: /Workloads/Checkout\n", "        path: /Workloads/Checkout\n        environment: prod\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(full, tc.old) {
				t.Fatalf("fixture text %q not found", tc.old)
			}
			body := strings.Replace(full, tc.old, tc.new, 1)
			_, err := Parse([]byte(body))
			if err == nil {
				t.Fatal("expected an error for the unknown key")
			}
			if !strings.Contains(err.Error(), "colour") && !strings.Contains(err.Error(), "environment") {
				t.Fatalf("error %q should name the key", err)
			}
		})
	}
}

func TestParseFlatLayoutNamesTheKey(t *testing.T) {
	base := "tenant: { code: testtnt, name: T }\n"
	flat := map[string]string{
		"cost_centers": "cost_centers:\n  - { code: ecom, name: E, business_unit: retail }\n",
		"apps":         "apps:\n  - { code: chk, name: C, project: shop, cost_center: ecom }\n",
		"ou_links":     "ou_links:\n  - { lz: main, ou: /a, app: chk }\n",
		"accounts":     "accounts:\n  - { account: \"1\", apps: [chk] }\n",
		"owner_teams":  "owner_teams:\n  - { code: web-team, name: Web }\n",
	}
	for key, text := range flat {
		_, err := Parse([]byte(base + text))
		if err == nil {
			t.Errorf("%s: expected an error", key)
			continue
		}
		if !strings.Contains(err.Error(), `"`+key+`"`) || !strings.Contains(err.Error(), "flat layout") {
			t.Errorf("%s: error %q should name the key and the flat layout", key, err)
		}
	}
}

func TestLZsAndLZConns(t *testing.T) {
	lz := "  - { code: main, name: Main, csp: aws, currency: USD, management_account: \"1\" }\n"
	base := "tenant: { code: testtnt, name: T }\n"
	for _, key := range []string{"lzs", "lz_conns"} {
		tax := mustParse(t, base+key+":\n"+lz)
		if len(tax.LZs) != 1 || tax.LZs[0].Code != "main" {
			t.Errorf("%s: LZs = %+v", key, tax.LZs)
		}
	}
	_, err := Parse([]byte(base + "lzs:\n" + lz + "lz_conns:\n" + lz))
	if err == nil || !strings.Contains(err.Error(), "lzs") || !strings.Contains(err.Error(), "lz_conns") {
		t.Fatalf("error = %v, want one naming both keys", err)
	}
	if tax := mustParse(t, base); len(tax.LZs) != 0 {
		t.Errorf("no LZ key: LZs = %+v", tax.LZs)
	}
}

func TestParseRejects(t *testing.T) {
	cases := []struct{ name, body, want string }{
		{"empty", "", "empty"},
		{"a list", "- a\n- b\n", ""},
		{"a scalar tenant", "tenant: testtnt\n", "mapping"},
		{"not yaml", "tenant: [\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want one containing %q", err, tc.want)
			}
		})
	}
}

func TestLoadNamesTheFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.yaml")
	if _, err := Load(missing); err == nil || !strings.Contains(err.Error(), missing) {
		t.Fatalf("error = %v, want one naming %s", err, missing)
	}
	bad := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(bad, []byte("tenant: { code: testtnt, name: T, colour: red }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(bad); err == nil || !strings.Contains(err.Error(), bad) || !strings.Contains(err.Error(), "colour") {
		t.Fatalf("error = %v, want one naming the file and the key", err)
	}
	good := filepath.Join(t.TempDir(), "good.yaml")
	if err := os.WriteFile(good, []byte(full), 0o644); err != nil {
		t.Fatal(err)
	}
	if tax, err := Load(good); err != nil || tax.Tenant.Code != "testtnt" {
		t.Fatalf("Load = %+v, %v", tax.Tenant, err)
	}
}

func TestUnmarshalInsideAnotherDocument(t *testing.T) {
	var doc struct {
		Name     string   `yaml:"name"`
		Taxonomy Taxonomy `yaml:"taxonomy"`
	}
	body := "name: a case\ntaxonomy:\n  tenant: { code: testtnt, name: T }\n  lzs:\n    - { code: main, name: Main, csp: aws, currency: USD, management_account: \"1\" }\n"
	if err := yaml.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Taxonomy.Tenant.Code != "testtnt" || len(doc.Taxonomy.LZs) != 1 {
		t.Fatalf("taxonomy = %+v", doc.Taxonomy)
	}
	bad := "name: a case\ntaxonomy:\n  tenant: { code: testtnt, name: T, colour: red }\n"
	if err := yaml.Unmarshal([]byte(bad), &doc); err == nil || !strings.Contains(err.Error(), "colour") {
		t.Fatalf("error = %v, want the unknown key", err)
	}
}

func TestURNsAreBuiltFromCodesLowercased(t *testing.T) {
	tax := mustParse(t, full)
	cases := []struct{ name, got, want string }{
		{"bu", tax.BUURN("Retail").String(), "urn:tnt:testtnt:bzu:retail"},
		{"cc under its own bu", tax.CCURN("Retail", "Ecom").String(), "urn:tnt:testtnt:bzu:retail:cct:ecom"},
		{"project", tax.ProjectURN("PRJ_Shop").String(), "urn:tnt:testtnt:prj:prj_shop"},
		{"app", tax.AppURN("PRJ_Shop", "Chk").String(), "urn:tnt:testtnt:prj:prj_shop:app:chk"},
		{"env", tax.EnvURN("PRJ_Shop", "Chk", "Prod").String(), "urn:tnt:testtnt:prj:prj_shop:app:chk:env:prod"},
		{"lz", tax.LZURN("Main").String(), "urn:tnt:testtnt:lzc:main"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}
}

func TestIndexFindsReferencesInAnyCase(t *testing.T) {
	tax := mustParse(t, full)
	ix := tax.Index()
	if bu, ok := ix.LookupBU("URN:TNT:TESTTNT:BZU:Retail"); !ok || bu.Code != "retail" {
		t.Errorf("LookupBU = %v, %v", bu, ok)
	}
	if cc, ok := ix.LookupCC("urn:tnt:testtnt:bzu:retail:cct:ecom"); !ok || cc.CC.Code != "ecom" || cc.BU.Code != "retail" {
		t.Errorf("LookupCC = %+v, %v", cc, ok)
	}
	if a, ok := ix.LookupApp("urn:tnt:testtnt:prj:prj_shop:app:chk"); !ok || a.App.Code != "chk" || a.Project.Code != "PRJ_Shop" {
		t.Errorf("LookupApp = %+v, %v (the project's code keeps its case)", a, ok)
	}
	if e, ok := ix.LookupEnv("urn:tnt:testtnt:prj:prj_shop:app:chk:env:prod"); !ok || e.Env.Code != "prod" || e.App.Code != "chk" {
		t.Errorf("LookupEnv = %+v, %v", e, ok)
	}
	if p, ok := ix.LookupProject("urn:tnt:testtnt:prj:prj_shop"); !ok || p.Code != "PRJ_Shop" {
		t.Errorf("LookupProject = %v, %v", p, ok)
	}
	if a, ok := ix.AppByCode("CHK"); !ok || a.App.Code != "chk" {
		t.Errorf("AppByCode = %+v, %v", a, ok)
	}
	for _, miss := range []string{"", "urn:tnt:testtnt:prj:prj_shop:app:nope", "urn:tnt:othertnt:prj:prj_shop:app:chk"} {
		if _, ok := ix.LookupApp(miss); ok {
			t.Errorf("LookupApp(%q) found a node", miss)
		}
	}
	if !tax.Projects[0].Apps[0].HasEnv("dev") || tax.Projects[0].Apps[0].HasEnv("qa-x") {
		t.Error("HasEnv is wrong")
	}
}

func TestLZManagement(t *testing.T) {
	if got := (LZ{ManagementAccount: "1", RefID: "2"}).Management(); got != "1" {
		t.Errorf("management_account first: %q", got)
	}
	if got := (LZ{RefID: "2"}).Management(); got != "2" {
		t.Errorf("ref_id when there is no management_account: %q", got)
	}
}
