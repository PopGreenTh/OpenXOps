package schema

import (
	"strings"
	"testing"
)

func TestEveryContractKindHasSchema(t *testing.T) {
	want := []string{"ou", "account", "cost", "project_tag_cost", "allocation", "change", "finding"}
	for _, k := range want {
		if _, ok := ByKind[k]; !ok {
			t.Fatalf("missing schema for kind %q", k)
		}
	}
	if len(ByKind) != len(want) {
		t.Fatalf("ByKind has %d entries, want %d", len(ByKind), len(want))
	}
}

func TestEverySchemaRequiresTenant(t *testing.T) {
	for kind, sch := range ByKind {
		found := false
		for _, f := range sch.Fields {
			if f.Name == "tenant" {
				found = true
				if f.Type != TypeString || !f.Required {
					t.Errorf("kind %q: tenant must be a required string", kind)
				}
			}
		}
		if !found {
			t.Errorf("kind %q: no tenant field", kind)
		}
	}
}

func TestValidateMissingTenant(t *testing.T) {
	rec := map[string]any{"kind": "finding", "severity": "warning", "code": "x", "subject": "s", "message": "m", "command": "c"}
	check := map[string]struct{}{"finding": {}}
	err := Validate("f.jsonl", 3, rec, check)
	if err == nil || !strings.Contains(err.Error(), "f.jsonl:3:") || !strings.Contains(err.Error(), `"tenant"`) {
		t.Fatalf("error = %v, want missing tenant at f.jsonl:3", err)
	}
}

func TestValidateMissingField(t *testing.T) {
	rec := map[string]any{"kind": "finding", "severity": "warning", "code": "x", "subject": "s", "message": "m"}
	check := map[string]struct{}{"finding": {}}
	err := Validate("stdin", 2, rec, check)
	if err == nil {
		t.Fatal("expected missing command field")
	}
	if !strings.Contains(err.Error(), "stdin:2:") || !strings.Contains(err.Error(), `"command"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestValidateWrongType(t *testing.T) {
	rec := map[string]any{
		"kind": "cost", "csp": "aws", "management_account": "1", "granularity": "monthly",
		"period": "2026-01", "metric": "UnblendedCost", "amount": "not-a-number", "currency": "USD",
		"estimated": false, "account_id": "1", "service": "s", "app_tag": "",
	}
	check := map[string]struct{}{"cost": {}}
	err := Validate("f.jsonl", 1, rec, check)
	if err == nil {
		t.Fatal("expected wrong type on amount")
	}
	if !strings.Contains(err.Error(), "f.jsonl:1:") || !strings.Contains(err.Error(), `"amount"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestValidateExtraFieldPasses(t *testing.T) {
	rec := map[string]any{
		"kind": "finding", "severity": "warning", "code": "c", "subject": "s", "message": "m",
		"command": "cmd", "tenant": "testtnt", "extra": "allowed",
	}
	check := map[string]struct{}{"finding": {}}
	if err := Validate("stdin", 1, rec, check); err != nil {
		t.Fatal(err)
	}
}

func TestValidateUndeclaredKindSkipped(t *testing.T) {
	rec := map[string]any{"kind": "custom", "foo": 1}
	check := map[string]struct{}{"cost": {}}
	if err := Validate("stdin", 1, rec, check); err != nil {
		t.Fatal(err)
	}
}

var urnFields = []string{"business_unit_urn", "cost_center_urn", "project_urn", "app_urn", "environment_urn", "lz_urn", "ou_urn", "account_urn"}

func fieldOf(kind, name string) (Field, bool) {
	for _, f := range ByKind[kind].Fields {
		if f.Name == name {
			return f, true
		}
	}
	return Field{}, false
}

func TestURNFieldsRequiredOnAllocationOptionalOnCost(t *testing.T) {
	for _, name := range urnFields {
		f, ok := fieldOf("allocation", name)
		if !ok || f.Type != TypeString || !f.Required {
			t.Errorf("allocation.%s = %+v, want a required string", name, f)
		}
		f, ok = fieldOf("cost", name)
		if !ok || f.Type != TypeString || f.Required {
			t.Errorf("cost.%s = %+v, want an optional string", name, f)
		}
	}
	if f, ok := fieldOf("cost", "service_urn"); !ok || f.Required || f.Type != TypeString {
		t.Errorf("cost.service_urn = %+v, want an optional string", f)
	}
	if _, ok := fieldOf("allocation", "service_urn"); ok {
		t.Error("allocation must not have service_urn")
	}
}

func TestValidateAllocationMissingURN(t *testing.T) {
	rec := map[string]any{
		"kind": "allocation", "account_id": "1", "account_name": "a", "lz": "main", "ou_path": "/", "project": "p",
		"app": "a", "environment": "e", "business_unit": "b", "cost_center": "c", "app_rule": "r", "environment_rule": "r",
		"shared": false, "tenant": "testtnt",
	}
	check := map[string]struct{}{"allocation": {}}
	err := Validate("m.jsonl", 4, rec, check)
	if err == nil || !strings.Contains(err.Error(), "m.jsonl:4:") || !strings.Contains(err.Error(), "_urn") {
		t.Fatalf("error = %v, want a missing URN field at m.jsonl:4", err)
	}
	for _, name := range urnFields {
		rec[name] = "urn:tnt:testtnt"
	}
	if err := Validate("m.jsonl", 4, rec, check); err != nil {
		t.Fatal(err)
	}
}

func TestValidateCostWithoutURNsPasses(t *testing.T) {
	rec := map[string]any{
		"kind": "cost", "csp": "aws", "management_account": "1", "granularity": "monthly", "period": "2026-01",
		"metric": "UnblendedCost", "amount": 1.0, "currency": "USD", "estimated": false, "account_id": "1",
		"service": "s", "app_tag": "", "tenant": "testtnt",
	}
	if err := Validate("c.jsonl", 1, rec, map[string]struct{}{"cost": {}}); err != nil {
		t.Fatal(err)
	}
	rec["lz_urn"] = 5.0
	if err := Validate("c.jsonl", 1, rec, map[string]struct{}{"cost": {}}); err == nil {
		t.Fatal("a non-string URN must be rejected")
	}
}
