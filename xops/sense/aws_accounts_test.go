package sense

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
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
