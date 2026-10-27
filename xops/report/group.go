package report

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/PopGreenTh/OpenXOps/record"
)

// KindTable holds one kind's column order and row values (strings for output).
type KindTable struct {
	Kind    string
	Columns []string
	Rows    [][]string
}

// Group splits records into one table per kind (first-seen order). kindFilter limits kinds when non-empty.
func Group(recs []record.Record, kindFilter []string, tableNumbers bool) ([]KindTable, error) {
	allow := map[string]struct{}{}
	for _, k := range kindFilter {
		k = strings.TrimSpace(k)
		if k != "" {
			allow[k] = struct{}{}
		}
	}
	filter := len(allow) > 0

	var order []string
	byKind := map[string][]record.Record{}
	for _, r := range recs {
		k := r.Kind()
		if k == "" {
			return nil, fmt.Errorf("record missing kind")
		}
		if filter {
			if _, ok := allow[k]; !ok {
				continue
			}
		}
		if _, ok := byKind[k]; !ok {
			order = append(order, k)
		}
		byKind[k] = append(byKind[k], r)
	}

	out := make([]KindTable, 0, len(order))
	for _, k := range order {
		recsK := byKind[k]
		cols := columnsFirstSeen(recsK)
		rows := RowsForHeader(cols, recsK, tableNumbers)
		out = append(out, KindTable{Kind: k, Columns: cols, Rows: rows})
	}
	return out, nil
}

func columnsFirstSeen(recs []record.Record) []string {
	var cols []string
	for _, r := range recs {
		for _, field := range fieldNames(r) {
			if !hasColumn(cols, field) {
				cols = append(cols, field)
			}
		}
	}
	return cols
}

func fieldNames(r record.Record) []string {
	names := make([]string, 0, len(r))
	for k := range r {
		if k != "kind" {
			names = append(names, k)
		}
	}
	sort.Strings(names)
	return names
}

func hasColumn(cols []string, name string) bool {
	for _, c := range cols {
		if c == name {
			return true
		}
	}
	return false
}

// ExtraFields returns record field names (except kind) not present in header.
func ExtraFields(header []string, recs []record.Record) []string {
	h := map[string]struct{}{}
	for _, c := range header {
		h[c] = struct{}{}
	}
	seen := map[string]struct{}{}
	var extra []string
	for _, r := range recs {
		for field := range r {
			if field == "kind" {
				continue
			}
			if _, ok := h[field]; ok {
				continue
			}
			if _, dup := seen[field]; dup {
				continue
			}
			seen[field] = struct{}{}
			extra = append(extra, field)
		}
	}
	return extra
}

// RowsForHeader builds string rows aligned to an existing header order.
func RowsForHeader(header []string, recs []record.Record, tableNumbers bool) [][]string {
	rows := make([][]string, 0, len(recs))
	for _, r := range recs {
		row := make([]string, len(header))
		for i, col := range header {
			row[i] = formatCell(r[col], tableNumbers)
		}
		rows = append(rows, row)
	}
	return rows
}

// formatCell renders a scalar record value for output.
func formatCell(v any, twoDecimals bool) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		if twoDecimals {
			return strconv.FormatFloat(x, 'f', 2, 64)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case json.Number:
		f, err := x.Float64()
		if err != nil {
			return x.String()
		}
		if twoDecimals {
			return strconv.FormatFloat(f, 'f', 2, 64)
		}
		return strconv.FormatFloat(f, 'f', -1, 64)
	default:
		return fmt.Sprint(v)
	}
}

// ScalarFloat returns a numeric value when v is a JSON number.
func ScalarFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, !math.IsNaN(x)
	case json.Number:
		f, err := x.Float64()
		return f, err == nil && !math.IsNaN(f)
	default:
		return 0, false
	}
}
