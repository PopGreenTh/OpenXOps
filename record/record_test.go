package record

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestNestedValueRejected(t *testing.T) {
	_, err := ReadFile("testdata/nested.jsonl", nil)
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
	_, err := ReadFile("testdata/bad.jsonl", nil)
	if err == nil {
		t.Fatal("expected bad line to be rejected")
	}
	if !strings.Contains(err.Error(), "testdata/bad.jsonl:2:") {
		t.Fatalf("error should name file and line 2: %v", err)
	}
}

func TestUnknownKindsPassThroughInOrder(t *testing.T) {
	in, err := ReadFile("testdata/mixed.jsonl", nil)
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

func TestTwoFlagsSameKindError(t *testing.T) {
	_, err := Assemble(nil, []From{
		{Name: "a", Kinds: []string{"cost"}, File: "testdata/from-cost.jsonl"},
		{Name: "b", Kinds: []string{"cost"}, File: "testdata/from-cost.jsonl"},
	})
	if err == nil {
		t.Fatal("expected conflict between flags")
	}
	if !strings.Contains(err.Error(), `"cost"`) {
		t.Fatalf("error should name the kind: %v", err)
	}
}

func TestPipePlusFlagSameKindError(t *testing.T) {
	stdin := strings.NewReader(`{"kind":"cost","id":"pipe","csp":"aws","management_account":"111111111111","granularity":"monthly","period":"2026-09","metric":"UnblendedCost","amount":1,"currency":"USD","estimated":false,"account_id":"200000000001","service":"AmazonEC2","app_tag":"","tenant":"testtnt"}` + "\n")
	_, err := Assemble(stdin, []From{
		{Name: "pipe", Kinds: []string{"cost"}, File: "-"},
		{Name: "file", Kinds: []string{"cost"}, File: "testdata/from-cost.jsonl"},
	})
	if err == nil {
		t.Fatal("expected conflict between stdin and file")
	}
	if !strings.Contains(err.Error(), `"cost"`) {
		t.Fatalf("error should name the kind: %v", err)
	}
	if !strings.Contains(err.Error(), "stdin") && !strings.Contains(err.Error(), "--from-") {
		t.Fatalf("error should name both inputs: %v", err)
	}
}

func TestAssembleFileKeepsUndeclaredKinds(t *testing.T) {
	recs, err := Assemble(nil, []From{
		{Name: "accounts", Kinds: []string{"ou", "account"}, File: "testdata/mixed.jsonl"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var costCount int
	for _, r := range recs {
		if r.Kind() == "cost" {
			costCount++
		}
	}
	if costCount != 2 {
		t.Fatalf("want 2 cost records pass-through from file, got %d in %v", costCount, kindsOf(recs))
	}
}

func TestAssembleIgnoresOpenPipeWhenOnlyFileFlags(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	done := make(chan error, 1)
	go func() {
		_, err := Assemble(r, []From{{
			Name:  "cost",
			Kinds: []string{"cost"},
			File:  "testdata/from-cost.jsonl",
		}})
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Assemble blocked on open stdin pipe")
	}
}

func TestAssembleMixFileAndStdinFlag(t *testing.T) {
	stdin := strings.NewReader(`{"kind":"cost","id":"pipe-cost","csp":"aws","management_account":"111111111111","granularity":"monthly","period":"2026-09","metric":"UnblendedCost","amount":1,"currency":"USD","estimated":false,"account_id":"200000000001","service":"AmazonEC2","app_tag":"","tenant":"testtnt"}` + "\n")
	recs, err := Assemble(stdin, []From{
		{Name: "accounts", Kinds: []string{"ou", "account"}, File: "testdata/mixed.jsonl"},
		{Name: "cost", Kinds: []string{"cost"}, File: "-"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var hasAccount, hasPipeCost bool
	var fileCosts int
	for _, r := range recs {
		if r.Kind() == "account" {
			hasAccount = true
		}
		if r.Kind() == "cost" {
			if r["id"] == "pipe-cost" {
				hasPipeCost = true
			}
			if r["id"] == "c1" || r["id"] == "c2" {
				fileCosts++
			}
		}
	}
	if !hasAccount || !hasPipeCost || fileCosts != 2 {
		t.Fatalf("got %v", recs)
	}
}

func TestAssembleStdinPreservesInputOrder(t *testing.T) {
	f, err := os.Open("testdata/mixed.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	recs, err := Assemble(f, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantIDs := []string{"c1", "a1", "c2", "o1"}
	if len(recs) != len(wantIDs) {
		t.Fatalf("got %d records, want %d", len(recs), len(wantIDs))
	}
	for i, id := range wantIDs {
		if recs[i]["id"] != id {
			t.Fatalf("record %d id = %v, want %q (order: %v)", i, recs[i]["id"], id, kindsOf(recs))
		}
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
	out, err := Read(&buf, "mem", nil)
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
	recs, err := Assemble(nil, []From{{
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

func TestTenantIDRule(t *testing.T) {
	cases := []struct {
		id      string
		wantErr string
	}{
		{"testtnt", ""},
		{"acme-retail", ""},
		{"", "--tenant is required"},
		{"Acme", "invalid --tenant"},
		{"acme_retail", "invalid --tenant"},
		{"acme--retail", "invalid --tenant"},
	}
	for _, tc := range cases {
		err := CheckTenantID(tc.id)
		if tc.wantErr == "" {
			if err != nil {
				t.Errorf("CheckTenantID(%q) = %v, want nil", tc.id, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
			t.Errorf("CheckTenantID(%q) = %v, want %q", tc.id, err, tc.wantErr)
		}
	}
}

func TestCheckTaxonomyTenant(t *testing.T) {
	cases := []struct {
		name, fileTenant string
		want             []string
	}{
		{"same", "testtnt", nil},
		{"missing", "", []string{"tax.yaml", "missing tenant"}},
		{"malformed", "Test_Tnt", []string{"tax.yaml", "invalid tenant", "Test_Tnt"}},
		{"other tenant", "othertnt", []string{"tax.yaml", `"othertnt"`, `"testtnt"`}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckTaxonomyTenant("tax.yaml", tc.fileTenant, "testtnt")
			if tc.want == nil {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error")
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Fatalf("error %q should contain %q", err, w)
				}
			}
		})
	}
}

func TestReadOtherTenantFails(t *testing.T) {
	cases := []struct {
		name  string
		input string
		froms func(path string) []From
		want  []string
	}{
		{
			name: "declared kind from file",
			froms: func(path string) []From {
				return []From{{Name: "cost", Kinds: []string{"cost"}, File: path}}
			},
			want: []string{"other-tenant.jsonl:2:", `"othertnt"`, `"testtnt"`},
		},
		{
			name: "passed-through kind from file",
			froms: func(path string) []From {
				return []From{{Name: "accounts", Kinds: []string{"ou", "account"}, File: path}}
			},
			want: []string{"other-tenant.jsonl:2:", `"othertnt"`, `"testtnt"`},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := "testdata/other-tenant.jsonl"
			_, err := AssembleTenant(nil, "testtnt", tc.froms(path))
			if err == nil {
				t.Fatal("expected tenant error")
			}
			var te *TenantError
			if !errors.As(err, &te) {
				t.Fatalf("error %v is not a TenantError", err)
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Fatalf("error %q should contain %q", err, w)
				}
			}
		})
	}
}

func TestReadOtherTenantFromStdin(t *testing.T) {
	stdin := strings.NewReader(`{"kind":"note","tenant":"testtnt"}` + "\n" + `{"kind":"note","tenant":"othertnt"}` + "\n")
	_, err := AssembleTenant(stdin, "testtnt", nil)
	if err == nil {
		t.Fatal("expected tenant error")
	}
	for _, w := range []string{"stdin:2:", `"othertnt"`, `"testtnt"`} {
		if !strings.Contains(err.Error(), w) {
			t.Fatalf("error %q should contain %q", err, w)
		}
	}
}

func TestReadWithoutTenantFieldPasses(t *testing.T) {
	stdin := strings.NewReader(`{"kind":"note","id":"n1"}` + "\n")
	recs, err := AssembleTenant(stdin, "testtnt", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %v", recs)
	}
}

func TestSetTenantOwnLeavesPassThroughUnchanged(t *testing.T) {
	in := []Record{
		{"kind": "note", "id": "n1"},
		{"kind": "cost", "id": "c1", "tenant": "testtnt"},
	}
	own := []Record{Finding("warning", "x", "s", "m", "cmd"), {"kind": "allocation", "id": "a1"}}
	out := PassThrough(in, []string{"cost"}, own)
	SetTenantOwn(in, out, "testtnt")
	for _, r := range out {
		switch r["id"] {
		case "n1":
			if _, ok := r["tenant"]; ok {
				t.Fatalf("pass-through record was changed: %v", r)
			}
		default:
			if r["tenant"] != "testtnt" {
				t.Fatalf("own record without tenant: %v", r)
			}
		}
	}
}

func TestTenantFlagStopsBeforeRun(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"missing", nil, "tenant"},
		{"malformed", []string{"--tenant", "Bad_Tenant"}, "invalid --tenant"},
		{"empty", []string{"--tenant", ""}, "--tenant is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ran := false
			var tenant string
			cmd := &cobra.Command{
				Use:           "x",
				SilenceUsage:  true,
				SilenceErrors: true,
				RunE: func(cmd *cobra.Command, args []string) error {
					ran = true
					return nil
				},
			}
			AddTenantFlag(cmd, &tenant)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
			if ran {
				t.Fatal("RunE ran without a valid --tenant")
			}
		})
	}
}
