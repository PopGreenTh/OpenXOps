package report

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PopGreenTh/OpenXOps/record"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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

const (
	demoTreeJSONL = "../enrich/testdata/demo-tree.jsonl"
	demoTreeYAML  = "testdata/demo-tree.yaml"
)

func runTree(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := treeCommand()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestReportTreeGolden(t *testing.T) {
	want, err := os.ReadFile(demoTreeYAML)
	if err != nil {
		t.Fatal(err)
	}
	for run := 1; run <= 2; run++ {
		got, err := runTree(t, "--tenant", "testtnt", "--from-node", demoTreeJSONL)
		if err != nil {
			t.Fatal(err)
		}
		if got != string(want) {
			t.Fatalf("run %d: stdout differs from %s", run, demoTreeYAML)
		}
	}
	out := filepath.Join(t.TempDir(), "tree.yaml")
	for run := 1; run <= 2; run++ {
		if _, err := runTree(t, "--tenant", "testtnt", "--from-node", demoTreeJSONL, "--out", out); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("run %d: --out file differs from %s", run, demoTreeYAML)
		}
	}
}

func TestReportTreeReplacesOutFile(t *testing.T) {
	out := filepath.Join(t.TempDir(), "tree.yaml")
	if err := os.WriteFile(out, []byte("old content that is longer than nothing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runTree(t, "--tenant", "testtnt", "--from-node", demoTreeJSONL, "--out", out); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(out)
	if strings.Contains(string(got), "old content") || !strings.HasPrefix(string(got), "tenant: testtnt\n") {
		t.Fatalf("--out was appended to, not replaced:\n%s", got)
	}
}

func TestReportTreeFailsWritingNothing(t *testing.T) {
	data, err := os.ReadFile(demoTreeJSONL)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	write := func(name string, ls []string) string {
		path := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(path, []byte(strings.Join(ls, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	noLZ := write("no-lz.jsonl", lines[1:])
	twoLZ := write("two-lz.jsonl", append(append([]string{}, lines...), strings.Replace(lines[0], "lzc:main", "lzc:other", 2)))
	dangling := write("dangling.jsonl", append([]string{lines[0]}, strings.Replace(lines[1], `"parent_urn":"urn:tnt:testtnt:lzc:main"`, `"parent_urn":"urn:tnt:testtnt:lzc:gone"`, 1)))

	cases := []struct {
		name, file string
		want       []string
	}{
		{"no lzc node", noLZ, []string{noLZ, "no lzc node"}},
		{"two lzc nodes", twoLZ, []string{twoLZ + ":21:", "urn:tnt:testtnt:lzc:other"}},
		{"dangling parent_urn", dangling, []string{dangling + ":2:", "urn:tnt:testtnt:lzc:gone"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "tree.yaml")
			stdout, err := runTree(t, "--tenant", "testtnt", "--from-node", tc.file, "--out", out)
			if err == nil {
				t.Fatal("expected an error")
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Fatalf("error %q should contain %q", err, w)
				}
			}
			if _, statErr := os.Stat(out); statErr == nil {
				t.Fatal("--out was written")
			}
			if stdout != "" {
				t.Fatalf("stdout = %q", stdout)
			}
			stdout, err = runTree(t, "--tenant", "testtnt", "--from-node", tc.file)
			if err == nil || stdout != "" {
				t.Fatalf("stdout run: err = %v, stdout = %q", err, stdout)
			}
		})
	}
}

func TestReportTreeTenant(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.jsonl")
	for _, tenant := range [][]string{nil, {"--tenant", "Bad_Tenant"}} {
		_, err := runTree(t, append(tenant, "--from-node", missing)...)
		if err == nil || !strings.Contains(err.Error(), "tenant") || strings.Contains(err.Error(), "missing.jsonl") {
			t.Fatalf("args %v: error = %v, want a --tenant error before reading", tenant, err)
		}
	}
	data, _ := os.ReadFile(demoTreeJSONL)
	other := filepath.Join(t.TempDir(), "other.jsonl")
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	lines[2] = strings.Replace(lines[2], `"tenant":"testtnt"`, `"tenant":"othertnt"`, 1)
	if err := os.WriteFile(other, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := runTree(t, "--tenant", "testtnt", "--from-node", other)
	if err == nil {
		t.Fatal("expected a record tenant error")
	}
	for _, w := range []string{other + ":3:", `"othertnt"`, `"testtnt"`} {
		if !strings.Contains(err.Error(), w) {
			t.Fatalf("error %q should contain %q", err, w)
		}
	}
}

func TestDemoYAMLAccountsCarryNameAndJoined(t *testing.T) {
	data, err := os.ReadFile(demoTreeYAML)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		LZ yamlNode `yaml:"lz"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	var accs []yamlNode
	var walk func(n yamlNode)
	walk = func(n yamlNode) {
		if n.Type == "acc" {
			accs = append(accs, n)
		} else if n.JoinedAt != "" || n.JoinedMethod != "" {
			t.Errorf("%s node %s shows join fields", n.Type, n.URN)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(doc.LZ)

	in, err := record.ReadFile("../sense/testdata/aws-accounts.jsonl", nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]record.Record{}
	for _, r := range in {
		if r.Kind() == "account" {
			want[r["id"].(string)] = r
		}
	}
	if len(accs) != len(want) || len(accs) == 0 {
		t.Fatalf("%d accounts in the YAML, %d account records", len(accs), len(want))
	}
	for _, a := range accs {
		r, ok := want[a.ID]
		if !ok {
			t.Fatalf("account %s is not in the account records", a.ID)
		}
		if a.Name != r["name"] || a.JoinedAt != r["joined_at"] || a.JoinedMethod != r["joined_method"] || a.JoinedAt == "" || a.JoinedMethod == "" {
			t.Errorf("account %s: YAML %q %q %q, record %v %v %v", a.ID, a.Name, a.JoinedAt, a.JoinedMethod, r["name"], r["joined_at"], r["joined_method"])
		}
	}
}

type yamlNode struct {
	Type         string     `yaml:"type"`
	ID           string     `yaml:"id"`
	Name         string     `yaml:"name"`
	URN          string     `yaml:"urn"`
	JoinedAt     string     `yaml:"joined_at"`
	JoinedMethod string     `yaml:"joined_method"`
	Children     []yamlNode `yaml:"children"`
}
