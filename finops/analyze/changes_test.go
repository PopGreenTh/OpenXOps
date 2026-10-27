package analyze

import (
	"path/filepath"
	"testing"

	"github.com/PopGreenTh/OpenXOps/finops/enrich"
	"github.com/PopGreenTh/OpenXOps/record"
)

func TestChangesAccountAddedRemoved(t *testing.T) {
	before := []record.Record{
		{"kind": "account", "id": "1", "ou_path": "/a"},
	}
	now := []record.Record{
		{"kind": "account", "id": "2", "ou_path": "/b"},
	}
	ch := CompareChanges(before, now)
	if len(ch) != 2 {
		t.Fatalf("changes = %v", ch)
	}
}

func TestChangesAccountMoved(t *testing.T) {
	before := []record.Record{{"kind": "account", "id": "1", "ou_path": "/old"}}
	now := []record.Record{
		{"kind": "account", "id": "1", "ou_path": "/new"},
		{"kind": "cost", "account_id": "1", "amount": 10.0, "currency": "USD", "period": "2026-09", "metric": "UnblendedCost"},
	}
	ch := CompareChanges(before, now)
	found := false
	for _, c := range ch {
		if c["change"] == "moved" && c["field"] == "ou_path" {
			found = true
			if c["cost_moved"] == nil {
				t.Fatal("expected cost_moved")
			}
		}
	}
	if !found {
		t.Fatalf("no moved change: %v", ch)
	}
}

func TestChangesSingleAppRemapped(t *testing.T) {
	before := []record.Record{
		{"kind": "allocation", "account_id": "1", "app": "APP-A", "environment": "E"},
	}
	now := []record.Record{
		{"kind": "allocation", "account_id": "1", "app": "APP-B", "environment": "E"},
	}
	ch := CompareChanges(before, now)
	if len(ch) != 1 || ch[0]["change"] != "remapped" || ch[0]["subject_type"] != "account" {
		t.Fatalf("changes = %v", ch)
	}
}

func TestChangesSharedAllocationAdded(t *testing.T) {
	before := []record.Record{
		{"kind": "allocation", "account_id": "1", "app": "APP-A"},
	}
	now := []record.Record{
		{"kind": "allocation", "account_id": "1", "app": "APP-A"},
		{"kind": "allocation", "account_id": "1", "app": "APP-B"},
	}
	ch := CompareChanges(before, now)
	found := false
	for _, c := range ch {
		if c["change"] == "added" && c["subject"] == "1/APP-B" {
			found = true
		}
	}
	if !found {
		t.Fatalf("changes = %v", ch)
	}
}

func TestChangesTwoCurrencies(t *testing.T) {
	before := []record.Record{
		{"kind": "account", "id": "1", "ou_path": "/old"},
		{"kind": "allocation", "account_id": "1", "app": "A", "environment": "E"},
	}
	now := []record.Record{
		{"kind": "account", "id": "1", "ou_path": "/new"},
		{"kind": "allocation", "account_id": "1", "app": "A", "environment": "E"},
		{"kind": "cost", "account_id": "1", "amount": 5.0, "currency": "USD", "period": "2026-09", "metric": "M"},
		{"kind": "cost", "account_id": "1", "amount": 3.0, "currency": "EUR", "period": "2026-09", "metric": "M"},
	}
	ch := CompareChanges(before, now)
	if len(ch) != 2 {
		t.Fatalf("want 2 change rows (one per currency), got %d: %v", len(ch), ch)
	}
}

func TestChangesWithoutCost(t *testing.T) {
	before := []record.Record{
		{"kind": "allocation", "account_id": "1", "app": "A", "environment": "E1", "project": "P"},
	}
	now := []record.Record{
		{"kind": "allocation", "account_id": "1", "app": "A", "environment": "E2", "project": "P"},
	}
	ch := CompareChanges(before, now)
	if len(ch) != 1 {
		t.Fatalf("changes = %v", ch)
	}
	c := ch[0]
	if c["cost_moved"] != nil || c["currency"] != "" || c["period_from"] != "" || c["period_to"] != "" {
		t.Fatalf("expected empty cost fields, got %v", c)
	}
}

func TestChangesDemoEnvironmentRemapped(t *testing.T) {
	accounts, err := record.ReadFile(filepath.Join("..", "enrich", "testdata", "demo-accounts.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	costs, err := record.ReadFile(filepath.Join("testdata", "cost.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	in := append(accounts, costs...)
	taxBefore, err := enrich.LoadTaxonomy(filepath.Join("..", "enrich", "testdata", "demo-taxonomy.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	taxNow, err := enrich.LoadTaxonomy(filepath.Join("testdata", "demo-taxonomy-env-changed.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	before := enrich.Map(taxBefore, in)
	now := enrich.Map(taxNow, in)
	ch := CompareChanges(before, now)
	var remapped []record.Record
	for _, c := range ch {
		if c["change"] == "remapped" && c["field"] == "environment" {
			remapped = append(remapped, c)
		}
	}
	if len(remapped) != 1 {
		t.Fatalf("remapped environment changes = %d, all changes: %v", len(remapped), ch)
	}
	c := remapped[0]
	if c["subject_type"] != "account" || c["subject"] != "200000000001" {
		t.Fatalf("subject = %v / %v", c["subject_type"], c["subject"])
	}
	if c["old"] != "ENV-PROD" || c["new"] != "ENV-NONPROD" {
		t.Fatalf("old/new = %v / %v", c["old"], c["new"])
	}
	if c["cost_moved"] != 42.5 || c["currency"] != "USD" {
		t.Fatalf("cost = %v %v", c["cost_moved"], c["currency"])
	}
}
