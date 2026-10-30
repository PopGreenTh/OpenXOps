package enrich_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
)

func TestMixingPipelineMatchesFileSteps(t *testing.T) {
	repoRoot := repoRoot(t)
	tax := filepath.Join(repoRoot, "finops", "enrich", "testdata", "demo-taxonomy.yaml")
	accounts := filepath.Join(repoRoot, "finops", "enrich", "testdata", "demo-accounts.jsonl")
	tmp := t.TempDir()

	costOut := filepath.Join(tmp, "cost.jsonl")
	run(t, repoRoot, costOut, "go", "run", "./finops", "sense", "aws-cost", "--tenant", "testtnt", "--mock")

	mapFileOut := filepath.Join(tmp, "map-file.jsonl")
	run(t, repoRoot, mapFileOut, "go", "run", "./finops", "enrich", "map", "--tenant", "testtnt",
		"--taxonomy", tax,
		"--from-accounts", accounts,
		"--from-cost", costOut,
	)

	shareFileOut := filepath.Join(tmp, "share-file.jsonl")
	run(t, repoRoot, shareFileOut, "go", "run", "./finops", "enrich", "share", "--tenant", "testtnt",
		"--taxonomy", tax,
		"--from-cost", mapFileOut,
	)

	var costBuf bytes.Buffer
	cost := exec.Command("go", "run", "./finops", "sense", "aws-cost", "--tenant", "testtnt", "--mock")
	cost.Dir = repoRoot
	cost.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	cost.Stdout = &costBuf
	cost.Stderr = os.Stderr
	if err := cost.Run(); err != nil {
		t.Fatalf("aws-cost --mock: %v", err)
	}

	mapPipe := exec.Command("go", "run", "./finops", "enrich", "map", "--tenant", "testtnt",
		"--taxonomy", tax,
		"--from-accounts", accounts,
		"--from-cost", "-",
	)
	mapPipe.Dir = repoRoot
	mapPipe.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	mapPipe.Stdin = &costBuf
	var mapBuf bytes.Buffer
	mapPipe.Stdout = &mapBuf
	mapPipe.Stderr = os.Stderr
	if err := mapPipe.Run(); err != nil {
		t.Fatalf("map mixed pipe: %v", err)
	}

	sharePipe := exec.Command("go", "run", "./finops", "enrich", "share", "--tenant", "testtnt",
		"--taxonomy", tax,
		"--from-cost", "-",
	)
	sharePipe.Dir = repoRoot
	sharePipe.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	sharePipe.Stdin = &mapBuf
	var sharePipeOut bytes.Buffer
	sharePipe.Stdout = &sharePipeOut
	sharePipe.Stderr = os.Stderr
	if err := sharePipe.Run(); err != nil {
		t.Fatalf("share mixed pipe: %v", err)
	}

	fileBytes, err := os.ReadFile(shareFileOut)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(fileBytes, sharePipeOut.Bytes()) {
		t.Fatalf("pipe pipeline output differs from file steps (%d vs %d bytes)", len(fileBytes), sharePipeOut.Len())
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(root, "..", "..")
}

func run(t *testing.T, dir, outPath string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("%v: %v", args, err)
	}
	if err := os.WriteFile(outPath, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readJSONL(t *testing.T, path string) []map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return readJSONLBytes(t, data)
}

func readJSONLBytes(t *testing.T, data []byte) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal(line, &rec); err != nil {
			t.Fatal(err)
		}
		out = append(out, rec)
	}
	return out
}

func kindCounts(recs []map[string]any) map[string]int {
	c := map[string]int{}
	for _, r := range recs {
		k, _ := r["kind"].(string)
		c[k]++
	}
	return c
}

func countsEqual(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func TestMapMixedPipeAndAccountsFile(t *testing.T) {
	repoRoot := repoRoot(t)
	tax := filepath.Join(repoRoot, "finops", "enrich", "testdata", "demo-taxonomy.yaml")
	accounts := filepath.Join(repoRoot, "finops", "enrich", "testdata", "demo-accounts.jsonl")

	cost := exec.Command("go", "run", "./finops", "sense", "aws-cost", "--tenant", "testtnt", "--mock")
	cost.Dir = repoRoot
	cost.Env = append(os.Environ(), "GOTOOLCHAIN=local")

	mapCmd := exec.Command("go", "run", "./finops", "enrich", "map", "--tenant", "testtnt",
		"--taxonomy", tax,
		"--from-accounts", accounts,
		"--from-cost", "-",
	)
	mapCmd.Dir = repoRoot
	mapCmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")

	var costOut bytes.Buffer
	cost.Stdout = &costOut
	cost.Stderr = os.Stderr
	if err := cost.Run(); err != nil {
		t.Fatalf("aws-cost --mock: %v", err)
	}

	mapCmd.Stdin = &costOut
	var mapOut bytes.Buffer
	mapCmd.Stdout = &mapOut
	mapCmd.Stderr = os.Stderr
	if err := mapCmd.Run(); err != nil {
		t.Fatalf("enrich map mixed pipe: %v", err)
	}
	counts := kindCounts(readJSONLBytes(t, mapOut.Bytes()))
	for _, k := range []string{"allocation", "cost"} {
		if counts[k] == 0 {
			t.Fatalf("map output missing %q records: %v", k, counts)
		}
	}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
}
