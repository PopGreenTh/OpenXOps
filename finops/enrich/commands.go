package enrich

import (
	"fmt"
	"os"

	"github.com/PopGreenTh/OpenXOps/record"
	"github.com/spf13/cobra"
)

var accountsFrom = &record.From{Name: "accounts", Kinds: []string{"ou", "account"}}
var costFrom = &record.From{Name: "cost", Kinds: []string{"cost", "project_tag_cost"}}

// Commands returns Enrich-stage commands for finops.
func Commands() []*cobra.Command {
	enrich := &cobra.Command{
		Use:   "enrich",
		Short: "Validate taxonomy and map accounts to apps",
	}
	enrich.AddCommand(validateCommand(), mapCommand())
	return []*cobra.Command{enrich}
}

func validateCommand() *cobra.Command {
	var taxonomy string
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate a taxonomy YAML file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			record.LogStart("finops enrich validate", map[string]string{"taxonomy": taxonomy}, nil)
			tax, err := LoadTaxonomy(taxonomy)
			if err != nil {
				return err
			}
			findings := ValidateTaxonomy(tax)
			if err := record.Write(cmd.OutOrStdout(), findings); err != nil {
				return err
			}
			if HasErrorSeverity(findings) {
				return fmt.Errorf("taxonomy validation failed")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&taxonomy, "taxonomy", "", "taxonomy YAML file")
	_ = cmd.MarkFlagRequired("taxonomy")
	return cmd
}

func mapCommand() *cobra.Command {
	var taxonomy string
	cmd := &cobra.Command{
		Use:   "map",
		Short: "Map accounts to apps using a taxonomy",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			record.LogStart("finops enrich map", map[string]string{
				"taxonomy": taxonomy,
				"accounts": accountsFrom.File,
				"cost":     costFrom.File,
			}, nil)
			tax, err := LoadTaxonomy(taxonomy)
			if err != nil {
				return err
			}
			if HasErrorSeverity(ValidateTaxonomy(tax)) {
				return fmt.Errorf("taxonomy has validation errors; run finops enrich validate")
			}
			recs, err := record.Assemble(os.Stdin, record.IsPipe(os.Stdin), []record.From{*accountsFrom, *costFrom})
			if err != nil {
				return err
			}
			if !hasInventory(recs) {
				return fmt.Errorf("--from-accounts is required (ou and account records)")
			}
			out := Map(tax, recs)
			return record.Write(cmd.OutOrStdout(), out)
		},
	}
	cmd.Flags().StringVar(&taxonomy, "taxonomy", "", "taxonomy YAML file")
	record.AddFromFlags(cmd, []*record.From{accountsFrom, costFrom})
	_ = cmd.MarkFlagRequired("taxonomy")
	return cmd
}

func hasInventory(recs []record.Record) bool {
	for _, r := range recs {
		if r.Kind() == "ou" || r.Kind() == "account" {
			return true
		}
	}
	return false
}
