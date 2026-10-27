package report

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PopGreenTh/OpenXOps/record"
)

// WriteCSV writes one CSV file per kind under dir.
func WriteCSV(dir string, tables []KindTable, recs []record.Record) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	for _, t := range tables {
		path := filepath.Join(dir, t.Kind+".csv")
		recsK := recordsOfKind(recs, t.Kind)
		if err := appendCSV(path, t.Kind, recsK); err != nil {
			return err
		}
	}
	return nil
}

func appendCSV(path, kind string, recs []record.Record) error {
	header, exists, err := readCSVHeader(path)
	if err != nil {
		return err
	}
	var cols []string
	var rows [][]string
	if exists {
		if extra := ExtraFields(header, recs); len(extra) > 0 {
			return fmt.Errorf("kind %q: record fields not in header: %s", kind, strings.Join(extra, ", "))
		}
		cols = header
		rows = RowsForHeader(header, recs, false)
	} else {
		cols = columnsFirstSeen(recs)
		rows = RowsForHeader(cols, recs, false)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if !exists {
		if err := w.Write(cols); err != nil {
			return err
		}
	}
	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}
	return f.Sync()
}

func readCSVHeader(path string) ([]string, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return nil, false, fmt.Errorf("read %s: %w", path, err)
	}
	return header, true, nil
}
