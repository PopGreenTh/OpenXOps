package enrich

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/PopGreenTh/OpenXOps/record"
	"github.com/PopGreenTh/OpenXOps/record/taxonomy"
)

// baseTaxonomy is a valid nested taxonomy: two Apps in two projects, one LZ with one OU entry and
// one account entry. The validate tests change one thing in it and look at the findings.
const baseTaxonomy = `tenant: { code: testtnt, name: Test tenant }
business_units:
  - code: retail
    name: Retail
    cost_centers:
      - { code: ecom, name: E-commerce }
  - code: digital
    name: Digital
    cost_centers:
      - { code: pay, name: Payments }
projects:
  - code: shop
    name: Shop
    ref_urn_bu: urn:tnt:testtnt:bzu:retail
    apps:
      - code: chk
        name: Checkout
        ref_urn_cct: urn:tnt:testtnt:bzu:retail:cct:ecom
        environments: [{ code: prod, name: Prod }, { code: dev, name: Dev }]
  - code: payments
    name: Payments
    ref_urn_bu: urn:tnt:testtnt:bzu:digital
    apps:
      - code: pay
        name: Pay
        ref_urn_cct: urn:tnt:testtnt:bzu:digital:cct:pay
        environments: [{ code: prod, name: Prod }, { code: dev, name: Dev }]
lzs:
  - code: main
    name: Main
    csp: aws
    currency: USD
    management_account: "111111111111"
    ou_paths:
      - code: checkout
        name: Checkout
        path: /Workloads/Checkout
        ref_urn_apps: [urn:tnt:testtnt:prj:shop:app:chk]
        ref_urn_env: urn:tnt:testtnt:prj:shop:app:chk:env:prod
        accounts:
          - code: chk-prod
            name: chk-prod
            ref_id: "200000000001"
            ref_urn_apps: [urn:tnt:testtnt:prj:shop:app:chk]
            ref_urn_env: urn:tnt:testtnt:prj:shop:app:chk:env:prod
defaults: { app: chk, environment: dev }
`

func parseTaxonomy(t *testing.T, body string) taxonomy.Taxonomy {
	t.Helper()
	tax, err := taxonomy.Parse([]byte(body))
	if err != nil {
		t.Fatalf("parse: %v\n%s", err, body)
	}
	return tax
}

// change returns baseTaxonomy with old replaced by new; old must be in it exactly once.
func change(t *testing.T, old, new string) string {
	t.Helper()
	if n := strings.Count(baseTaxonomy, old); n != 1 {
		t.Fatalf("%q is in baseTaxonomy %d times, want 1", old, n)
	}
	return strings.Replace(baseTaxonomy, old, new, 1)
}

func validateBody(t *testing.T, body string) []record.Record {
	t.Helper()
	return ValidateTaxonomy(parseTaxonomy(t, body))
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

// wantFinding fails unless findings hold exactly one finding with this code, severity and subject.
func wantFinding(t *testing.T, findings []record.Record, code, severity, subject string) {
	t.Helper()
	got := findCode(findings, code)
	if len(got) != 1 {
		t.Fatalf("%d %q findings, want 1: %v", len(got), code, findings)
	}
	if got[0]["severity"] != severity || got[0]["subject"] != subject {
		t.Fatalf("%q finding is %v %q, want %s %q", code, got[0]["severity"], got[0]["subject"], severity, subject)
	}
}

func TestBaseTaxonomyIsClean(t *testing.T) {
	if f := validateBody(t, baseTaxonomy); len(f) != 0 {
		t.Fatalf("findings = %v", f)
	}
}

func TestValidateURNMismatch(t *testing.T) {
	cases := []struct {
		name, old, new, subject string
	}{
		// The example of WO-0111-B: the code is CVD-Spoke, so the URN ends in cvd-spoke.
		{"app", "      - code: pay\n", "      - code: CVD-Spoke\n        urn: urn:tnt:testtnt:prj:payments:app:cvd-spk\n", "urn:tnt:testtnt:prj:payments:app:cvd-spk"},
		{"tenant", "tenant: { code: testtnt, name: Test tenant }", "tenant: { code: testtnt, name: T, urn: \"urn:tnt:other\" }", "urn:tnt:other"},
		{"business unit", "  - code: retail\n", "  - code: retail\n    urn: urn:tnt:testtnt:bzu:retial\n", "urn:tnt:testtnt:bzu:retial"},
		{"cost center", "{ code: ecom, name: E-commerce }", "{ code: ecom, name: E, urn: \"urn:tnt:testtnt:cct:ecom\" }", "urn:tnt:testtnt:cct:ecom"},
		{"project", "  - code: shop\n", "  - code: shop\n    urn: urn:tnt:testtnt:prj:shp\n", "urn:tnt:testtnt:prj:shp"},
		{"lz", "  - code: main\n", "  - code: main\n    urn: urn:tnt:testtnt:lzc:primary\n", "urn:tnt:testtnt:lzc:primary"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := change(t, tc.old, tc.new)
			wantFinding(t, validateBody(t, body), "taxonomy-urn-mismatch", "error", tc.subject)
		})
	}

	t.Run("a matching urn is fine, in any case of the code", func(t *testing.T) {
		body := change(t, "      - code: pay\n", "      - code: CVD-Spoke\n        urn: urn:tnt:testtnt:prj:payments:app:cvd-spoke\n")
		if f := findCode(validateBody(t, body), "taxonomy-urn-mismatch"); len(f) != 0 {
			t.Fatalf("unexpected %v", f)
		}
	})
	t.Run("a deeper node is checked against its own parents", func(t *testing.T) {
		body := change(t, "        environments: [{ code: prod, name: Prod }, { code: dev, name: Dev }]\n  - code: payments",
			"        environments:\n          - { code: prod, name: Prod, urn: \"urn:tnt:testtnt:prj:payments:app:chk:env:prod\" }\n  - code: payments")
		wantFinding(t, validateBody(t, body), "taxonomy-urn-mismatch", "error", "urn:tnt:testtnt:prj:payments:app:chk:env:prod")
	})
}

func TestValidateUnknownReference(t *testing.T) {
	cases := []struct {
		name, old, new, subject string
	}{
		{"project business unit", "ref_urn_bu: urn:tnt:testtnt:bzu:digital", "ref_urn_bu: urn:tnt:testtnt:bzu:nowhere", "urn:tnt:testtnt:bzu:nowhere"},
		{"app cost center", "ref_urn_cct: urn:tnt:testtnt:bzu:digital:cct:pay", "ref_urn_cct: urn:tnt:testtnt:bzu:retail:cct:pay", "urn:tnt:testtnt:bzu:retail:cct:pay"},
		{"ou apps", "        ref_urn_apps: [urn:tnt:testtnt:prj:shop:app:chk]\n        ref_urn_env", "        ref_urn_apps: [urn:tnt:testtnt:prj:shop:app:gone]\n        ref_urn_env", "urn:tnt:testtnt:prj:shop:app:gone"},
		{"ou env", "        ref_urn_env: urn:tnt:testtnt:prj:shop:app:chk:env:prod\n        accounts", "        ref_urn_env: urn:tnt:testtnt:prj:shop:app:chk:env:gone\n        accounts", "urn:tnt:testtnt:prj:shop:app:chk:env:gone"},
		{"account apps", "            ref_urn_apps: [urn:tnt:testtnt:prj:shop:app:chk]", "            ref_urn_apps: [urn:tnt:testtnt:prj:shop:app:chk, urn:tnt:testtnt:prj:other:app:chk]", "urn:tnt:testtnt:prj:other:app:chk"},
		{"account env", "            ref_urn_env: urn:tnt:testtnt:prj:shop:app:chk:env:prod", "            ref_urn_env: urn:tnt:testtnt:prj:shop:app:chk:env:stage", "urn:tnt:testtnt:prj:shop:app:chk:env:stage"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := change(t, tc.old, tc.new)
			wantFinding(t, validateBody(t, body), "taxonomy-unknown-reference", "error", tc.subject)
		})
	}

	t.Run("references match in any case", func(t *testing.T) {
		body := change(t, "ref_urn_bu: urn:tnt:testtnt:bzu:digital", "ref_urn_bu: urn:tnt:testtnt:bzu:DIGITAL")
		if f := findCode(validateBody(t, body), "taxonomy-unknown-reference"); len(f) != 0 {
			t.Fatalf("unexpected %v", f)
		}
	})
	t.Run("an environment of another app is not the app's", func(t *testing.T) {
		// pay defines prod, chk defines prod: pay's prod is not chk's.
		body := change(t, "            ref_urn_env: urn:tnt:testtnt:prj:shop:app:chk:env:prod", "            ref_urn_env: urn:tnt:testtnt:prj:shop:app:pay:env:prod")
		wantFinding(t, validateBody(t, body), "taxonomy-unknown-reference", "error", "urn:tnt:testtnt:prj:shop:app:pay:env:prod")
	})
}

func TestValidateDuplicateCodes(t *testing.T) {
	cases := []struct {
		name, old, new string
		want           string // subject: the parent's URN and the code; "" for none
	}{
		{"business units differing in case", "  - code: digital\n", "  - code: RETAIL\n", "urn:tnt:testtnt RETAIL"},
		{"cost centers of one business unit", "      - { code: ecom, name: E-commerce }", "      - { code: ecom, name: E-commerce }\n      - { code: ECOM, name: Again }", "urn:tnt:testtnt:bzu:retail ECOM"},
		{"projects", "  - code: payments\n", "  - code: Shop\n", "urn:tnt:testtnt Shop"},
		{"apps of the tenant, across projects", "      - code: pay\n", "      - code: CHK\n", "urn:tnt:testtnt CHK"},
		{"environments of one app", "{ code: dev, name: Dev }]\n  - code: payments", "{ code: DEV, name: Dev }, { code: dev, name: Dev }]\n  - code: payments", "urn:tnt:testtnt:prj:shop:app:chk dev"},
		{"the same cost center code under two business units", "{ code: pay, name: Payments }", "{ code: ecom, name: Payments }", ""},
		{"the same environment code under two apps", "      - code: pay\n", "      - code: pay\n", ""},
		{"a project and an app with one code", "  - code: payments\n", "  - code: pay\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := findCode(validateBody(t, change(t, tc.old, tc.new)), "taxonomy-duplicate-code")
			if tc.want == "" {
				if len(f) != 0 {
					t.Fatalf("unexpected %v", f)
				}
				return
			}
			if len(f) != 1 || f[0]["severity"] != "error" || f[0]["subject"] != tc.want {
				t.Fatalf("want one error with subject %q, got %v", tc.want, f)
			}
		})
	}

	t.Run("ou entries of one lz", func(t *testing.T) {
		body := change(t, "    ou_paths:\n", "    ou_paths:\n      - { code: Checkout, name: Again, path: /Other }\n")
		wantFinding(t, validateBody(t, body), "taxonomy-duplicate-code", "error", "urn:tnt:testtnt:lzc:main checkout")
	})
	t.Run("accounts of one ou", func(t *testing.T) {
		body := change(t, "        accounts:\n", "        accounts:\n          - { code: CHK-PROD, name: Again, ref_id: \"200000000009\" }\n")
		wantFinding(t, validateBody(t, body), "taxonomy-duplicate-code", "error", "urn:tnt:testtnt:lzc:main:org:checkout chk-prod")
	})
	t.Run("the same account code under two ous is fine", func(t *testing.T) {
		body := change(t, "    ou_paths:\n", "    ou_paths:\n      - code: other\n        name: Other\n        path: /Other\n        accounts:\n          - { code: chk-prod, name: x, ref_id: \"200000000008\" }\n")
		if f := findCode(validateBody(t, body), "taxonomy-duplicate-code"); len(f) != 0 {
			t.Fatalf("unexpected %v", f)
		}
	})
}

// TestValidateAliasUniqueness checks every node type that can carry an alias (D-057).
func TestValidateAliasUniqueness(t *testing.T) {
	const doc = `tenant: { code: testtnt, name: T }
business_units:
  - code: bu-a
    name: BU A
    alias: infra
    cost_centers:
      - code: cc-a
        name: CC A
        alias: cc-1
        owner_teams: [{ code: team-a, name: Team A, alias: alpha }]
  - code: bu-b
    name: BU B
    alias: platform
    cost_centers:
      - code: cc-b
        name: CC B
        alias: cc-2
        owner_teams: [{ code: team-b, name: Team B, alias: bravo }]
projects:
  - code: prj-a
    name: Prj A
    alias: proj-1
    ref_urn_bu: urn:tnt:testtnt:bzu:bu-a
    apps:
      - code: app-a
        name: App A
        alias: shop
        ref_urn_cct: urn:tnt:testtnt:bzu:bu-a:cct:cc-a
        environments: [{ code: prod, name: Prod, alias: p }, { code: stage, name: Stage, alias: s }]
  - code: prj-b
    name: Prj B
    alias: proj-2
    ref_urn_bu: urn:tnt:testtnt:bzu:bu-b
    apps:
      - code: app-b
        name: App B
        alias: hub
        ref_urn_cct: urn:tnt:testtnt:bzu:bu-b:cct:cc-b
        environments: [{ code: dev, name: Dev, alias: d }]
lzs:
  - code: lz-1
    name: Main LZ
    csp: aws
    currency: USD
    management_account: "111111111111"
    alias: aws-lz
    ou_paths:
      - code: ou-1
        name: Workloads
        path: /workloads
        alias: wkld
        accounts: [{ code: acct-main, name: Main, ref_id: "111111111111", alias: acct-main }]
      - code: ou-2
        name: Shared
        path: /shared
        alias: shrd
        accounts: [{ code: acct-sub, name: Sub, ref_id: "222222222222", alias: acct-sub }]
`
	if f := findCode(validateBody(t, doc), "taxonomy-duplicate-alias"); len(f) != 0 {
		t.Fatalf("unexpected duplicate alias on a valid taxonomy: %v", f)
	}
	cases := []struct {
		name, old, new, subject string
	}{
		{"business unit", "alias: platform", "alias: infra", "bzu infra"},
		{"cost center", "alias: cc-2", "alias: CC-1", "cct CC-1"},
		{"owner team", "alias: bravo", "alias: alpha", "ont alpha"},
		{"project", "alias: proj-2", "alias: proj-1", "prj proj-1"},
		{"app", "alias: hub", "alias: shop", "app shop"},
		{"environment, across apps", "alias: d }", "alias: p }", "env p"},
		{"lz is alone of its type", "alias: aws-lz", "alias: aws-lz", ""},
		{"ou", "alias: shrd", "alias: wkld", "org wkld"},
		{"account", "alias: acct-sub", "alias: acct-main", "acc acct-main"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if strings.Count(doc, tc.old) != 1 {
				t.Fatalf("%q is in the taxonomy %d times", tc.old, strings.Count(doc, tc.old))
			}
			f := findCode(validateBody(t, strings.Replace(doc, tc.old, tc.new, 1)), "taxonomy-duplicate-alias")
			if tc.subject == "" {
				if len(f) != 0 {
					t.Fatalf("unexpected %v", f)
				}
				return
			}
			if len(f) != 1 || f[0]["severity"] != "error" || f[0]["subject"] != tc.subject {
				t.Fatalf("want one error with subject %q, got %v", tc.subject, f)
			}
		})
	}
	t.Run("one alias on different types is allowed", func(t *testing.T) {
		body := strings.Replace(doc, "alias: infra", "alias: common", 1)
		body = strings.Replace(body, "alias: shop", "alias: common", 1)
		body = strings.Replace(body, "alias: proj-1", "alias: common", 1)
		if f := findCode(validateBody(t, body), "taxonomy-duplicate-alias"); len(f) != 0 {
			t.Fatalf("unexpected %v", f)
		}
	})
}

func TestValidateEnvironmentNotInApp(t *testing.T) {
	cases := []struct {
		name, old, new, subject string
	}{
		{"global default", "defaults: { app: chk, environment: dev }", "defaults: { app: chk, environment: stage }", "defaults"},
		{"lz default", "    management_account: \"111111111111\"\n", "    management_account: \"111111111111\"\n    default: { app: pay, environment: stage }\n", "main default"},
		{"account entry", "            ref_urn_apps: [urn:tnt:testtnt:prj:shop:app:chk]\n            ref_urn_env: urn:tnt:testtnt:prj:shop:app:chk:env:prod", "            ref_urn_apps: [urn:tnt:testtnt:prj:shop:app:chk]\n            ref_urn_env: urn:tnt:testtnt:prj:payments:app:pay:env:prod", "chk-prod"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wantFinding(t, validateBody(t, change(t, tc.old, tc.new)), "taxonomy-environment-not-in-app", "error", tc.subject)
		})
	}
	t.Run("an account naming only an environment is fine", func(t *testing.T) {
		body := change(t, "            ref_urn_apps: [urn:tnt:testtnt:prj:shop:app:chk]\n            ref_urn_env: urn:tnt:testtnt:prj:shop:app:chk:env:prod", "            ref_urn_env: urn:tnt:testtnt:prj:payments:app:pay:env:prod")
		if f := findCode(validateBody(t, body), "taxonomy-environment-not-in-app"); len(f) != 0 {
			t.Fatalf("unexpected %v", f)
		}
	})
	t.Run("a default naming an unknown app", func(t *testing.T) {
		wantFinding(t, validateBody(t, change(t, "defaults: { app: chk, environment: dev }", "defaults: { app: nope, environment: dev }")),
			"taxonomy-unknown-reference", "error", "defaults")
	})
}

func TestValidateMissingFields(t *testing.T) {
	cases := []struct {
		name, old, new, subject string
	}{
		{"business unit name", "  - code: retail\n    name: Retail\n", "  - code: retail\n", "retail"},
		{"project reference", "    ref_urn_bu: urn:tnt:testtnt:bzu:retail\n", "", "shop"},
		{"app reference", "        ref_urn_cct: urn:tnt:testtnt:bzu:retail:cct:ecom\n", "", "chk"},
		{"lz currency", "    currency: USD\n", "", "main"},
		{"ou path and ref_id", "        path: /Workloads/Checkout\n", "", "checkout"},
		{"account ref_id", "            ref_id: \"200000000001\"\n", "", "chk-prod"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wantFinding(t, validateBody(t, change(t, tc.old, tc.new)), "taxonomy-missing-field", "error", tc.subject)
		})
	}
	t.Run("an ou entry with a ref_id and no path is fine", func(t *testing.T) {
		body := change(t, "        path: /Workloads/Checkout\n", "        ref_id: ou-1\n")
		if f := findCode(validateBody(t, body), "taxonomy-missing-field"); len(f) != 0 {
			t.Fatalf("unexpected %v", f)
		}
	})
	t.Run("an ou or account entry with no references is fine", func(t *testing.T) {
		body := change(t, "        ref_urn_apps: [urn:tnt:testtnt:prj:shop:app:chk]\n        ref_urn_env: urn:tnt:testtnt:prj:shop:app:chk:env:prod\n", "")
		body = strings.Replace(body, "            ref_urn_apps: [urn:tnt:testtnt:prj:shop:app:chk]\n            ref_urn_env: urn:tnt:testtnt:prj:shop:app:chk:env:prod\n", "", 1)
		if f := validateBody(t, body); len(f) != 0 {
			t.Fatalf("unexpected %v", f)
		}
	})
}

func TestValidateDuplicateOULink(t *testing.T) {
	byPath := change(t, "    ou_paths:\n", "    ou_paths:\n      - { code: again, name: Again, path: /Workloads/Checkout }\n")
	wantFinding(t, validateBody(t, byPath), "taxonomy-duplicate-ou-link", "error", "/Workloads/Checkout")
	byID := change(t, "    ou_paths:\n", "    ou_paths:\n      - { code: one, name: One, ref_id: ou-1 }\n      - { code: two, name: Two, ref_id: ou-1 }\n")
	wantFinding(t, validateBody(t, byID), "taxonomy-duplicate-ou-link", "error", "ou-1")
}

func TestValidateUnused(t *testing.T) {
	body := change(t, "business_units:\n", "business_units:\n  - { code: orphan, name: Orphan, cost_centers: [{ code: lonely, name: Lonely }] }\n")
	body = strings.Replace(body, "projects:\n", "projects:\n  - { code: idle, name: Idle, ref_urn_bu: \"urn:tnt:testtnt:bzu:retail\" }\n", 1)
	f := findCode(validateBody(t, body), "taxonomy-unused")
	got := map[string]bool{}
	for _, x := range f {
		if x["severity"] != "warning" {
			t.Fatalf("severity = %v", x["severity"])
		}
		got[strID(x["subject"])] = true
	}
	for _, want := range []string{"orphan", "lonely", "idle"} {
		if !got[want] {
			t.Errorf("no taxonomy-unused for %s: %v", want, f)
		}
	}
	if len(got) != 3 {
		t.Errorf("taxonomy-unused findings = %v, want 3", f)
	}
	if HasErrorSeverity(validateBody(t, body)) {
		t.Errorf("unused nodes are warnings, not errors")
	}
}

func TestValidateIDRule(t *testing.T) {
	cases := []struct {
		name, old, new, subject string
	}{
		{"business unit too short", "  - code: retail\n", "  - code: ab\n", "ab"},
		{"cost center with a slash", "{ code: ecom, name: E-commerce }", "{ code: a/b, name: C }", "a/b"},
		{"project with a space", "  - code: shop\n", "  - code: my shop\n", "my shop"},
		{"app 51 characters", "      - code: chk\n", "      - code: " + strings.Repeat("x", 51) + "\n", strings.Repeat("x", 51)},
		{"environment too short", "{ code: dev, name: Dev }]\n  - code: payments", "{ code: qa, name: Dev }]\n  - code: payments", "qa"},
		{"lz with a colon", "  - code: main\n", "  - code: \"main:1\"\n", "main:1"},
		{"non-ASCII", "  - code: digital\n", "  - code: café\n", "café"},
		{"owner team", "{ code: ecom, name: E-commerce }", "{ code: ecom, name: E, owner_teams: [{ code: t, name: T }] }", "t"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wantFinding(t, validateBody(t, change(t, tc.old, tc.new)), "taxonomy-invalid-id", "error", tc.subject)
		})
	}
}

func TestValidateCodesThatPassTheRule(t *testing.T) {
	body := strings.NewReplacer(
		"code: retail", "code: Retail",
		"code: shop", "code: PRJ_Shop",
		"urn:tnt:testtnt:bzu:retail", "urn:tnt:testtnt:bzu:retail",
		"urn:tnt:testtnt:prj:shop", "urn:tnt:testtnt:prj:prj_shop",
		"code: chk\n", "code: "+strings.Repeat("a", 50)+"\n",
	).Replace(baseTaxonomy)
	body = strings.ReplaceAll(body, "app:chk", "app:"+strings.Repeat("a", 50))
	body = strings.Replace(body, "defaults: { app: chk,", "defaults: { app: "+strings.Repeat("a", 50)+",", 1)
	for _, f := range validateBody(t, body) {
		t.Fatalf("unexpected finding %v", f)
	}
}

func TestValidateReservedIDsInAnyCase(t *testing.T) {
	for _, code := range []string{"unalloc", "UNALLOC", "Unspec", "uNsPeC", "invalid", "INVALID"} {
		findings := validateBody(t, change(t, "  - code: digital\n", "  - code: "+code+"\n"))
		if n := len(findCode(findings, "taxonomy-reserved-code")); n != 1 {
			t.Errorf("code %q: %d taxonomy-reserved-code findings, want 1", code, n)
		}
		if n := len(findCode(findings, "taxonomy-invalid-id")); n != 0 {
			t.Errorf("code %q: %d taxonomy-invalid-id findings, want 0", code, n)
		}
	}
	// A longer code that merely contains one is fine.
	if f := findCode(validateBody(t, change(t, "  - code: digital\n", "  - code: unalloc-eu\n")), "taxonomy-reserved-code"); len(f) != 0 {
		t.Fatalf("unalloc-eu is not reserved: %v", f)
	}
	// A reserved environment id.
	if f := findCode(validateBody(t, change(t, "{ code: dev, name: Dev }]\n  - code: payments", "{ code: unspec, name: Dev }]\n  - code: payments")), "taxonomy-reserved-code"); len(f) != 1 {
		t.Fatalf("reserved environment: %v", f)
	}
}

func TestValidateLZCount(t *testing.T) {
	one := "  - { code: main, name: L, csp: aws, currency: USD, management_account: \"1\" }\n"
	two := one + "  - { code: other, name: L, csp: aws, currency: USD, management_account: \"2\" }\n"
	cases := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"none", "tenant: { code: testtnt, name: T }\n", true},
		{"one", "tenant: { code: testtnt, name: T }\nlzs:\n" + one, false},
		{"two", "tenant: { code: testtnt, name: T }\nlzs:\n" + two, true},
		{"two under lz_conns", "tenant: { code: testtnt, name: T }\nlz_conns:\n" + two, true},
		{"one under lz_conns", "tenant: { code: testtnt, name: T }\nlz_conns:\n" + one, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			found := findCode(validateBody(t, tc.body), "taxonomy-lz-count")
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
	t.Run("both keys stop the loader and name both", func(t *testing.T) {
		_, err := taxonomy.Parse([]byte("tenant: { code: testtnt, name: T }\nlzs:\n" + one + "lz_conns:\n" + one))
		if err == nil || !strings.Contains(err.Error(), "lzs") || !strings.Contains(err.Error(), "lz_conns") {
			t.Fatalf("err = %v, want one naming lzs and lz_conns", err)
		}
	})
}

func TestLoadNestedTaxonomy(t *testing.T) {
	tax, err := taxonomy.Load(filepath.Join("testdata", "nested-taxonomy.yaml"))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if tax.Tenant.Code != "nesttnt" {
		t.Errorf("want tenant nesttnt, got %q", tax.Tenant.Code)
	}
	if len(tax.BusinessUnits) != 2 {
		t.Errorf("want 2 BUs, got %d", len(tax.BusinessUnits))
	}
	ccs, apps := 0, 0
	for _, bu := range tax.BusinessUnits {
		ccs += len(bu.CostCenters)
	}
	for _, p := range tax.Projects {
		apps += len(p.Apps)
	}
	if ccs != 2 {
		t.Errorf("want 2 CCs, got %d", ccs)
	}
	if len(tax.Projects) != 2 {
		t.Errorf("want 2 Projects, got %d", len(tax.Projects))
	}
	if apps != 3 {
		t.Errorf("want 3 Apps, got %d", apps)
	}
	if len(tax.LZs) != 1 {
		t.Errorf("want 1 LZ, got %d", len(tax.LZs))
	}
	for _, f := range ValidateTaxonomy(tax) {
		if f["severity"] == "error" {
			t.Errorf("unexpected validation error: %v", f)
		}
	}
}

// TestCodesAreReadAsWritten checks that a code is not taken from its URN (WO-0111-B): PRJ_Shop
// stays PRJ_Shop in the model, and its URN is built lowercase.
func TestCodesAreReadAsWritten(t *testing.T) {
	body := strings.ReplaceAll(strings.ReplaceAll(baseTaxonomy, "code: shop", "code: PRJ_Shop"), "prj:shop", "prj:prj_shop")
	tax := parseTaxonomy(t, body)
	if got := tax.Projects[0].Code; got != "PRJ_Shop" {
		t.Fatalf("project code = %q", got)
	}
	if f := validateBody(t, body); len(f) != 0 {
		t.Fatalf("findings = %v", f)
	}
}
