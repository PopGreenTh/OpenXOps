package report_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/PopGreenTh/OpenXOps/record"
	"github.com/PopGreenTh/OpenXOps/xops/report"
	"github.com/xuri/excelize/v2"
)

func TestReportTableGolden(t *testing.T) {
	mixed := filepath.Join("testdata", "mixed.jsonl")
	recs := readMixed(t, mixed)
	tables, err := report.Group(recs, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := report.WriteTable(&buf, "", tables, recs); err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "table.txt", buf.Bytes())
}

func TestReportCSVGolden(t *testing.T) {
	dir := t.TempDir()
	mixed := filepath.Join("testdata", "mixed.jsonl")
	recs := readMixed(t, mixed)
	tables, err := report.Group(recs, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := report.WriteCSV(dir, tables, recs); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"alpha", "beta", "gamma"} {
		data, err := os.ReadFile(filepath.Join(dir, kind+".csv"))
		if err != nil {
			t.Fatal(err)
		}
		assertGolden(t, kind+".csv", data)
	}
}

func TestReportExcelGolden(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.xlsx")
	mixed := filepath.Join("testdata", "mixed.jsonl")
	recs := readMixed(t, mixed)
	tables, err := report.Group(recs, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := report.WriteExcel(path, tables, recs); err != nil {
		t.Fatal(err)
	}
	got := excelToText(t, path)
	assertGolden(t, "excel.txt", []byte(got))
}

func TestAppendTwiceCSV(t *testing.T) {
	dir := t.TempDir()
	mixed := filepath.Join("testdata", "mixed.jsonl")
	recs := readMixed(t, mixed)
	tables, err := report.Group(recs, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := report.WriteCSV(dir, tables, recs); err != nil {
		t.Fatal(err)
	}
	if err := report.WriteCSV(dir, tables, recs); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"alpha", "beta", "gamma"} {
		data, err := os.ReadFile(filepath.Join(dir, kind+".csv"))
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
		if len(lines) != 5 {
			t.Fatalf("%s.csv: want 5 lines (header + 4 rows), got %d:\n%s", kind, len(lines), data)
		}
		if lines[0] == lines[1] {
			t.Fatalf("%s.csv: duplicate header row", kind)
		}
	}
}

func TestAppendTwiceExcel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.xlsx")
	mixed := filepath.Join("testdata", "mixed.jsonl")
	recs := readMixed(t, mixed)
	tables, err := report.Group(recs, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := report.WriteExcel(path, tables, recs); err != nil {
		t.Fatal(err)
	}
	if err := report.WriteExcel(path, tables, recs); err != nil {
		t.Fatal(err)
	}
	got := excelToText(t, path)
	for _, kind := range []string{"alpha", "beta", "gamma"} {
		block := sheetBlock(got, kind)
		lines := strings.Split(strings.TrimSpace(block), "\n")
		if len(lines) != 6 {
			t.Fatalf("sheet %q: want title + header + 4 rows, got %d lines:\n%s", kind, len(lines), block)
		}
	}
}

func TestExcelAppendAfterAllEmptyRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.xlsx")
	first := []record.Record{
		{"kind": "alpha", "name": "first", "score": 1.0},
		{"kind": "alpha", "name": "", "score": nil},
	}
	tables, err := report.Group(first, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := report.WriteExcel(path, tables, first); err != nil {
		t.Fatal(err)
	}
	second := []record.Record{{"kind": "alpha", "name": "second", "score": 2.0}}
	tables, err = report.Group(second, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := report.WriteExcel(path, tables, second); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	row2Name, _ := f.GetCellValue("alpha", "A3")
	if row2Name != "" {
		t.Fatalf("row 3 name = %q, want empty (blank append row preserved)", row2Name)
	}
	row3Name, _ := f.GetCellValue("alpha", "A4")
	if row3Name != "second" {
		t.Fatalf("row 4 name = %q, want second", row3Name)
	}
}

func TestAppendTwiceTableFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.txt")
	mixed := filepath.Join("testdata", "mixed.jsonl")
	recs := readMixed(t, mixed)
	tables, err := report.Group(recs, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := report.WriteTable(nil, path, tables, recs); err != nil {
		t.Fatal(err)
	}
	if err := report.WriteTable(nil, path, tables, recs); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "table-append.txt", data)
}

func TestAppendNewFieldCSVSecondKind(t *testing.T) {
	dir := t.TempDir()
	mixed := filepath.Join("testdata", "mixed.jsonl")
	recs := readMixed(t, mixed)
	tables, err := report.Group(recs, []string{"alpha", "beta"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := report.WriteCSV(dir, tables, recs); err != nil {
		t.Fatal(err)
	}
	alphaBefore, err := os.ReadFile(filepath.Join(dir, "alpha.csv"))
	if err != nil {
		t.Fatal(err)
	}
	betaBefore, err := os.ReadFile(filepath.Join(dir, "beta.csv"))
	if err != nil {
		t.Fatal(err)
	}
	bad := append([]record.Record{}, recs[2])
	bad[0] = record.Record{"kind": "beta", "code": "b3", "qty": 1.0, "newfield": "x"}
	badTables, err := report.Group(bad, []string{"beta"}, false)
	if err != nil {
		t.Fatal(err)
	}
	err = report.WriteCSV(dir, badTables, bad)
	if err == nil {
		t.Fatal("expected error for new field on append")
	}
	alphaAfter, err := os.ReadFile(filepath.Join(dir, "alpha.csv"))
	if err != nil {
		t.Fatal(err)
	}
	betaAfter, err := os.ReadFile(filepath.Join(dir, "beta.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(alphaBefore, alphaAfter) {
		t.Fatalf("alpha.csv changed when beta append failed")
	}
	if !bytes.Equal(betaBefore, betaAfter) {
		t.Fatalf("beta.csv changed after failed append")
	}
}

func TestAppendNewFieldCSV(t *testing.T) {
	dir := t.TempDir()
	mixed := filepath.Join("testdata", "mixed.jsonl")
	recs := readMixed(t, mixed)
	tables, err := report.Group(recs, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := report.WriteCSV(dir, tables, recs); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(dir, "alpha.csv"))
	if err != nil {
		t.Fatal(err)
	}
	bad := append([]record.Record{}, recs[0])
	bad[0] = record.Record{"kind": "alpha", "name": "x", "score": 1.0, "extra": "nope"}
	badTables, err := report.Group(bad, []string{"alpha"}, false)
	if err != nil {
		t.Fatal(err)
	}
	err = report.WriteCSV(dir, badTables, bad)
	if err == nil {
		t.Fatal("expected error for new field on append")
	}
	after, err := os.ReadFile(filepath.Join(dir, "alpha.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("alpha.csv changed after failed append:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func readMixed(t *testing.T, path string) []record.Record {
	t.Helper()
	recs, err := record.ReadFile(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	return recs
}

func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	golden := filepath.Join("testdata", name)
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with UPDATE_GOLDEN=1)", golden, err)
	}
	if !bytes.Equal(want, got) {
		t.Fatalf("output differs from %s\n--- want ---\n%s--- got ---\n%s", golden, want, got)
	}
}

func excelToText(t *testing.T, path string) string {
	t.Helper()
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	names := f.GetSheetList()
	sort.Strings(names)
	var b strings.Builder
	for _, name := range names {
		fmt.Fprintf(&b, "--- %s ---\n", name)
		rows, err := f.GetRows(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range rows {
			b.WriteString(strings.Join(row, ","))
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func sheetBlock(all, kind string) string {
	marker := "--- " + kind + " ---"
	idx := strings.Index(all, marker)
	if idx < 0 {
		return ""
	}
	rest := all[idx+len(marker):]
	if next := strings.Index(rest, "--- "); next >= 0 {
		rest = rest[:next]
	}
	return marker + rest
}
