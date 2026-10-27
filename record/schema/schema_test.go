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
		"command": "cmd", "extra": "allowed",
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
