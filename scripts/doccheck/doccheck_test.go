package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoccheckOK(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	if err := Run(root); err != nil {
		t.Fatal(err)
	}
}

func TestDoccheckFailures(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		subdir string
		want   string
	}{
		{"missing record field", "missing-field", "missing field"},
		{"missing finding", "missing-finding", "missing finding code"},
		{"missing command", "missing-command", "missing section for command"},
		{"broken link", "broken-link", "broken link"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(root, "scripts/doccheck/testdata", tc.subdir)
			err := Run(dir)
			if err == nil {
				t.Fatalf("expected error in %s", dir)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q should contain %q", err.Error(), tc.want)
			}
		})
	}
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
