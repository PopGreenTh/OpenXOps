package record

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

// Write writes records as UTF-8 JSON Lines with `\n` endings and no blank lines.
// The kind field is written first; remaining keys are sorted.
func Write(w io.Writer, recs []Record) error {
	for i, rec := range recs {
		if err := CheckFlat(rec); err != nil {
			return fmt.Errorf("record %d: %w", i+1, err)
		}
		if rec.Kind() == "" {
			return fmt.Errorf("record %d: missing kind", i+1)
		}
		raw, err := marshalFlat(rec)
		if err != nil {
			return fmt.Errorf("record %d: %w", i+1, err)
		}
		if _, err := w.Write(raw); err != nil {
			return err
		}
		if _, err := w.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return nil
}

func marshalFlat(rec Record) ([]byte, error) {
	keys := make([]string, 0, len(rec))
	for k := range rec {
		if k != "kind" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var buf []byte
	buf = append(buf, '{')
	first := true
	appendField := func(k string) error {
		v, ok := rec[k]
		if !ok {
			return nil
		}
		if !first {
			buf = append(buf, ',')
		}
		first = false
		key, err := json.Marshal(k)
		if err != nil {
			return err
		}
		val, err := json.Marshal(v)
		if err != nil {
			return err
		}
		buf = append(buf, key...)
		buf = append(buf, ':')
		buf = append(buf, val...)
		return nil
	}
	if err := appendField("kind"); err != nil {
		return nil, err
	}
	for _, k := range keys {
		if err := appendField(k); err != nil {
			return nil, err
		}
	}
	buf = append(buf, '}')
	return buf, nil
}
