// Package record reads and writes flat JSON Lines records and assembles command input.
package record

import (
	"encoding/json"
	"fmt"
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
