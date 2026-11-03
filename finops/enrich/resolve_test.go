package enrich

import (
	"fmt"
	"strings"
	"testing"

	"github.com/PopGreenTh/OpenXOps/record"
)

// resolveTaxonomy is a tenant with two Apps, chk and pay, both with the Environments prod and dev,
// and one LZ whose ou_paths come from the test.
const resolveTaxonomy = `tenant: { code: testtnt, name: Test tenant }
business_units:
  - { code: retail, name: Retail, cost_centers: [{ code: ecom, name: E-commerce }] }
projects:
  - code: shop
    name: Shop
    ref_urn_bu: urn:tnt:testtnt:bzu:retail
    apps:
      - code: chk
        name: Checkout
        ref_urn_cct: urn:tnt:testtnt:bzu:retail:cct:ecom
        environments: [{ code: prod, name: Prod }, { code: dev, name: Dev }]
      - code: pay
        name: Pay
        ref_urn_cct: urn:tnt:testtnt:bzu:retail:cct:ecom
        environments: [{ code: prod, name: Prod }, { code: dev, name: Dev }]
lzs:
  - code: main
    name: Main
    csp: aws
    currency: USD
    management_account: "111111111111"
    ou_paths:
%s`

const (
	chkURN  = "urn:tnt:testtnt:prj:shop:app:chk"
	payURN  = "urn:tnt:testtnt:prj:shop:app:pay"
	chkProd = chkURN + ":env:prod"
	payProd = payURN + ":env:prod"
	payDev  = payURN + ":env:dev"
)

func ouRec(id, parent, name, path, idPath string, children bool) record.Record {
	return record.Record{"kind": "ou", "csp": "aws", "org_id": "o-main", "management_account": "111111111111",
		"id": id, "name": name, "parent_id": parent, "path": path, "id_path": idPath, "has_children": children}
}

// resolveInventory has account 200000000001 in the leaf OU /Workloads/Checkout/Prod.
func resolveInventory() []record.Record {
	return []record.Record{
		ouRec("r-1", "", "Root", "/", "r-1", true),
		ouRec("ou-1", "r-1", "Workloads", "/Workloads", "r-1/ou-1", true),
		ouRec("ou-2", "ou-1", "Checkout", "/Workloads/Checkout", "r-1/ou-1/ou-2", true),
		ouRec("ou-3", "ou-2", "Prod", "/Workloads/Checkout/Prod", "r-1/ou-1/ou-2/ou-3", false),
		ouRec("ou-9", "r-1", "Other", "/Other", "r-1/ou-9", false),
		{"kind": "account", "csp": "aws", "org_id": "o-main", "management_account": "111111111111", "id": "200000000001",
			"name": "chk-prod", "email": "a@example.com", "status": "ACTIVE", "parent_id": "ou-3",
			"ou_path": "/Workloads/Checkout/Prod", "ou_id_path": "r-1/ou-1/ou-2/ou-3"},
	}
}

// mapWith maps resolveInventory with the given ou_paths entries (indented list items).
func mapWith(t *testing.T, ouPaths string) []record.Record {
	t.Helper()
	tax := parseTaxonomy(t, fmt.Sprintf(resolveTaxonomy, ouPaths))
	return Map(tax, resolveInventory())
}

func TestOUWithTwoAppsMakesItsAccountsShared(t *testing.T) {
	out := mapWith(t, fmt.Sprintf("      - { code: checkout, name: Checkout, path: /Workloads/Checkout, ref_urn_apps: [%s, %s], ref_urn_env: %s }\n", chkURN, payURN, payDev))
	allocs := filterKind(out, "allocation")
	if len(allocs) != 2 {
		t.Fatalf("allocations = %v", allocs)
	}
	for i, app := range []string{"chk", "pay"} {
		a := allocs[i]
		if a["app"] != app || a["shared"] != true || a["app_rule"] != "ou:/Workloads/Checkout" || a["environment"] != "dev" {
			t.Errorf("allocation %d = %v", i, a)
		}
	}
	if f := filterKind(out, "finding"); len(f) != 0 {
		t.Errorf("findings = %v", f)
	}
}

func TestAccountEntryNamingOneAppBeatsAnOUWithTwo(t *testing.T) {
	out := mapWith(t, fmt.Sprintf(`      - { code: checkout, name: Checkout, path: /Workloads/Checkout, ref_urn_apps: [%s, %s] }
      - code: prod
        name: Prod
        path: /Workloads/Checkout/Prod
        accounts:
          - { code: chk-prod, name: chk-prod, ref_id: "200000000001", ref_urn_apps: [%s], ref_urn_env: %s }
`, chkURN, payURN, payURN, payProd))
	allocs := filterKind(out, "allocation")
	if len(allocs) != 1 || allocs[0]["app"] != "pay" || allocs[0]["shared"] != false ||
		allocs[0]["app_rule"] != "account" || allocs[0]["environment"] != "prod" || allocs[0]["environment_rule"] != "account" {
		t.Fatalf("allocations = %v", allocs)
	}
}

func TestAppOnALeafOUApplies(t *testing.T) {
	out := mapWith(t, fmt.Sprintf("      - { code: prod, name: Prod, path: /Workloads/Checkout/Prod, ref_urn_apps: [%s], ref_urn_env: %s }\n", payURN, payProd))
	allocs := filterKind(out, "allocation")
	if len(allocs) != 1 || allocs[0]["app"] != "pay" || allocs[0]["app_rule"] != "ou:/Workloads/Checkout/Prod" || allocs[0]["environment"] != "prod" {
		t.Fatalf("allocations = %v", allocs)
	}
	if f := filterKind(out, "finding"); len(f) != 0 {
		t.Errorf("an App on a leaf OU is not a finding any more: %v", f)
	}
}

func TestOUNameIsNotAnEnvironment(t *testing.T) {
	// The OU is named Prod and chk has an Environment prod, but nothing links them.
	out := mapWith(t, fmt.Sprintf("      - { code: prod, name: Prod, path: /Workloads/Checkout/Prod, ref_urn_apps: [%s] }\n", chkURN))
	allocs := filterKind(out, "allocation")
	if len(allocs) != 1 || allocs[0]["app"] != "chk" || allocs[0]["environment"] != "unspec" || allocs[0]["environment_rule"] != "unspec" {
		t.Fatalf("allocations = %v", allocs)
	}
	if f := findCode(out, "environment-unresolved"); len(f) != 1 || f[0]["subject"] != "200000000001" {
		t.Fatalf("findings = %v", filterKind(out, "finding"))
	}
}

func TestOUEnvironmentDoesNotSetTheApp(t *testing.T) {
	out := mapWith(t, fmt.Sprintf("      - { code: prod, name: Prod, path: /Workloads/Checkout/Prod, ref_urn_env: %s }\n", chkProd))
	allocs := filterKind(out, "allocation")
	if len(allocs) != 1 || allocs[0]["app"] != "unalloc" || allocs[0]["environment"] != "prod" {
		t.Fatalf("allocations = %v", allocs)
	}
	if f := findCode(out, "account-unmapped"); len(f) != 1 {
		t.Fatalf("findings = %v", filterKind(out, "finding"))
	}
}

func TestAccountEnvironmentSetsTheAppOnlyWithoutApps(t *testing.T) {
	entry := func(fields string) string {
		return fmt.Sprintf(`      - code: prod
        name: Prod
        path: /Workloads/Checkout/Prod
        accounts:
          - { code: chk-prod, name: chk-prod, ref_id: "200000000001", %s }
`, fields)
	}
	out := mapWith(t, entry("ref_urn_env: "+payProd))
	if a := filterKind(out, "allocation"); len(a) != 1 || a[0]["app"] != "pay" || a[0]["app_rule"] != "account" || a[0]["environment"] != "prod" {
		t.Errorf("env only: allocations = %v", a)
	}
	out = mapWith(t, entry(fmt.Sprintf("ref_urn_apps: [%s], ref_urn_env: %s", chkURN, payProd)))
	if a := filterKind(out, "allocation"); len(a) != 1 || a[0]["app"] != "chk" || a[0]["environment"] != "prod" {
		t.Errorf("apps and env: allocations = %v", a)
	}
}

func TestMovedAccountKeepsItsReferencesAndGetsAMismatch(t *testing.T) {
	// The taxonomy lists the account under /Other; it was discovered under /Workloads/Checkout/Prod.
	out := mapWith(t, fmt.Sprintf(`      - code: other
        name: Other
        path: /Other
        accounts:
          - { code: chk-prod, name: chk-prod, ref_id: "200000000001", ref_urn_apps: [%s], ref_urn_env: %s }
`, payURN, payDev))
	allocs := filterKind(out, "allocation")
	if len(allocs) != 1 || allocs[0]["app"] != "pay" || allocs[0]["environment"] != "dev" || allocs[0]["app_rule"] != "account" {
		t.Fatalf("allocations = %v", allocs)
	}
	f := findCode(out, "account-ou-mismatch")
	if len(f) != 1 || f[0]["severity"] != "warning" || f[0]["subject"] != "200000000001" {
		t.Fatalf("findings = %v", filterKind(out, "finding"))
	}
	msg := fmt.Sprint(f[0]["message"])
	if !strings.Contains(msg, "/Other") || !strings.Contains(msg, "/Workloads/Checkout/Prod") {
		t.Errorf("message %q should name both paths", msg)
	}
	if n := len(filterKind(out, "finding")); n != 1 {
		t.Errorf("want only the mismatch, got %v", filterKind(out, "finding"))
	}
}

func TestAccountUnderItsOUHasNoMismatch(t *testing.T) {
	out := mapWith(t, fmt.Sprintf(`      - code: prod
        name: Prod
        ref_id: ou-3
        accounts:
          - { code: chk-prod, name: chk-prod, ref_id: "200000000001", ref_urn_apps: [%s], ref_urn_env: %s }
`, payURN, payDev))
	if f := filterKind(out, "finding"); len(f) != 0 {
		t.Fatalf("findings = %v", f)
	}
}

func TestOUEntryWithARefIDMatchesByIDOnly(t *testing.T) {
	// The id names no OU; the path would match, but a ref_id decides (D-061).
	out := mapWith(t, fmt.Sprintf("      - { code: checkout, name: Checkout, ref_id: ou-zzzz, path: /Workloads/Checkout, ref_urn_apps: [%s] }\n", chkURN))
	if f := findCode(out, "ou-not-found"); len(f) != 1 || f[0]["subject"] != "/Workloads/Checkout" {
		t.Fatalf("findings = %v", filterKind(out, "finding"))
	}
	if a := filterKind(out, "allocation"); len(a) != 1 || a[0]["app"] != "unalloc" {
		t.Fatalf("allocations = %v", a)
	}
}

// collisionInventory has two OUs named Prod under different parents and two accounts named
// chk-prod in one OU.
func collisionInventory() []record.Record {
	acct := func(id, name, parent, path string) record.Record {
		return record.Record{"kind": "account", "csp": "aws", "org_id": "o-main", "management_account": "111111111111", "id": id,
			"name": name, "email": id + "@example.com", "status": "ACTIVE", "parent_id": parent, "ou_path": path}
	}
	return []record.Record{
		ouRec("r-1", "", "Root", "/", "r-1", true),
		ouRec("ou-a", "r-1", "Checkout", "/Checkout", "r-1/ou-a", true),
		ouRec("ou-b", "r-1", "Payments", "/Payments", "r-1/ou-b", true),
		ouRec("ou-ap", "ou-a", "Prod", "/Checkout/Prod", "r-1/ou-a/ou-ap", false),
		ouRec("ou-bp", "ou-b", "Prod", "/Payments/Prod", "r-1/ou-b/ou-bp", false),
		acct("200000000001", "chk-prod", "ou-ap", "/Checkout/Prod"),
		acct("200000000002", "chk-prod", "ou-ap", "/Checkout/Prod"),
	}
}

func TestCollisionsStopOnTwoOUsWithOneURN(t *testing.T) {
	recs := collisionInventory()[:5] // no accounts
	err := CheckCollisions(parseTaxonomy(t, fmt.Sprintf(resolveTaxonomy, "      - { code: checkout, name: Checkout, path: /Checkout }\n")), recs)
	if err == nil {
		t.Fatal("two OUs named Prod under different parents should collide")
	}
	for _, want := range []string{"/Checkout/Prod", "/Payments/Prod", "urn:tnt:testtnt:lzc:main:org:prod"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should contain %q", err, want)
		}
	}
}

func TestListingOneOUBreaksTheTie(t *testing.T) {
	tax := parseTaxonomy(t, fmt.Sprintf(resolveTaxonomy, "      - { code: payments-prod, name: Prod, path: /Payments/Prod }\n"))
	if err := CheckCollisions(tax, collisionInventory()[:5]); err != nil {
		t.Fatal(err)
	}
}

func TestCollisionsStopOnTwoAccountsWithOneURN(t *testing.T) {
	err := CheckCollisions(parseTaxonomy(t, fmt.Sprintf(resolveTaxonomy, "      - { code: prod, name: Prod, path: /Checkout/Prod }\n")), collisionInventory()[:1:1])
	if err != nil {
		t.Fatalf("no accounts, no collision: %v", err)
	}
	recs := append(collisionInventory()[:4:4], collisionInventory()[5:]...)
	err = CheckCollisions(parseTaxonomy(t, fmt.Sprintf(resolveTaxonomy, "      - { code: prod, name: Prod, path: /Checkout/Prod }\n")), recs)
	if err == nil {
		t.Fatal("two accounts named chk-prod in one OU should collide")
	}
	for _, want := range []string{"200000000001", "200000000002", "org:prod:acc:chk-prod"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should contain %q", err, want)
		}
	}
	// Listing one account gives it a code of its own.
	listed := "      - code: prod\n        name: Prod\n        path: /Checkout/Prod\n        accounts:\n          - { code: chk-prod-two, name: second, ref_id: \"200000000002\" }\n"
	if err := CheckCollisions(parseTaxonomy(t, fmt.Sprintf(resolveTaxonomy, listed)), recs); err != nil {
		t.Fatal(err)
	}
}

func TestOneNameInTwoOUsIsNotACollision(t *testing.T) {
	recs := collisionInventory()
	recs[6]["parent_id"], recs[6]["ou_path"] = "ou-bp", "/Payments/Prod" // the second chk-prod moves to the other Prod
	tax := parseTaxonomy(t, fmt.Sprintf(resolveTaxonomy, "      - { code: checkout-prod, name: Prod, path: /Checkout/Prod }\n      - { code: payments-prod, name: Prod, path: /Payments/Prod }\n"))
	if err := CheckCollisions(tax, recs); err != nil {
		t.Fatal(err)
	}
}
