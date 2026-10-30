// Package record reads and writes flat JSON Lines records and assembles command input.
package record

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/PopGreenTh/OpenXOps/record/urn"
)

// Record is one flat JSON object. Values are strings, numbers, booleans or null.
type Record map[string]any

// Kind returns the record's kind field, or "" if missing or not a string.
func (r Record) Kind() string {
	v, ok := r["kind"]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// Finding builds a finding record that reports a problem without failing the command.
func Finding(severity, code, subject, message, command string) Record {
	return Record{
		"kind":     "finding",
		"severity": severity,
		"code":     code,
		"subject":  subject,
		"message":  message,
		"command":  command,
	}
}

// CheckTenantID returns an error unless id is set and follows the id rule (D-045, D-047).
func CheckTenantID(id string) error {
	if id == "" {
		return fmt.Errorf("--tenant is required")
	}
	if !urn.ValidID(id) {
		return fmt.Errorf("invalid --tenant %q (want lowercase letters and digits, joined by single dashes)", id)
	}
	return nil
}

// CheckTaxonomyTenant returns an error unless the taxonomy at path declares tenant fileTenant
// and it equals tenant (the --tenant flag). The error names both.
func CheckTaxonomyTenant(path, fileTenant, tenant string) error {
	if fileTenant == "" {
		return fmt.Errorf("taxonomy %s: missing tenant (the file must start with `tenant: <id>`)", path)
	}
	if !urn.ValidID(fileTenant) {
		return fmt.Errorf("taxonomy %s: invalid tenant %q (want lowercase letters and digits, joined by single dashes)", path, fileTenant)
	}
	if fileTenant != tenant {
		return fmt.Errorf("taxonomy %s is for tenant %q, not --tenant %q", path, fileTenant, tenant)
	}
	return nil
}

// SetTenant sets tenant on every record in recs: for commands whose output is all their own.
func SetTenant(recs []Record, tenant string) {
	for _, r := range recs {
		r["tenant"] = tenant
	}
}

// SetTenantOwn sets tenant on every record in out that is not one of the records in in, so
// records passed through from the input stay unchanged. Records are compared by identity.
func SetTenantOwn(in, out []Record, tenant string) {
	seen := make(map[uintptr]struct{}, len(in))
	for _, r := range in {
		seen[reflect.ValueOf(r).Pointer()] = struct{}{}
	}
	for _, r := range out {
		if _, ok := seen[reflect.ValueOf(r).Pointer()]; ok {
			continue
		}
		r["tenant"] = tenant
	}
}

// PassThrough returns out (the command's own records) followed by every input
// record whose kind is not in handled, in input order.
func PassThrough(in []Record, handled []string, out []Record) []Record {
	skip := make(map[string]struct{}, len(handled))
	for _, k := range handled {
		skip[k] = struct{}{}
	}
	result := append([]Record{}, out...)
	for _, r := range in {
		if _, ok := skip[r.Kind()]; ok {
			continue
		}
		result = append(result, r)
	}
	return result
}

// CheckFlat reports whether every value is a scalar (string, number, bool or null).
func CheckFlat(r Record) error {
	for k, v := range r {
		if !isScalar(v) {
			return fmt.Errorf("field %q is nested (objects and arrays are not allowed)", k)
		}
	}
	return nil
}

func isScalar(v any) bool {
	switch v.(type) {
	case nil, bool, string, float64, json.Number, int, int32, int64, uint, uint32, uint64:
		return true
	default:
		return false
	}
}
