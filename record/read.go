package record

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/PopGreenTh/OpenXOps/record/schema"
)

const maxRecordLine = 10 << 20 // 10 MiB

// LineError names the file and line of a record that could not be read.
type LineError struct {
	File string
	Line int
	Err  error
}

func (e *LineError) Error() string {
	return fmt.Sprintf("%s:%d: %v", e.File, e.Line, e.Err)
}

func (e *LineError) Unwrap() error { return e.Err }

// TenantError names the file and line of a record whose tenant is not the command's --tenant.
type TenantError struct {
	File   string
	Line   int
	Got    string
	Tenant string
}

func (e *TenantError) Error() string {
	return fmt.Sprintf("%s:%d: record is for tenant %q, not --tenant %q", e.File, e.Line, e.Got, e.Tenant)
}

// Read parses UTF-8 JSON Lines from r. name is the file path or "stdin", used in errors.
// When checkKinds is non-nil, records whose kind is in the set are validated against schema.
func Read(r io.Reader, name string, checkKinds map[string]struct{}) ([]Record, error) {
	return ReadTenant(r, name, "", checkKinds)
}

// ReadTenant is Read, and when tenant is set it also stops at the first record of any kind that
// has a tenant field other than tenant, with a TenantError.
func ReadTenant(r io.Reader, name, tenant string, checkKinds map[string]struct{}) ([]Record, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), maxRecordLine)

	var recs []Record
	line := 0
	for sc.Scan() {
		line++
		raw := bytes.TrimRight(sc.Bytes(), "\r")
		if len(bytes.TrimSpace(raw)) == 0 {
			return nil, &LineError{File: name, Line: line, Err: fmt.Errorf("blank line")}
		}
		rec, err := parseRecord(raw)
		if err != nil {
			return nil, &LineError{File: name, Line: line, Err: err}
		}
		if v, ok := rec["tenant"]; ok && tenant != "" {
			if got, _ := v.(string); got != tenant {
				return nil, &TenantError{File: name, Line: line, Got: fmt.Sprint(v), Tenant: tenant}
			}
		}
		if err := schema.Validate(name, line, rec, checkKinds); err != nil {
			return nil, err
		}
		recs = append(recs, rec)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return recs, nil
}

// ReadFile reads JSON Lines from path. A path of "-" reads stdin.
func ReadFile(path string, checkKinds map[string]struct{}) ([]Record, error) {
	return ReadFileTenant(path, "", checkKinds)
}

// ReadFileTenant is ReadFile with the tenant check of ReadTenant.
func ReadFileTenant(path, tenant string, checkKinds map[string]struct{}) ([]Record, error) {
	if path == "-" {
		return ReadTenant(os.Stdin, "stdin", tenant, checkKinds)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	return ReadTenant(f, path, tenant, checkKinds)
}

func parseRecord(raw []byte) (Record, error) {
	var rec Record
	if err := json.Unmarshal(raw, &rec); err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("record is null")
	}
	if err := CheckFlat(rec); err != nil {
		return nil, err
	}
	v, ok := rec["kind"]
	if !ok {
		return nil, fmt.Errorf("missing kind")
	}
	kind, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("kind must be a string")
	}
	if strings.TrimSpace(kind) == "" {
		return nil, fmt.Errorf("kind is empty")
	}
	return rec, nil
}
