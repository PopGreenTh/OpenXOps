package enrich_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMapMixedPipeAndAccountsFile(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Join(root, "..", "..")
	tax := filepath.Join(repoRoot, "finops", "enrich", "testdata", "demo-taxonomy.yaml")
	accounts := filepath.Join(repoRoot, "finops", "enrich", "testdata", "demo-accounts.jsonl")

	cost := exec.Command("go", "run", filepath.Join(repoRoot, "finops"), "sense", "aws-cost", "--mock")
	cost.Dir = repoRoot
	cost.Env = append(os.Environ(), "GOTOOLCHAIN=local")

	mapCmd := exec.Command("go", "run", filepath.Join(repoRoot, "finops"), "enrich", "map",
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
	if !strings.Contains(mapOut.String(), `"kind":"allocation"`) {
		t.Fatalf("map output missing allocation records")
	}
	if !strings.Contains(mapOut.String(), `"kind":"cost"`) {
		t.Fatalf("map output missing cost records from pipe")
	}
}
