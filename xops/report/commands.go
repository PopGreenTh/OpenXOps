package report

import (
	"fmt"
	"os"
	"strings"

	"github.com/PopGreenTh/OpenXOps/record"
	"github.com/spf13/cobra"
)

var reportFroms = []record.From{
	{Name: "cost", Kinds: []string{"cost"}},
	{Name: "change", Kinds: []string{"change"}},
	{Name: "allocation", Kinds: []string{"allocation"}},
	{Name: "finding", Kinds: []string{"finding"}},
	{Name: "accounts", Kinds: []string{"ou", "account"}},
}

// Commands returns Report-stage commands for xops.
func Commands() []*cobra.Command {
	report := &cobra.Command{
		Use:   "report",
		Short: "Turn records into tables, CSV or Excel",
	}
	report.AddCommand(tableCommand(), csvCommand(), excelCommand())
	return []*cobra.Command{report}
}

func tableCommand() *cobra.Command {
	var out, kinds string
	froms := append([]record.From(nil), reportFroms...)
	cmd := &cobra.Command{
		Use:   "table",
		Short: "Print aligned tables (optionally append to a text file)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReport(cmd, kinds, out, "table", froms)
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "append table text to FILE instead of stdout")
	cmd.Flags().StringVar(&kinds, "kinds", "", "limit to comma-separated kinds")
	record.AddFromFlags(cmd, fromPtrs(froms))
	return cmd
}

func csvCommand() *cobra.Command {
	var out, kinds string
	froms := append([]record.From(nil), reportFroms...)
	cmd := &cobra.Command{
		Use:   "csv",
		Short: "Write one CSV per kind under a directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(out) == "" {
				return fmt.Errorf("--out is required (directory for CSV files)")
			}
			return runReport(cmd, kinds, out, "csv", froms)
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "directory for `<kind>.csv` files (required)")
	cmd.Flags().StringVar(&kinds, "kinds", "", "limit to comma-separated kinds")
	record.AddFromFlags(cmd, fromPtrs(froms))
	return cmd
}

func excelCommand() *cobra.Command {
	var out, kinds string
	froms := append([]record.From(nil), reportFroms...)
	cmd := &cobra.Command{
		Use:   "excel",
		Short: "Write or append an Excel workbook (one sheet per kind)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(out) == "" {
				return fmt.Errorf("--out is required (path to .xlsx file)")
			}
			return runReport(cmd, kinds, out, "excel", froms)
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "workbook path (required)")
	cmd.Flags().StringVar(&kinds, "kinds", "", "limit to comma-separated kinds")
	record.AddFromFlags(cmd, fromPtrs(froms))
	return cmd
}

func runReport(cmd *cobra.Command, kinds, out, format string, froms []record.From) error {
	files := map[string]string{"out": out}
	for _, f := range froms {
		if f.File != "" {
			files["from-"+f.Name] = f.File
		}
	}
	record.LogStart("xops report "+format, files, map[string]string{"kinds": kinds})

	pipeOn := record.IsPipe(os.Stdin)
	recs, err := record.Assemble(os.Stdin, pipeOn, froms)
	if err != nil {
		return err
	}
	filter := splitKinds(kinds)
	tableNumbers := format == "table"
	tables, err := Group(recs, filter, tableNumbers)
	if err != nil {
		return err
	}
	switch format {
	case "table":
		return WriteTable(cmd.OutOrStdout(), out, tables, recs)
	case "csv":
		return WriteCSV(out, tables, recs)
	case "excel":
		return WriteExcel(out, tables, recs)
	default:
		return fmt.Errorf("unknown format %q", format)
	}
}

func fromPtrs(froms []record.From) []*record.From {
	ptrs := make([]*record.From, len(froms))
	for i := range froms {
		ptrs[i] = &froms[i]
	}
	return ptrs
}

func splitKinds(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
