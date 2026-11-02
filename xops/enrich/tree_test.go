package enrich

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PopGreenTh/OpenXOps/record"
	"github.com/PopGreenTh/OpenXOps/record/urn"
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
			"type": "lzc", "parent_urn": "", "code": "main", "name": "Main", "ref_id": "111111111111", "path": "", "status": "", "csp": "aws"}},
		{"the root OU", "urn:tnt:testtnt:lzc:main:org:root", map[string]string{
			"type": "org", "parent_urn": "urn:tnt:testtnt:lzc:main", "code": "root", "name": "Root", "ref_id": "r-1", "path": "/", "status": ""}},
		{"a nested OU", "urn:tnt:testtnt:lzc:main:org:workloads/checkout", map[string]string{
			"type": "org", "parent_urn": "urn:tnt:testtnt:lzc:main:org:workloads", "code": "checkout", "name": "Checkout",
			"ref_id": "ou-2", "path": "/Workloads/Checkout"}},
		{"an account in the root OU", "urn:tnt:testtnt:lzc:main:org:root:acc:111111111111", map[string]string{
			"type": "acc", "parent_urn": "urn:tnt:testtnt:lzc:main:org:root", "code": "111111111111", "name": "management",
			"ref_id": "111111111111", "path": "/", "status": "ACTIVE"}},
		{"an account in a CamelCase OU", "urn:tnt:testtnt:lzc:main:org:workloads/checkout/non-prod:acc:200000000003", map[string]string{
			"type": "acc", "parent_urn": "urn:tnt:testtnt:lzc:main:org:workloads/checkout/non-prod", "code": "200000000003",
			"name": "chk-nonprod", "path": "/Workloads/Checkout/NonProd"}},
		{"the CamelCase OU", "urn:tnt:testtnt:lzc:main:org:workloads/checkout/non-prod", map[string]string{
			"type": "org", "code": "non-prod", "name": "NonProd", "ref_id": "ou-3"}},
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
		{"malformed tenant", []string{"--tenant", "ab", "--taxonomy", missing, "--from-accounts", missing}, []string{"invalid --tenant"}},
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

func TestLZTreeJoined(t *testing.T) {
	with := account("200000000003", "chk-nonprod", "ou-3", "/Workloads/Checkout/NonProd")
	with["joined_at"] = "2020-01-15T08:22:07Z"
	with["joined_method"] = "INVITED"
	without := account("111111111111", "management", "r-1", "/") // no joined_at, no joined_method
	in := []record.Record{ou("r-1", "Root", "", "/"), ou("ou-1", "Workloads", "r-1", "/Workloads"), without, with}
	m := byURN(LZTree("testtnt", testLZ, in))

	n := m["urn:tnt:testtnt:lzc:main:org:root:acc:111111111111"]
	if str(n, "joined_at") != "" || str(n, "joined_method") != "" {
		t.Errorf("an account record without the fields: joined_at = %q, joined_method = %q, want both empty", n["joined_at"], n["joined_method"])
	}
	for _, k := range []string{"joined_at", "joined_method"} {
		if v, ok := n[k]; !ok || v != "" {
			t.Errorf("node.%s = %v (present: %v), want an empty string", k, v, ok)
		}
	}
	n = m["urn:tnt:testtnt:lzc:main:org:workloads:acc:200000000003"]
	if n == nil {
		n = m["urn:tnt:testtnt:lzc:main:org:workloads/checkout/non-prod:acc:200000000003"]
	}
	if str(n, "joined_at") != "2020-01-15T08:22:07Z" || str(n, "joined_method") != "INVITED" {
		t.Errorf("account node joined = %q %q, want the account record's", n["joined_at"], n["joined_method"])
	}
	for u, n := range m {
		if str(n, "type") != "acc" && (n["joined_at"] != "" || n["joined_method"] != "") {
			t.Errorf("%s node %s has joined_at/joined_method %q/%q, want empty", str(n, "type"), u, n["joined_at"], n["joined_method"])
		}
	}
}

func TestDemoAccountNodesCarryNameAndJoined(t *testing.T) {
	in, err := record.ReadFile(mockAccounts(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	out, err := record.ReadFile(demoTreeFile, map[string]struct{}{"node": {}})
	if err != nil {
		t.Fatal(err)
	}
	nodes := map[string]record.Record{}
	for _, n := range out {
		if str(n, "type") == "acc" {
			nodes[str(n, "code")] = n
		}
	}
	checked := 0
	for _, r := range in {
		if r.Kind() != "account" {
			continue
		}
		n, ok := nodes[str(r, "id")]
		if !ok {
			t.Fatalf("no acc node for account %s", str(r, "id"))
		}
		for _, f := range []string{"name", "joined_at", "joined_method"} {
			if str(n, f) == "" || str(n, f) != str(r, f) {
				t.Errorf("account %s: node %s = %q, account record %q", str(r, "id"), f, str(n, f), str(r, f))
			}
		}
		checked++
	}
	if checked != len(nodes) || checked == 0 {
		t.Fatalf("checked %d accounts, %d acc nodes", checked, len(nodes))
	}
	for _, n := range out {
		if str(n, "type") != "acc" && (str(n, "joined_at") != "" || str(n, "joined_method") != "") {
			t.Errorf("%s node %s has join fields", str(n, "type"), str(n, "urn"))
		}
	}
}

// shortOUInventory has an OU named IT, whose slug `it` is shorter than a code may be.
func shortOUInventory() []record.Record {
	return []record.Record{
		ou("r-1", "Root", "", "/"),
		ou("ou-it", "IT", "r-1", "/IT"),
		account("111111111111", "management", "r-1", "/"),
	}
}

func TestOUNamedITKeepsItsShortCodeAndWarns(t *testing.T) {
	nodes := LZTree("testtnt", testLZ, shortOUInventory())
	n := byURN(nodes)["urn:tnt:testtnt:lzc:main:org:it"]
	if n == nil {
		t.Fatalf("no node for the IT OU in %v", nodes)
	}
	if str(n, "code") != "it" || str(n, "name") != "IT" || str(n, "ref_id") != "ou-it" || str(n, "path") != "/IT" {
		t.Fatalf("IT node = %v", n)
	}
	warnings, err := CheckNodes(nodes)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "urn:tnt:testtnt:lzc:main:org:it") || !strings.Contains(warnings[0], `"it"`) {
		t.Fatalf("warnings = %v, want exactly one naming the IT node and its code", warnings)
	}
}

func TestLZTreeCommandWarnsOnStderrOnce(t *testing.T) {
	var logs bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelWarn})))
	t.Cleanup(func() { slog.SetDefault(old) })

	var lines []string
	for _, r := range shortOUInventory() {
		var b bytes.Buffer
		if err := record.Write(&b, []record.Record{r}); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, strings.TrimSuffix(b.String(), "\n"))
	}
	accounts := writeFile(t, "accounts.jsonl", strings.Join(lines, "\n")+"\n")

	var out bytes.Buffer
	cmd := lzTreeCommand()
	cmd.SilenceUsage, cmd.SilenceErrors = true, true
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--tenant", "testtnt", "--taxonomy", demoTaxonomy, "--from-accounts", accounts})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(logs.String(), "level=WARN"); n != 1 {
		t.Fatalf("%d warnings, want 1:\n%s", n, logs.String())
	}
	if !strings.Contains(logs.String(), "urn:tnt:testtnt:lzc:main:org:it") {
		t.Fatalf("the warning does not name the node:\n%s", logs.String())
	}
	if !strings.Contains(out.String(), `"code":"it"`) || !strings.Contains(out.String(), "urn:tnt:testtnt:lzc:main:org:it") {
		t.Fatalf("the IT node was not written:\n%s", out.String())
	}
}

func TestCheckNodesStopsOnABadURN(t *testing.T) {
	long := strings.Repeat("x", 300)
	cases := []struct {
		name string
		lz   LZ
		inv  []record.Record
		want []string
	}{
		{"a space in the LZ code", LZ{Code: "my lz", Name: "Main", CSP: "aws", ManagementAccount: "111111111111"}, inventory(),
			[]string{"lzc node", "urn:tnt:testtnt:lzc:my lz", "URN rule"}},
		{"a URN over 250 characters", testLZ, []record.Record{ou("r-1", "Root", "", "/"), ou("ou-x", long, "r-1", "/"+long)},
			[]string{"org node", "URN rule"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CheckNodes(LZTree("testtnt", tc.lz, tc.inv))
			if err == nil {
				t.Fatal("expected an error")
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Fatalf("error %q should contain %q", err, w)
				}
			}
		})
	}

	// And the command stops without writing anything.
	taxonomy := writeFile(t, "tax.yaml", strings.Replace(mustRead(t, demoTaxonomy), "code: main", "code: my lz", 1))
	accounts := mockAccounts(t)
	_, err := execute(t, "enrich", "lz-tree", "--tenant", "testtnt", "--taxonomy", taxonomy, "--from-accounts", accounts)
	if err == nil || !strings.Contains(err.Error(), "URN rule") || !strings.Contains(err.Error(), "my lz") {
		t.Fatalf("error = %v, want the URN rule error naming the node", err)
	}
}

func TestCheckNodesEmptyNameStops(t *testing.T) {
	nodes := LZTree("testtnt", testLZ, inventory())
	for _, n := range nodes {
		if str(n, "type") == "acc" {
			n["name"] = ""
			break
		}
	}
	_, err := CheckNodes(nodes)
	if err == nil || !strings.Contains(err.Error(), "empty name") || !strings.Contains(err.Error(), "acc node") {
		t.Fatalf("error = %v, want an empty name error naming the account node", err)
	}
}

func TestEveryDemoNodeHasCodeURNAndName(t *testing.T) {
	out, err := record.ReadFile(demoTreeFile, map[string]struct{}{"node": {}})
	if err != nil {
		t.Fatal(err)
	}
	longest := ""
	for _, n := range out {
		for _, f := range []string{"code", "urn", "name", "ref_id"} {
			if str(n, f) == "" {
				t.Errorf("node %v has an empty %s", n["urn"], f)
			}
		}
		if _, has := n["id"]; has {
			t.Errorf("node %s still has id", str(n, "urn"))
		}
		if _, has := n["source_id"]; has {
			t.Errorf("node %s still has source_id", str(n, "urn"))
		}
		if !urn.ValidURN(str(n, "urn")) || !urn.ValidCode(str(n, "code")) {
			t.Errorf("node %s breaks the URN or code rule", str(n, "urn"))
		}
		if str(n, "type") == "org" && str(n, "code") != urn.Slug(str(n, "name")) {
			t.Errorf("OU %s: code %q, want the slug of its name %q", str(n, "urn"), str(n, "code"), urn.Slug(str(n, "name")))
		}
		if len(str(n, "urn")) > len(longest) {
			longest = str(n, "urn")
		}
	}
	if len(longest) < 40 {
		t.Fatalf("longest demo URN %q looks too short", longest)
	}
}
