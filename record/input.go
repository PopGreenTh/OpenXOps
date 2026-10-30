package record

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// From is one --from-<name> input: the kinds it supplies and the file path.
type From struct {
	// Name is the flag suffix, e.g. "cost" for --from-cost".
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

// CheckKindsFrom returns the set of kinds to validate from --from-* declarations.
func CheckKindsFrom(froms []From) map[string]struct{} {
	out := map[string]struct{}{}
	for _, f := range froms {
		for _, k := range f.Kinds {
			out[k] = struct{}{}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// Assemble reads inputs per D-030: stdin only when no --from-* flag is set, or when a flag's
// FILE is "-". Commands that pass only file paths never read stdin, even if it is an open pipe.
// Every record from each file or stdin is kept; schema checks and kind conflicts apply only to
// kinds declared on the flags. The same path is read once when two flags share it.
func Assemble(stdin io.Reader, froms []From) ([]Record, error) {
	return AssembleTenant(stdin, "", froms)
}

// AssembleTenant is Assemble, and when tenant is set every record read, of any kind, that has a
// tenant field must equal tenant (see ReadTenant).
func AssembleTenant(stdin io.Reader, tenant string, froms []From) ([]Record, error) {
	checkKinds := CheckKindsFrom(froms)
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
		return ReadTenant(stdin, "stdin", tenant, checkKinds)
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
		recs, err := ReadTenant(stdin, "stdin", tenant, checkKinds)
		if err != nil {
			return nil, err
		}
		stdinRecs = recs
	}

	kindFrom := map[string]string{}
	fileCache := map[string][]Record{}
	appendedFile := map[string]struct{}{}
	stdinAppended := false
	var out []Record

	for _, f := range active {
		flagName := "--from-" + f.Name
		var recs []Record
		var err error
		switch f.File {
		case "-":
			recs = stdinRecs
		default:
			if cached, ok := fileCache[f.File]; ok {
				recs = cached
			} else {
				recs, err = ReadFileTenant(f.File, tenant, checkKinds)
				if err != nil {
					return nil, err
				}
				fileCache[f.File] = recs
			}
		}

		for _, r := range recs {
			k := r.Kind()
			if !kindDeclared(k, f.Kinds) {
				continue
			}
			if prev, ok := kindFrom[k]; ok {
				if prev == flagName {
					continue
				}
				return nil, &ConflictError{Kind: k, Pipe: prev, Flag: flagName}
			}
			kindFrom[k] = flagName
		}

		switch f.File {
		case "-":
			if !stdinAppended {
				out = append(out, recs...)
				stdinAppended = true
			}
		default:
			if _, ok := appendedFile[f.File]; ok {
				continue
			}
			appendedFile[f.File] = struct{}{}
			out = append(out, recs...)
		}
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
