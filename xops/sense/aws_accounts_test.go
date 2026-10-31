package sense

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/PopGreenTh/OpenXOps/record"
)

func TestAWSAccountsMockGolden(t *testing.T) {
	golden := filepath.Join("testdata", "aws-accounts.jsonl")
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join("..", "..")
	cmd := exec.Command("go", "run", "./xops", "sense", "aws-accounts", "--tenant", "testtnt", "--mock")
	cmd.Dir = root
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), want) {
		t.Fatalf("mock output differs from %s", golden)
	}
}

func TestAWSAccountsGoldenHasJoined(t *testing.T) {
	recs, err := record.ReadFile(filepath.Join("testdata", "aws-accounts.jsonl"), map[string]struct{}{"account": {}})
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for i, rec := range recs {
		if rec["kind"] != "account" {
			continue
		}
		n++
		for _, name := range []string{"joined_at", "joined_method"} {
			if s, _ := rec[name].(string); s == "" {
				t.Errorf("line %d: account %v has empty %s", i+1, rec["id"], name)
			}
		}
	}
	if n == 0 {
		t.Fatal("no account records")
	}
}
