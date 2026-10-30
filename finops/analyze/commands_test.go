package analyze

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PopGreenTh/OpenXOps/record"
)

const (
	beforeMap = `{"kind":"account","csp":"aws","email":"a@example.com","id":"200000000001","management_account":"111111111111","name":"a","org_id":"o-main","ou_id_path":"r-1","ou_path":"/","parent_id":"r-1","status":"ACTIVE","tenant":"testtnt"}
{"kind":"allocation","account_id":"200000000001","account_name":"a","app":"APP-CHK","app_rule":"ou:/","business_unit":"BU-RETAIL","cost_center":"CC-ECOM","environment":"ENV-PROD","environment_rule":"ou:/","lz":"LZ-MAIN","ou_path":"/","project":"PRJ-SHOP","shared":false,"tenant":"testtnt"}
`
	nowMap = `{"kind":"account","csp":"aws","email":"a@example.com","id":"200000000001","management_account":"111111111111","name":"a","org_id":"o-main","ou_id_path":"r-1","ou_path":"/","parent_id":"r-1","status":"ACTIVE","tenant":"testtnt"}
{"kind":"allocation","account_id":"200000000001","account_name":"a","app":"APP-CHK","app_rule":"ou:/","business_unit":"BU-RETAIL","cost_center":"CC-ECOM","environment":"ENV-NONPROD","environment_rule":"ou:/","lz":"LZ-MAIN","ou_path":"/","project":"PRJ-SHOP","shared":false,"tenant":"testtnt"}
{"kind":"note","id":"n1"}
`
)

func writeFile(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func runChanges(t *testing.T, args ...string) ([]record.Record, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := changesCommand()
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

func TestChangesWithoutTenantFailsBeforeReading(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.jsonl")
	for _, tenant := range [][]string{nil, {"--tenant", "Bad"}, {"--tenant", ""}} {
		_, err := runChanges(t, append(tenant, "--before", missing, "--from-map", missing)...)
		if err == nil || !strings.Contains(err.Error(), "tenant") || strings.Contains(err.Error(), "missing.jsonl") {
			t.Fatalf("args %v: error = %v, want a --tenant error before reading", tenant, err)
		}
	}
}

func TestChangesRejectsRecordFromOtherTenant(t *testing.T) {
	before := writeFile(t, "before.jsonl", beforeMap)
	now := writeFile(t, "now.jsonl", nowMap)
	otherBefore := writeFile(t, "other-before.jsonl", strings.Replace(beforeMap, `"ENV-PROD","environment_rule":"ou:/","lz":"LZ-MAIN","ou_path":"/","project":"PRJ-SHOP","shared":false,"tenant":"testtnt"`, `"ENV-PROD","environment_rule":"ou:/","lz":"LZ-MAIN","ou_path":"/","project":"PRJ-SHOP","shared":false,"tenant":"othertnt"`, 1))
	otherNow := writeFile(t, "other-now.jsonl", strings.Replace(nowMap, `{"kind":"note","id":"n1"}`, `{"kind":"note","id":"n1","tenant":"othertnt"}`, 1))
	cases := []struct {
		name, want string
		args       []string
	}{
		{"before file", otherBefore + ":2:", []string{"--before", otherBefore, "--from-map", now}},
		{"map file, passed-through kind", otherNow + ":3:", []string{"--before", before, "--from-map", otherNow}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runChanges(t, append([]string{"--tenant", "testtnt"}, tc.args...)...)
			if err == nil {
				t.Fatal("expected a record tenant error")
			}
			for _, w := range []string{tc.want, `"othertnt"`, `"testtnt"`} {
				if !strings.Contains(err.Error(), w) {
					t.Fatalf("error %q should contain %q", err, w)
				}
			}
		})
	}
}

func TestEveryChangeHasTenant(t *testing.T) {
	before := writeFile(t, "before.jsonl", beforeMap)
	now := writeFile(t, "now.jsonl", nowMap)
	recs, err := runChanges(t, "--tenant", "testtnt", "--before", before, "--from-map", now)
	if err != nil {
		t.Fatal(err)
	}
	changes := 0
	for i, r := range recs {
		switch r.Kind() {
		case "change":
			changes++
			if r["tenant"] != "testtnt" {
				t.Fatalf("record %d has tenant %v: %v", i+1, r["tenant"], r)
			}
		case "note":
			if _, ok := r["tenant"]; ok {
				t.Fatalf("pass-through record was changed: %v", r)
			}
		}
	}
	if changes == 0 {
		t.Fatalf("no change records in %v", recs)
	}
}
