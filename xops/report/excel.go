package report

import (
	"fmt"
	"os"
	"strings"

	"github.com/PopGreenTh/OpenXOps/record"
	"github.com/xuri/excelize/v2"
)

// WriteExcel writes or appends rows in one workbook (one sheet per kind).
func WriteExcel(path string, tables []KindTable, recs []record.Record) error {
	f, err := openWorkbook(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	for _, t := range tables {
		recsK := recordsOfKind(recs, t.Kind)
		if err := appendSheet(f, t.Kind, recsK); err != nil {
			return err
		}
	}
	if err := f.SaveAs(path); err != nil {
		return fmt.Errorf("save %s: %w", path, err)
	}
	return nil
}

func openWorkbook(path string) (*excelize.File, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return excelize.NewFile(), nil
		}
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	return f, nil
}

func appendSheet(f *excelize.File, kind string, recs []record.Record) error {
	sheet := kind
	idx, err := f.GetSheetIndex(sheet)
	exists := err == nil && idx != -1
	if !exists {
		if _, err := f.NewSheet(sheet); err != nil {
			return fmt.Errorf("sheet %q: %w", kind, err)
		}
		if def, _ := f.GetSheetIndex("Sheet1"); def != -1 && f.GetSheetName(def) == "Sheet1" {
			_ = f.DeleteSheet("Sheet1")
		}
	}
	var cols []string
	if exists {
		header, err := readExcelHeader(f, sheet)
		if err != nil {
			return err
		}
		if extra := ExtraFields(header, recs); len(extra) > 0 {
			return fmt.Errorf("kind %q: record fields not in header: %s", kind, strings.Join(extra, ", "))
		}
		cols = header
	} else {
		cols = columnsFirstSeen(recs)
	}
	startRow := 1
	if exists {
		last, err := lastNonEmptyRow(f, sheet)
		if err != nil {
			return err
		}
		startRow = last + 1
	} else {
		for i, col := range cols {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			if err := f.SetCellValue(sheet, cell, col); err != nil {
				return err
			}
		}
		startRow = 2
	}
	for ri := range recs {
		for ci, colName := range cols {
			cell, _ := excelize.CoordinatesToCellName(ci+1, startRow+ri)
			val := recs[ri][colName]
			if err := setCellScalar(f, sheet, cell, val); err != nil {
				return err
			}
		}
	}
	return nil
}

func lastNonEmptyRow(f *excelize.File, sheet string) (int, error) {
	rows, err := f.GetRows(sheet)
	if err != nil {
		return 0, err
	}
	last := 0
	for i, row := range rows {
		if rowHasContent(row) {
			last = i + 1
		}
	}
	return last, nil
}

func rowHasContent(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return true
		}
	}
	return false
}

func readExcelHeader(f *excelize.File, sheet string) ([]string, error) {
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("sheet %q has no header row", sheet)
	}
	return rows[0], nil
}

func setCellScalar(f *excelize.File, sheet, cell string, v any) error {
	if v == nil {
		return f.SetCellValue(sheet, cell, "")
	}
	if f64, ok := ScalarFloat(v); ok {
		return f.SetCellValue(sheet, cell, f64)
	}
	switch x := v.(type) {
	case bool:
		return f.SetCellValue(sheet, cell, x)
	case string:
		return f.SetCellValue(sheet, cell, x)
	default:
		return f.SetCellValue(sheet, cell, fmt.Sprint(v))
	}
}
