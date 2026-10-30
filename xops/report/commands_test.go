package report

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestReportCommandsTenant(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.jsonl")
	other := filepath.Join(dir, "other.jsonl")
	body := `{"kind":"alpha","name":"first","tenant":"testtnt"}` + "\n" + `{"kind":"alpha","name":"second","tenant":"othertnt"}` + "\n"
	if err := os.WriteFile(other, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cmds := []struct {
		name string
		cmd  func() *cobra.Command
		out  string
	}{
		{"table", tableCommand, ""},
		{"csv", csvCommand, filepath.Join(dir, "csv")},
		{"excel", excelCommand, filepath.Join(dir, "out.xlsx")},
	}
	for _, c := range cmds {
		run := func(args ...string) (string, error) {
			var out bytes.Buffer
			cmd := c.cmd()
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			cmd.SetOut(&out)
			cmd.SetErr(&bytes.Buffer{})
			if c.out != "" {
				args = append(args, "--out", c.out)
			}
			cmd.SetArgs(args)
			err := cmd.Execute()
			return out.String(), err
		}
		t.Run(c.name+" without tenant", func(t *testing.T) {
			for _, tenant := range [][]string{nil, {"--tenant", "Bad"}, {"--tenant", ""}} {
				out, err := run(append(tenant, "--from-cost", missing)...)
				if err == nil || !strings.Contains(err.Error(), "tenant") || strings.Contains(err.Error(), "missing.jsonl") {
					t.Fatalf("args %v: error = %v, want a --tenant error before reading", tenant, err)
				}
				if out != "" {
					t.Fatalf("wrote %q", out)
				}
			}
		})
		t.Run(c.name+" other tenant", func(t *testing.T) {
			_, err := run("--tenant", "testtnt", "--from-cost", other)
			if err == nil {
				t.Fatal("expected a record tenant error")
			}
			for _, w := range []string{other + ":2:", `"othertnt"`, `"testtnt"`} {
				if !strings.Contains(err.Error(), w) {
					t.Fatalf("error %q should contain %q", err, w)
				}
			}
		})
	}
}
