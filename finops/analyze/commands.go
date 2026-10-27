package analyze

import (
	"fmt"
	"os"

	"github.com/PopGreenTh/OpenXOps/record"
	"github.com/spf13/cobra"
)

var mapFrom = &record.From{Name: "map", Kinds: []string{"account", "allocation", "cost"}}

// Commands returns Analyze-stage commands for finops.
func Commands() []*cobra.Command {
	analyze := &cobra.Command{
		Use:   "analyze",
		Short: "Compare map outputs over time",
	}
	analyze.AddCommand(changesCommand())
	return []*cobra.Command{analyze}
}

func changesCommand() *cobra.Command {
	var before string
	cmd := &cobra.Command{
		Use:   "changes",
		Short: "Compare two map outputs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			record.LogStart("finops analyze changes", map[string]string{
				"before": before,
				"map":    mapFrom.File,
			}, nil)
			beforeRecs, err := record.ReadFile(before)
			if err != nil {
				return err
			}
			nowRecs, err := record.Assemble(os.Stdin, record.IsPipe(os.Stdin), []record.From{*mapFrom})
			if err != nil {
				return err
			}
			if len(nowRecs) == 0 {
				return fmt.Errorf("--from-map or pipe input is required")
			}
			changes := CompareChanges(beforeRecs, nowRecs)
			var out []record.Record
			out = append(out, changes...)
			out = append(out, nowRecs...)
			return record.Write(cmd.OutOrStdout(), out)
		},
	}
	cmd.Flags().StringVar(&before, "before", "", "earlier map or share output file")
	record.AddFromFlags(cmd, []*record.From{mapFrom})
	_ = cmd.MarkFlagRequired("before")
	return cmd
}
