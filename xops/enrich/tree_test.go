package enrich

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PopGreenTh/OpenXOps/record"
	"github.com/PopGreenTh/OpenXOps/xops/sense"
	"github.com/spf13/cobra"
)

const (
	demoTaxonomy = "testdata/demo-taxonomy.yaml"
	demoTreeFile = "testdata/demo-tree.jsonl"
)

func ou(id, name, parent, path string) record.Record {
	return record.Record{"kind": "ou", "csp": "aws", "org_id": "o-main", "management_account": "111111111111",
		"id": id, "name": name, "parent_id": parent, "path": path, "id_path": id, "has_children": true, "tenant": "testtnt"}
}

func account(id, name, parent, ouPath string) record.Record {
	return record.Record{"kind": "account", "csp": "aws", "org_id": "o-main", "management_account": "111111111111",
		"id": id, "name": name, "email": name + "@example.com", "status": "ACTIVE", "parent_id": parent,
		"ou_path": ouPath, "ou_id_path": parent, "tenant": "testtnt"}
}

var testLZ = LZ{Code: "main", Name: "Main", CSP: "aws", ManagementAccount: "111111111111"}

func inventory() []record.Record {
	return []record.Record{
		ou("r-1", "Root", "", "/"),
		ou("ou-1", "Workloads", "r-1", "/Workloads"),
		ou("ou-2", "Checkout", "ou-1", "/Workloads/Checkout"),
		ou("ou-3", "NonProd", "ou-2", "/Workloads/Checkout/NonProd"),
		account("111111111111", "management", "r-1", "/"),
		account("200000000003", "chk-nonprod", "ou-3", "/Workloads/Checkout/NonProd"),
	}
}

func byURN(nodes []record.Record) map[string]record.Record {
	m := map[string]record.Record{}
	for _, n := range nodes {
		m[str(n, "urn")] = n
	}
	return m
}

func TestLZTreeNodes(t *testing.T) {
	nodes := LZTree("testtnt", testLZ, inventory())
	if len(nodes) != 7 {
		t.Fatalf("got %d nodes, want 7 (LZ, 4 OUs, 2 accounts)", len(nodes))
	}
	m := byURN(nodes)
	cases := []struct {
		name, urn string
		want      map[string]string
	}{
		{"the LZ", "urn:tnt:testtnt:lzc:main", map[string]string{
			"type": "lzc", "parent_urn": "", "id": "main", "name": "Main", "source_id": "111111111111", "path": "", "status": "", "csp": "aws"}},
		{"the root OU", "urn:tnt:testtnt:lzc:main:org:root", map[string]string{
			"type": "org", "parent_urn": "urn:tnt:testtnt:lzc:main", "id": "root", "name": "Root", "source_id": "r-1", "path": "/", "status": ""}},
		{"a nested OU", "urn:tnt:testtnt:lzc:main:org:workloads/checkout", map[string]string{
			"type": "org", "parent_urn": "urn:tnt:testtnt:lzc:main:org:workloads", "id": "workloads/checkout", "name": "Checkout",
			"source_id": "ou-2", "path": "/Workloads/Checkout"}},
		{"an account in the root OU", "urn:tnt:testtnt:lzc:main:org:root:acc:111111111111", map[string]string{
			"type": "acc", "parent_urn": "urn:tnt:testtnt:lzc:main:org:root", "id": "111111111111", "name": "management",
			"source_id": "111111111111", "path": "/", "status": "ACTIVE"}},
		{"an account in a CamelCase OU", "urn:tnt:testtnt:lzc:main:org:workloads/checkout/non-prod:acc:200000000003", map[string]string{
			"type": "acc", "parent_urn": "urn:tnt:testtnt:lzc:main:org:workloads/checkout/non-prod", "id": "200000000003",
			"name": "chk-nonprod", "path": "/Workloads/Checkout/NonProd"}},
		{"the CamelCase OU", "urn:tnt:testtnt:lzc:main:org:workloads/checkout/non-prod", map[string]string{
			"type": "org", "id": "workloads/checkout/non-prod", "name": "NonProd", "source_id": "ou-3"}},
	}
	for _, tc := range cases {
		n, ok := m[tc.urn]
		if !ok {
			t.Errorf("%s: no node with urn %s", tc.name, tc.urn)
			continue
		}
		for k, want := range tc.want {
			if got := str(n, k); got != want {
				t.Errorf("%s: %s = %q, want %q", tc.name, k, got, want)
			}
		}
		if n["kind"] != "node" || n["tenant"] != "testtnt" {
			t.Errorf("%s: kind/tenant = %v/%v", tc.name, n["kind"], n["tenant"])
		}
	}
}

func TestLZTreeOrder(t *testing.T) {
	in := inventory()
	// Shuffle the input: the output order must not depend on it.
	in[0], in[len(in)-1] = in[len(in)-1], in[0]
	nodes := LZTree("testtnt", testLZ, in)
	var types []string
	var lastOU, lastAcc string
	for _, n := range nodes {
		types = append(types, str(n, "type"))
		switch str(n, "type") {
		case "org":
			if u := str(n, "urn"); u < lastOU {
				t.Errorf("OU %s sorts before %s", u, lastOU)
			} else {
				lastOU = u
			}
		case "acc":
			if u := str(n, "urn"); u < lastAcc {
				t.Errorf("account %s sorts before %s", u, lastAcc)
			} else {
				lastAcc = u
			}
		}
	}
	if got, want := strings.Join(types, ","), "lzc,org,org,org,org,acc,acc"; got != want {
		t.Fatalf("types = %s, want %s (LZ, then OUs, then accounts)", got, want)
	}
}

func TestLZTreeIgnoresOtherKinds(t *testing.T) {
	in := append(inventory(), record.Finding("warning", "ou-path-ambiguous", "/x", "m", "xops sense aws-accounts"), record.Record{"kind": "cost"})
	nodes := LZTree("testtnt", testLZ, in)
	if len(nodes) != 7 {
		t.Fatalf("got %d nodes, want 7", len(nodes))
	}
	for _, n := range nodes {
		if n.Kind() != "node" {
			t.Fatalf("output has a %q record, want only node records", n.Kind())
		}
	}
}

func TestCheckManagement(t *testing.T) {
	in := inventory()
	if err := CheckManagement("accounts.jsonl", in, testLZ); err != nil {
		t.Fatal(err)
	}
	in[4]["management_account"] = "999999999999"
	err := CheckManagement("accounts.jsonl", in, testLZ)
	if err == nil {
		t.Fatal("expected an error for another management account")
	}
	for _, w := range []string{"accounts.jsonl:5:", `"999999999999"`, `"111111111111"`} {
		if !strings.Contains(err.Error(), w) {
			t.Fatalf("error %q should contain %q", err, w)
		}
	}
}

func execute(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := record.Root("xops", "")
	root.AddCommand(sense.Commands()...)
	root.AddCommand(Commands()...)
	return executeOn(t, root, args...)
}

func executeOn(t *testing.T, root *cobra.Command, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	root.SilenceUsage = true
	root.SilenceErrors = true
	root.SetContext(context.Background())
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		if out.Len() != 0 {
			t.Fatalf("command failed but wrote %q", out.String())
		}
		return "", err
	}
	return out.String(), nil
}

func writeFile(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// mockAccounts runs aws-accounts --mock and returns its output in a file.
func mockAccounts(t *testing.T) string {
	t.Helper()
	out, err := execute(t, "sense", "aws-accounts", "--tenant", "testtnt", "--mock")
	if err != nil {
		t.Fatal(err)
	}
	return writeFile(t, "accounts.jsonl", out)
}

func TestDemoTreeGolden(t *testing.T) {
	want, err := os.ReadFile(demoTreeFile)
	if err != nil {
		t.Fatal(err)
	}
	accounts := mockAccounts(t)
	for run := 1; run <= 2; run++ {
		got, err := execute(t, "enrich", "lz-tree", "--tenant", "testtnt", "--taxonomy", demoTaxonomy, "--from-accounts", accounts)
		if err != nil {
			t.Fatal(err)
		}
		if got != string(want) {
			t.Fatalf("run %d: output differs from %s", run, demoTreeFile)
		}
	}
}

func TestDemoTreeCoversEveryInputRecord(t *testing.T) {
	accounts := mockAccounts(t)
	in, err := record.ReadFile(accounts, nil)
	if err != nil {
		t.Fatal(err)
	}
	out, err := record.ReadFile(demoTreeFile, map[string]struct{}{"node": {}})
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, n := range out {
		counts[str(n, "type")]++
		if n["tenant"] != "testtnt" || n.Kind() != "node" {
			t.Fatalf("bad node %v", n)
		}
	}
	var ous, accs int
	for _, r := range in {
		switch r.Kind() {
		case "ou":
			ous++
		case "account":
			accs++
		}
	}
	if counts["lzc"] != 1 || counts["org"] != ous || counts["acc"] != accs {
		t.Fatalf("nodes %v for %d OUs and %d accounts", counts, ous, accs)
	}
}

func TestLZTreeCommandErrors(t *testing.T) {
	accounts := mockAccounts(t)
	otherTenant := writeFile(t, "other.yaml", strings.Replace(mustRead(t, demoTaxonomy), "tenant: testtnt", "tenant: othertnt", 1))
	twoLZs := writeFile(t, "two.yaml", mustRead(t, demoTaxonomy)+"  - code: other\n    name: Other\n    csp: aws\n    currency: USD\n    management_account: \"222222222222\"\n")
	otherLZ := writeFile(t, "otherlz.yaml", strings.Replace(mustRead(t, demoTaxonomy), `"111111111111"`, `"999999999999"`, 1))
	missing := filepath.Join(t.TempDir(), "missing.jsonl")

	cases := []struct {
		name string
		args []string
		want []string
	}{
		{"no tenant", []string{"--taxonomy", missing, "--from-accounts", missing}, []string{"tenant"}},
		{"malformed tenant", []string{"--tenant", "Bad_Tenant", "--taxonomy", missing, "--from-accounts", missing}, []string{"invalid --tenant"}},
		{"taxonomy for another tenant", []string{"--tenant", "testtnt", "--taxonomy", otherTenant, "--from-accounts", accounts}, []string{otherTenant, `"othertnt"`, `"testtnt"`}},
		{"taxonomy with two LZs", []string{"--tenant", "testtnt", "--taxonomy", twoLZs, "--from-accounts", accounts}, []string{"2 LZs", "exactly one"}},
		{"records of another management account", []string{"--tenant", "testtnt", "--taxonomy", otherLZ, "--from-accounts", accounts}, []string{accounts + ":1:", `"999999999999"`, `"111111111111"`}},
		{"no input", []string{"--tenant", "testtnt", "--taxonomy", demoTaxonomy, "--from-accounts", writeFile(t, "empty.jsonl", `{"kind":"note","tenant":"testtnt"}`+"\n")}, []string{"ou and account records"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := execute(t, append([]string{"enrich", "lz-tree"}, tc.args...)...)
			if err == nil {
				t.Fatal("expected an error")
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Fatalf("error %q should contain %q", err, w)
				}
			}
			if strings.Contains(err.Error(), "missing.jsonl") {
				t.Fatalf("command read a file before checking --tenant: %v", err)
			}
		})
	}
}

func TestLZTreeRejectsRecordFromOtherTenant(t *testing.T) {
	accounts := mustRead(t, "../sense/testdata/aws-accounts.jsonl")
	lines := strings.Split(strings.TrimSuffix(accounts, "\n"), "\n")
	lines[3] = strings.Replace(lines[3], `"tenant":"testtnt"`, `"tenant":"othertnt"`, 1)
	path := writeFile(t, "accounts.jsonl", strings.Join(lines, "\n")+"\n")
	_, err := execute(t, "enrich", "lz-tree", "--tenant", "testtnt", "--taxonomy", demoTaxonomy, "--from-accounts", path)
	if err == nil {
		t.Fatal("expected a tenant error")
	}
	for _, w := range []string{path + ":4:", `"othertnt"`, `"testtnt"`} {
		if !strings.Contains(err.Error(), w) {
			t.Fatalf("error %q should contain %q", err, w)
		}
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
