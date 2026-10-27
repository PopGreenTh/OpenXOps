package record

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestNestedValueRejected(t *testing.T) {
	_, err := ReadFile("testdata/nested.jsonl")
	if err == nil {
		t.Fatal("expected nested value to be rejected")
	}
	if !strings.Contains(err.Error(), "nested") {
		t.Fatalf("error should mention nested value: %v", err)
	}
	if !strings.Contains(err.Error(), "testdata/nested.jsonl:1:") {
		t.Fatalf("error should name file and line: %v", err)
	}
}

func TestBadLineReportsFileAndLine(t *testing.T) {
	_, err := ReadFile("testdata/bad.jsonl")
	if err == nil {
		t.Fatal("expected bad line to be rejected")
	}
	if !strings.Contains(err.Error(), "testdata/bad.jsonl:2:") {
		t.Fatalf("error should name file and line 2: %v", err)
	}
}

func TestUnknownKindsPassThroughInOrder(t *testing.T) {
	in, err := ReadFile("testdata/mixed.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	own := []Record{{"kind": "cost", "id": "processed"}}
	got := PassThrough(in, []string{"cost"}, own)
	wantKinds := []string{"cost", "account", "ou"}
	wantIDs := []string{"processed", "a1", "o1"}
	if len(got) != len(wantKinds) {
		t.Fatalf("got %d records %v, want %d", len(got), kindsOf(got), len(wantKinds))
	}
	for i := range wantKinds {
		if got[i].Kind() != wantKinds[i] {
			t.Fatalf("record %d kind = %q, want %q", i, got[i].Kind(), wantKinds[i])
		}
		if got[i]["id"] != wantIDs[i] {
			t.Fatalf("record %d id = %v, want %q", i, got[i]["id"], wantIDs[i])
		}
	}
}

func TestPipePlusFlagSameKindError(t *testing.T) {
	pipe := strings.NewReader("{\"kind\":\"cost\",\"id\":\"p\"}\n")
	_, err := Assemble(pipe, true, []From{{
		Name:  "cost",
		Kinds: []string{"cost"},
		File:  "testdata/from-cost.jsonl",
	}})
	if err == nil {
		t.Fatal("expected conflict between pipe and flag")
	}
	msg := err.Error()
	if !strings.Contains(msg, `"cost"`) {
		t.Fatalf("error should name the kind: %v", err)
	}
	if !strings.Contains(msg, "stdin") || !strings.Contains(msg, "--from-cost") {
		t.Fatalf("error should name both sources: %v", err)
	}
}

func TestMixedPipeSortedByKind(t *testing.T) {
	f, err := os.Open("testdata/mixed.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	recs, err := Assemble(f, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := kindsOf(recs)
	want := []string{"account", "cost", "cost", "ou"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	if recs[1]["id"] != "c1" || recs[2]["id"] != "c2" {
		t.Fatalf("same-kind order should be stable: %v %v", recs[1]["id"], recs[2]["id"])
	}
}

func TestWriteRoundTrip(t *testing.T) {
	in := []Record{
		Finding("warning", "lz-unknown", "o-ab12cd34", "unknown landing zone", "finops enrich map"),
		{"kind": "cost", "amount": 1.5, "period": "2026-01"},
	}
	var buf bytes.Buffer
	if err := Write(&buf, in); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(buf.Bytes(), []byte("\n\n")) {
		t.Fatal("writer emitted a blank line")
	}
	out, err := Read(&buf, "mem")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0].Kind() != "finding" || out[1].Kind() != "cost" {
		t.Fatalf("round trip = %v", kindsOf(out))
	}
	if out[0]["code"] != "lz-unknown" || out[0]["subject"] != "o-ab12cd34" {
		t.Fatalf("finding fields = %v", out[0])
	}
}

func TestAssembleFlagOnly(t *testing.T) {
	recs, err := Assemble(nil, false, []From{{
		Name:  "cost",
		Kinds: []string{"cost"},
		File:  "testdata/from-cost.jsonl",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0]["id"] != "flag" {
		t.Fatalf("got %v", recs)
	}
}

func kindsOf(recs []Record) []string {
	out := make([]string, len(recs))
	for i, r := range recs {
		out[i] = r.Kind()
	}
	return out
}
