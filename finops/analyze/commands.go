package analyze

import (
	"fmt"
	"os"

	"github.com/PopGreenTh/OpenXOps/record"
	"github.com/spf13/cobra"
)

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
	var before, tenant string
	mapFrom := &record.From{Name: "map", Kinds: []string{"account", "allocation", "cost"}}
	cmd := &cobra.Command{
		Use:   "changes",
		Short: "Compare two map outputs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			record.LogStart("finops analyze changes", map[string]string{
				"before": before,
				"map":    mapFrom.File,
			}, map[string]string{"tenant": tenant})
			check := record.CheckKindsFrom([]record.From{*mapFrom})
			beforeRecs, err := record.ReadFileTenant(before, tenant, check)
			if err != nil {
				return err
			}
			nowRecs, err := record.AssembleTenant(os.Stdin, tenant, []record.From{*mapFrom})
			if err != nil {
				return err
			}
			if len(nowRecs) == 0 {
				return fmt.Errorf("--from-map or pipe input is required")
			}
			changes := CompareChanges(beforeRecs, nowRecs)
			record.SetTenant(changes, tenant)
			var out []record.Record
			out = append(out, changes...)
			out = append(out, nowRecs...)
			return record.Write(cmd.OutOrStdout(), out)
		},
	}
	record.AddTenantFlag(cmd, &tenant)
	cmd.Flags().StringVar(&before, "before", "", "earlier map or share output file")
	record.AddFromFlags(cmd, []*record.From{mapFrom})
	_ = cmd.MarkFlagRequired("before")
	return cmd
}
