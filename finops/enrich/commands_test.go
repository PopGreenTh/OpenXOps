package enrich

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PopGreenTh/OpenXOps/record"
	"github.com/spf13/cobra"
)

const (
	demoTaxonomy = "testdata/demo-taxonomy.yaml"
	demoAccounts = "testdata/demo-accounts.jsonl"
)

// demoCost is one dedicated and one shared account's cost, each with one project_tag_cost.
const demoCost = `{"kind":"cost","account_id":"200000000001","amount":42.5,"app_tag":"","csp":"aws","currency":"USD","estimated":false,"granularity":"monthly","management_account":"111111111111","metric":"UnblendedCost","period":"2026-09","service":"AmazonEC2","tenant":"testtnt"}
{"kind":"cost","account_id":"200000000002","amount":100,"app_tag":"","csp":"aws","currency":"USD","estimated":false,"granularity":"monthly","management_account":"111111111111","metric":"UnblendedCost","period":"2026-09","service":"AmazonEC2","tenant":"testtnt"}
{"kind":"project_tag_cost","account_id":"200000000001","amount":42.5,"csp":"aws","currency":"USD","estimated":false,"granularity":"monthly","management_account":"111111111111","metric":"UnblendedCost","period":"2026-09","project_tag":"","tenant":"testtnt"}
`

func execute(t *testing.T, cmd *cobra.Command, args ...string) ([]record.Record, error) {
	t.Helper()
	var out bytes.Buffer
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		if out.Len() != 0 {
			t.Fatalf("command failed but wrote %q", out.String())
		}
		return nil, err
	}
	recs, err := record.Read(&out, "stdout", nil)
	if err != nil {
		t.Fatal(err)
	}
	return recs, nil
}

func writeFile(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// otherTenantTaxonomy is the demo taxonomy for tenant othertnt.
func otherTenantTaxonomy(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(demoTaxonomy)
	if err != nil {
		t.Fatal(err)
	}
	body := strings.Replace(string(data), "tenant: testtnt", "tenant: othertnt", 1)
	return writeFile(t, "other.yaml", body)
}

func TestCommandsWithoutTenantFailBeforeReading(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.jsonl")
	cases := []struct {
		name string
		cmd  func() *cobra.Command
		args []string
	}{
		{"validate", validateCommand, []string{"--taxonomy", missing}},
		{"map", mapCommand, []string{"--taxonomy", missing, "--from-accounts", missing}},
		{"share", shareCommand, []string{"--taxonomy", missing, "--from-cost", missing}},
	}
	for _, tc := range cases {
		for _, tenant := range [][]string{nil, {"--tenant", "Bad_Tenant"}, {"--tenant", ""}} {
			t.Run(tc.name, func(t *testing.T) {
				_, err := execute(t, tc.cmd(), append(tenant, tc.args...)...)
				if err == nil || !strings.Contains(err.Error(), "tenant") {
					t.Fatalf("error = %v, want a --tenant error", err)
				}
				if strings.Contains(err.Error(), "missing.jsonl") {
					t.Fatalf("command read a file before checking --tenant: %v", err)
				}
			})
		}
	}
}

func TestCommandsRejectTaxonomyForOtherTenant(t *testing.T) {
	other := otherTenantTaxonomy(t)
	noTenant := writeFile(t, "none.yaml", "business_units: []\n")
	cases := []struct {
		name string
		cmd  func() *cobra.Command
		args []string
	}{
		{"validate", validateCommand, nil},
		{"map", mapCommand, []string{"--from-accounts", demoAccounts}},
		{"share", shareCommand, []string{"--from-cost", demoAccounts}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := execute(t, tc.cmd(), append([]string{"--tenant", "testtnt", "--taxonomy", other}, tc.args...)...)
			if err == nil {
				t.Fatal("expected a taxonomy tenant error")
			}
			for _, w := range []string{other, `"othertnt"`, `"testtnt"`} {
				if !strings.Contains(err.Error(), w) {
					t.Fatalf("error %q should contain %q", err, w)
				}
			}
			_, err = execute(t, tc.cmd(), append([]string{"--tenant", "testtnt", "--taxonomy", noTenant}, tc.args...)...)
			if err == nil || !strings.Contains(err.Error(), "missing tenant") {
				t.Fatalf("error = %v, want missing tenant", err)
			}
		})
	}
}

func TestCommandsRejectRecordFromOtherTenant(t *testing.T) {
	accounts, err := os.ReadFile(demoAccounts)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(accounts), "\n"), "\n")
	lines[2] = strings.Replace(lines[2], `"tenant":"testtnt"`, `"tenant":"othertnt"`, 1)
	otherAccounts := writeFile(t, "accounts.jsonl", strings.Join(lines, "\n")+"\n")
	// A kind no command declares still has its tenant checked.
	passThrough := writeFile(t, "cost.jsonl", `{"kind":"note","tenant":"testtnt"}`+"\n"+`{"kind":"note","tenant":"othertnt"}`+"\n")

	cases := []struct {
		name, file string
		line       string
		cmd        func() *cobra.Command
		args       []string
	}{
		{"map declared kind", otherAccounts, ":3:", mapCommand, []string{"--from-accounts", otherAccounts}},
		{"map passed-through kind", passThrough, ":2:", mapCommand, []string{"--from-accounts", demoAccounts, "--from-cost", passThrough}},
		{"share passed-through kind", passThrough, ":2:", shareCommand, []string{"--from-cost", passThrough}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := execute(t, tc.cmd(), append([]string{"--tenant", "testtnt", "--taxonomy", demoTaxonomy}, tc.args...)...)
			if err == nil {
				t.Fatal("expected a record tenant error")
			}
			for _, w := range []string{tc.file + tc.line, `"othertnt"`, `"testtnt"`} {
				if !strings.Contains(err.Error(), w) {
					t.Fatalf("error %q should contain %q", err, w)
				}
			}
		})
	}
}

func TestEveryRecordEnrichWritesHasTenant(t *testing.T) {
	cost := writeFile(t, "cost.jsonl", demoCost)
	mapped, err := execute(t, mapCommand(), "--tenant", "testtnt", "--taxonomy", demoTaxonomy, "--from-accounts", demoAccounts, "--from-cost", cost)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := record.Write(&buf, mapped); err != nil {
		t.Fatal(err)
	}
	mapOut := writeFile(t, "map.jsonl", buf.String())
	cases := []struct {
		name  string
		cmd   func() *cobra.Command
		args  []string
		kinds []string
	}{
		{"validate", validateCommand, nil, nil},
		{"map", mapCommand, []string{"--from-accounts", demoAccounts, "--from-cost", cost}, []string{"allocation", "cost", "finding"}},
		{"share", shareCommand, []string{"--from-cost", mapOut}, []string{"allocation", "cost", "finding"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recs, err := execute(t, tc.cmd(), append([]string{"--tenant", "testtnt", "--taxonomy", demoTaxonomy}, tc.args...)...)
			if err != nil {
				t.Fatal(err)
			}
			seen := map[string]bool{}
			for i, r := range recs {
				seen[r.Kind()] = true
				if r["tenant"] != "testtnt" {
					t.Fatalf("record %d has tenant %v: %v", i+1, r["tenant"], r)
				}
			}
			for _, k := range tc.kinds {
				if !seen[k] {
					t.Fatalf("no %q records in %v", k, seen)
				}
			}
		})
	}

	t.Run("pass-through unchanged", func(t *testing.T) {
		in := []record.Record{{"kind": "note", "id": "n1"}}
		out := Map(Taxonomy{}, in)
		record.SetTenantOwn(in, out, "testtnt")
		if len(out) != 1 {
			t.Fatalf("out = %v", out)
		}
		if _, ok := out[0]["tenant"]; ok {
			t.Fatalf("pass-through record was changed: %v", out[0])
		}
	})
}
