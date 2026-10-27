// Command check is the Verify step: build, vet, gofmt and test.
package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	os.Exit(run())
}

func run() int {
	steps := []struct {
		name string
		fn   func() error
	}{
		{"build", stepBuild},
		{"vet", stepVet},
		{"gofmt", stepGofmt},
		{"test", stepTest},
	}
	failed := false
	for _, s := range steps {
		if err := s.fn(); err != nil {
			fmt.Printf("%s FAIL\n", s.name)
			fmt.Fprintln(os.Stderr, err)
			failed = true
			continue
		}
		fmt.Printf("%s PASS\n", s.name)
	}
	if failed {
		return 1
	}
	return 0
}

func stepBuild() error {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	if err := os.MkdirAll("bin", 0o755); err != nil {
		return err
	}
	if err := runCmd("go", "build", "-o", filepath.Join("bin", "xops"+ext), "./xops"); err != nil {
		return err
	}
	return runCmd("go", "build", "-o", filepath.Join("bin", "finops"+ext), "./finops")
}

func stepVet() error {
	return runCmd("go", "vet", "./...")
}

func stepGofmt() error {
	var listed []string
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "bin", "data", "vendor":
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		// Windows checkouts may use CRLF; gofmt always writes LF.
		src = bytes.ReplaceAll(src, []byte("\r\n"), []byte("\n"))
		src = bytes.ReplaceAll(src, []byte("\r"), []byte("\n"))
		formatted, err := format.Source(src)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if !bytes.Equal(src, formatted) {
			listed = append(listed, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(listed) > 0 {
		return fmt.Errorf("unformatted files:\n%s", strings.Join(listed, "\n"))
	}
	return nil
}

func stepTest() error {
	return runCmd("go", "test", "./...")
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, out)
	}
	return nil
}
