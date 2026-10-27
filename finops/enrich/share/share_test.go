package share

import (
	"testing"

	"github.com/PopGreenTh/OpenXOps/record"
)

func TestPassThroughNonCostOnceInInputOrder(t *testing.T) {
	in := []record.Record{
		{"kind": "ou", "id": "ou-1"},
		{"kind": "account", "id": "acct-1"},
		{"kind": "allocation", "account_id": "200000000002", "app": "APP-A", "project": "PRJ", "business_unit": "BU", "cost_center": "CC", "app_rule": "account", "environment": "ENV-PROD"},
		{"kind": "finding", "code": "map-finding", "subject": "s1", "command": "finops enrich map"},
		{"kind": "cost", "account_id": "200000000001", "shared": false, "amount": 1.0, "period": "2026-09", "metric": "UnblendedCost", "currency": "USD"},
		{"kind": "cost", "account_id": "200000000002", "shared": true, "amount": 100.0, "period": "2026-09", "metric": "UnblendedCost", "currency": "USD", "app_tag": "APP-A"},
		{"kind": "finding", "code": "after-cost", "subject": "s2", "command": "finops enrich map"},
	}

	opt := Options{AppTagKey: "App", AppCodes: []string{"APP-A", "APP-B"}}
	out, err := Split(in, opt)
	if err != nil {
		t.Fatal(err)
	}

	var wantPassthrough []record.Record
	for _, r := range in {
		if r.Kind() != "cost" {
			wantPassthrough = append(wantPassthrough, r)
		}
	}

	idx := 0
	for idx < len(out) && out[idx].Kind() == "cost" {
		idx++
	}
	for idx < len(out) && out[idx].Kind() == "finding" && isShareFinding(out[idx]) {
		idx++
	}
	gotPassthrough := out[idx:]

	if len(gotPassthrough) != len(wantPassthrough) {
		t.Fatalf("passthrough count = %d, want %d\nfull out=%v", len(gotPassthrough), len(wantPassthrough), out)
	}
	for i := range wantPassthrough {
		if !recordsEqual(gotPassthrough[i], wantPassthrough[i]) {
			t.Fatalf("passthrough[%d]: got %v, want %v", i, gotPassthrough[i], wantPassthrough[i])
		}
	}
	if idx == 0 || out[0].Kind() != "cost" {
		t.Fatalf("output should start with cost records, got %v", out)
	}
}

func isShareFinding(r record.Record) bool {
	if r.Kind() != "finding" {
		return false
	}
	switch r["code"] {
	case "shared-untagged", "shared-tag-wrong-app", "shared-tag-unknown", "app-tag-not-seen":
		return true
	}
	return r["command"] == cmdShare
}

func recordsEqual(a, b record.Record) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
