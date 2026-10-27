package record

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// From is one --from-<name> input: the kinds it supplies and the file path.
type From struct {
	// Name is the flag suffix, e.g. "cost" for --from-cost.
	Name string
	// Kinds are the record kinds this flag is declared to read.
	Kinds []string
	// File is the path, "-" for stdin, or empty when the flag is unset.
	File string
}

// ConflictError is returned when the same kind arrives from both the pipe and a flag.
type ConflictError struct {
	Kind string
	Pipe string
	Flag string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("kind %q from both %s and %s", e.Kind, e.Pipe, e.Flag)
}

// IsPipe reports whether f is not a terminal (stdin is a pipe or file).
func IsPipe(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice == 0
}

// Assemble reads the pipe (when pipeOn) and every set From flag.
// Pipe records are sorted by kind. A kind present in both the pipe and a flag is an error.
func Assemble(pipe io.Reader, pipeOn bool, froms []From) ([]Record, error) {
	var stdinFlag string
	for _, f := range froms {
		if f.File == "-" {
			stdinFlag = "--from-" + f.Name
			break
		}
	}
	if pipeOn && stdinFlag != "" {
		return nil, fmt.Errorf("stdin is used by both the pipe and %s", stdinFlag)
	}

	var out []Record
	pipeKinds := map[string]struct{}{}
	if pipeOn && pipe != nil {
		recs, err := Read(pipe, "stdin")
		if err != nil {
			return nil, err
		}
		sort.SliceStable(recs, func(i, j int) bool {
			return recs[i].Kind() < recs[j].Kind()
		})
		for _, r := range recs {
			pipeKinds[r.Kind()] = struct{}{}
		}
		out = append(out, recs...)
	}

	for _, f := range froms {
		if strings.TrimSpace(f.File) == "" {
			continue
		}
		recs, err := ReadFile(f.File)
		if err != nil {
			return nil, err
		}
		flagName := "--from-" + f.Name
		seen := map[string]struct{}{}
		for _, r := range recs {
			k := r.Kind()
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			if _, ok := pipeKinds[k]; ok {
				return nil, &ConflictError{Kind: k, Pipe: "stdin", Flag: flagName}
			}
		}
		out = append(out, recs...)
	}
	return out, nil
}
