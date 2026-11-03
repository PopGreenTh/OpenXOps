package report

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PopGreenTh/OpenXOps/record"
	"github.com/PopGreenTh/OpenXOps/record/taxonomy"
	"github.com/PopGreenTh/OpenXOps/record/urn"
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
			for _, tenant := range [][]string{nil, {"--tenant", "ab"}, {"--tenant", ""}} {
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

func TestReportTreeFormatTaxonomy(t *testing.T) {
	out := filepath.Join(t.TempDir(), "tree.yaml")
	if _, err := runTree(t, "--tenant", "testtnt", "--from-node", demoTreeJSONL, "--format", "taxonomy", "--out", out); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(out)
	if !strings.Contains(string(got), "lz_conns:") {
		t.Fatalf("expected lz_conns in output:\n%s", got)
	}
	if _, err := runTree(t, "--tenant", "testtnt", "--from-node", demoTreeJSONL, "--format", "invalid"); err == nil {
		t.Fatal("expected error for invalid format")
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
	// The first node under the LZ gets a parent that is not there.
	child := 1
	for i := 1; i < len(lines); i++ {
		if strings.Contains(lines[i], `"parent_urn":"urn:tnt:testtnt:lzc:main"`) {
			child = i
			break
		}
	}
	dangling := write("dangling.jsonl", append([]string{lines[0]}, strings.Replace(lines[child], `"parent_urn":"urn:tnt:testtnt:lzc:main"`, `"parent_urn":"urn:tnt:testtnt:lzc:gone"`, 1)))

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
	for _, tenant := range [][]string{nil, {"--tenant", "ab"}} {
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
		r, ok := want[a.RefID]
		if !ok {
			t.Fatalf("account %s (%s) is not in the account records", a.RefID, a.Code)
		}
		if a.Name != r["name"] || a.JoinedAt != r["joined_at"] || a.JoinedMethod != r["joined_method"] || a.JoinedAt == "" || a.JoinedMethod == "" {
			t.Errorf("account %s: YAML %q %q %q, record %v %v %v", a.Code, a.Name, a.JoinedAt, a.JoinedMethod, r["name"], r["joined_at"], r["joined_method"])
		}
	}
}

type yamlNode struct {
	Type         string     `yaml:"type"`
	Code         string     `yaml:"code"`
	Name         string     `yaml:"name"`
	URN          string     `yaml:"urn"`
	RefID        string     `yaml:"ref_id"`
	JoinedAt     string     `yaml:"joined_at"`
	JoinedMethod string     `yaml:"joined_method"`
	Children     []yamlNode `yaml:"children"`
}

// TestTaxonomyScaffoldReadsBackWithTheSameURNs checks that the scaffold writes each OU's and
// account's code as its slug (or listed code), so reading it back passes the code rule and builds
// the URNs lz-tree wrote (D-059).
func TestTaxonomyScaffoldReadsBackWithTheSameURNs(t *testing.T) {
	out := filepath.Join(t.TempDir(), "scaffold.yaml")
	if _, err := runTree(t, "--tenant", "testtnt", "--from-node", demoTreeJSONL, "--format", "taxonomy", "--out", out); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	tax, err := taxonomy.Parse(data)
	if err != nil {
		t.Fatalf("the scaffold is not a taxonomy: %v", err)
	}
	nodes, err := record.ReadFile(demoTreeJSONL, map[string]struct{}{"node": {}})
	if err != nil {
		t.Fatal(err)
	}
	have := map[string]bool{}
	for _, n := range nodes {
		have[n["urn"].(string)] = true
	}
	lz := urn.Tenant("testtnt").Child("lzc", tax.LZs[0].Code)
	cloud := tax.Cloud(tax.LZs[0].Code)
	var accounts int
	for _, ou := range tax.LZs[0].OUPaths {
		if !urn.ValidCode(ou.Code) {
			t.Errorf("OU code %q breaks the code rule", ou.Code)
		}
		ouURN := cloud.OUURN(lz, taxonomy.DiscoveredOU{ID: ou.RefID, Path: ou.Path, Name: ou.Name})
		if !have[ouURN.String()] || ou.URN != ouURN.String() {
			t.Errorf("OU %s: written urn %q, built %q, in the tree: %v", ou.Path, ou.URN, ouURN, have[ouURN.String()])
		}
		for _, a := range ou.Accounts {
			accounts++
			if !urn.ValidCode(a.Code) {
				t.Errorf("account code %q breaks the code rule", a.Code)
			}
			built := cloud.AccountURN(lz, taxonomy.DiscoveredOU{ID: ou.RefID, Path: ou.Path, Name: ou.Name}, taxonomy.DiscoveredAccount{ID: a.RefID, Name: a.Name})
			if !have[built.String()] || a.URN != built.String() {
				t.Errorf("account %s: written urn %q, built %q, in the tree: %v", a.RefID, a.URN, built, have[built.String()])
			}
		}
	}
	if accounts == 0 {
		t.Fatal("the scaffold has no accounts")
	}
}
