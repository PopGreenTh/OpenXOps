package report

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/PopGreenTh/OpenXOps/record"
)

// WriteTable renders tables to w, or appends to path when path is non-empty.
func WriteTable(w io.Writer, path string, tables []KindTable, recs []record.Record) error {
	if path == "" {
		for i, t := range tables {
			if i > 0 {
				fmt.Fprintln(w)
			}
			fmt.Fprint(w, formatTableBlock(t.Kind, t.Columns, t.Rows, false))
		}
		return nil
	}
	headers, err := tableHeadersFromFile(path)
	if err != nil {
		return err
	}
	prepared, err := prepareTablesForAppend(tables, recs, headers)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	for _, t := range prepared {
		appendOnly := len(headers[t.Kind]) > 0
		block := formatTableBlock(t.Kind, t.Columns, t.Rows, appendOnly)
		if _, err := io.WriteString(f, block); err != nil {
			return err
		}
		if !strings.HasSuffix(block, "\n") {
			if _, err := f.WriteString("\n"); err != nil {
				return err
			}
		}
	}
	return f.Sync()
}

func prepareTablesForAppend(tables []KindTable, recs []record.Record, existing map[string][]string) ([]KindTable, error) {
	out := make([]KindTable, len(tables))
	for i, t := range tables {
		recsK := recordsOfKind(recs, t.Kind)
		if header, ok := existing[t.Kind]; ok && len(header) > 0 {
			if extra := ExtraFields(header, recsK); len(extra) > 0 {
				return nil, fmt.Errorf("kind %q: record fields not in header: %s", t.Kind, strings.Join(extra, ", "))
			}
			out[i] = KindTable{
				Kind:    t.Kind,
				Columns: header,
				Rows:    RowsForHeader(header, recsK, true),
			}
			continue
		}
		out[i] = t
	}
	return out, nil
}

func recordsOfKind(recs []record.Record, kind string) []record.Record {
	var out []record.Record
	for _, r := range recs {
		if r.Kind() == kind {
			out = append(out, r)
		}
	}
	return out
}

func tableHeadersFromFile(path string) (map[string][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string][]string{}, nil
		}
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	out := map[string][]string{}
	var currentKind string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "--- ") && strings.HasSuffix(line, " ---") {
			currentKind = strings.TrimSuffix(strings.TrimPrefix(line, "--- "), " ---")
			continue
		}
		if currentKind != "" && strings.TrimSpace(line) != "" {
			out[currentKind] = strings.Fields(line)
			currentKind = ""
		}
	}
	return out, sc.Err()
}

func formatTableBlock(kind string, header []string, rows [][]string, appendOnlyRows bool) string {
	var b strings.Builder
	if !appendOnlyRows {
		fmt.Fprintf(&b, "--- %s ---\n", kind)
		if len(header) > 0 {
			widths := columnWidths(header, rows)
			b.WriteString(formatRow(header, widths))
			b.WriteByte('\n')
		}
	}
	if len(rows) > 0 {
		widths := columnWidths(header, rows)
		for _, row := range rows {
			b.WriteString(formatRow(row, widths))
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func columnWidths(header []string, rows [][]string) []int {
	n := len(header)
	if n == 0 && len(rows) > 0 {
		n = len(rows[0])
	}
	widths := make([]int, n)
	for i, h := range header {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i >= len(widths) {
				continue
			}
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	return widths
}

func formatRow(cells []string, widths []int) string {
	parts := make([]string, len(cells))
	for i, c := range cells {
		if i < len(widths) {
			parts[i] = fmt.Sprintf("%-*s", widths[i], c)
		} else {
			parts[i] = c
		}
	}
	return strings.Join(parts, "  ")
}
