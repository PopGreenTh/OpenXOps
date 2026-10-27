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

// ConflictError is returned when the same kind arrives from two inputs.
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

// CheckKindsFrom returns the set of kinds to validate from explicit kinds or from --from-* declarations.
// A non-nil empty validateKinds skips schema checks (used in assembly tests with minimal fixtures).
func CheckKindsFrom(froms []From, validateKinds []string) map[string]struct{} {
	if validateKinds != nil && len(validateKinds) == 0 {
		return nil
	}
	if len(validateKinds) > 0 {
		return kindSet(validateKinds)
	}
	out := map[string]struct{}{}
	for _, f := range froms {
		for _, k := range f.Kinds {
			out[k] = struct{}{}
		}
	}
	return out
}

func kindSet(kinds []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, k := range kinds {
		out[k] = struct{}{}
	}
	return out
}

// Assemble reads inputs per D-030: stdin only when no --from-* flag is set, or when a flag's
// FILE is "-". Commands that pass only file paths never read stdin, even if it is an open pipe.
// validateKinds limits schema checks; when nil, every kind declared on froms is checked.
func Assemble(stdin io.Reader, froms []From, validateKinds []string) ([]Record, error) {
	checkKinds := CheckKindsFrom(froms, validateKinds)
	var active []From
	for _, f := range froms {
		if strings.TrimSpace(f.File) != "" {
			active = append(active, f)
		}
	}

	if len(active) == 0 {
		if stdin == nil || !isNonTTY(stdin) {
			return nil, nil
		}
		recs, err := Read(stdin, "stdin", checkKinds)
		if err != nil {
			return nil, err
		}
		sort.SliceStable(recs, func(i, j int) bool {
			return recs[i].Kind() < recs[j].Kind()
		})
		return recs, nil
	}

	stdinViaFlag := false
	for _, f := range active {
		if f.File == "-" {
			stdinViaFlag = true
			break
		}
	}

	var stdinRecs []Record
	if stdinViaFlag {
		if stdin == nil {
			return nil, fmt.Errorf("stdin: no input")
		}
		recs, err := Read(stdin, "stdin", checkKinds)
		if err != nil {
			return nil, err
		}
		sort.SliceStable(recs, func(i, j int) bool {
			return recs[i].Kind() < recs[j].Kind()
		})
		stdinRecs = recs
	}

	kindFrom := map[string]string{}
	var out []Record

	for _, f := range active {
		flagName := "--from-" + f.Name
		var recs []Record
		if f.File == "-" {
			for _, r := range stdinRecs {
				if kindDeclared(r.Kind(), f.Kinds) {
					recs = append(recs, r)
				}
			}
		} else {
			var err error
			allRecs, err := ReadFile(f.File, checkKinds)
			if err != nil {
				return nil, err
			}
			for _, r := range allRecs {
				if kindDeclared(r.Kind(), f.Kinds) {
					recs = append(recs, r)
				}
			}
		}
		seen := map[string]struct{}{}
		for _, r := range recs {
			k := r.Kind()
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			if prev, ok := kindFrom[k]; ok {
				return nil, &ConflictError{Kind: k, Pipe: prev, Flag: flagName}
			}
			kindFrom[k] = flagName
		}
		out = append(out, recs...)
	}
	return out, nil
}

func kindDeclared(kind string, kinds []string) bool {
	for _, k := range kinds {
		if k == kind {
			return true
		}
	}
	return false
}

func isNonTTY(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return true
	}
	return IsPipe(f)
}
