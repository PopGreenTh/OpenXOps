package enrich

import (
	"fmt"
	"os"

	"github.com/PopGreenTh/OpenXOps/finops/enrich/share"
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
	enrich.AddCommand(validateCommand(), mapCommand(), shareCommand())
	return []*cobra.Command{enrich}
}

func validateCommand() *cobra.Command {
	var taxonomy, tenant string
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate a taxonomy YAML file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			record.LogStart("finops enrich validate", map[string]string{"taxonomy": taxonomy}, map[string]string{"tenant": tenant})
			tax, err := loadTaxonomy(taxonomy, tenant)
			if err != nil {
				return err
			}
			findings := ValidateTaxonomy(tax)
			record.SetTenant(findings, tenant)
			if err := record.Write(cmd.OutOrStdout(), findings); err != nil {
				return err
			}
			if HasErrorSeverity(findings) {
				return fmt.Errorf("taxonomy validation failed")
			}
			return nil
		},
	}
	record.AddTenantFlag(cmd, &tenant)
	cmd.Flags().StringVar(&taxonomy, "taxonomy", "", "taxonomy YAML file (its tenant must be --tenant)")
	_ = cmd.MarkFlagRequired("taxonomy")
	return cmd
}

func mapCommand() *cobra.Command {
	var taxonomy, tenant string
	cmd := &cobra.Command{
		Use:   "map",
		Short: "Map accounts to apps using a taxonomy",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			record.LogStart("finops enrich map", map[string]string{
				"taxonomy": taxonomy,
				"accounts": accountsFrom.File,
				"cost":     costFrom.File,
			}, map[string]string{"tenant": tenant})
			tax, err := loadTaxonomy(taxonomy, tenant)
			if err != nil {
				return err
			}
			if HasErrorSeverity(ValidateTaxonomy(tax)) {
				return fmt.Errorf("taxonomy has validation errors; run finops enrich validate")
			}
			recs, err := record.AssembleTenant(os.Stdin, tenant, []record.From{*accountsFrom, *costFrom})
			if err != nil {
				return err
			}
			if !hasInventory(recs) {
				return fmt.Errorf("--from-accounts is required (ou and account records)")
			}
			out := Map(tax, recs)
			record.SetTenantOwn(recs, out, tenant)
			return record.Write(cmd.OutOrStdout(), out)
		},
	}
	record.AddTenantFlag(cmd, &tenant)
	cmd.Flags().StringVar(&taxonomy, "taxonomy", "", "taxonomy YAML file (its tenant must be --tenant)")
	record.AddFromFlags(cmd, []*record.From{accountsFrom, costFrom})
	_ = cmd.MarkFlagRequired("taxonomy")
	return cmd
}

var mapCostFrom = &record.From{Name: "cost", Kinds: []string{"allocation", "cost"}}

func shareCommand() *cobra.Command {
	var taxonomy, tenant string
	cmd := &cobra.Command{
		Use:   "share",
		Short: "Split shared-account cost by App tag",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			record.LogStart("finops enrich share", map[string]string{
				"taxonomy": taxonomy,
				"cost":     mapCostFrom.File,
			}, map[string]string{"tenant": tenant})
			tax, err := loadTaxonomy(taxonomy, tenant)
			if err != nil {
				return err
			}
			recs, err := record.AssembleTenant(os.Stdin, tenant, []record.From{*mapCostFrom})
			if err != nil {
				return err
			}
			out, err := share.Split(recs, shareOptions(tax))
			if err != nil {
				return err
			}
			record.SetTenantOwn(recs, out, tenant)
			return record.Write(cmd.OutOrStdout(), out)
		},
	}
	record.AddTenantFlag(cmd, &tenant)
	cmd.Flags().StringVar(&taxonomy, "taxonomy", "", "taxonomy YAML file (its tenant must be --tenant)")
	record.AddFromFlags(cmd, []*record.From{mapCostFrom})
	_ = cmd.MarkFlagRequired("taxonomy")
	return cmd
}

// loadTaxonomy reads the taxonomy at path and stops unless it is for tenant.
func loadTaxonomy(path, tenant string) (Taxonomy, error) {
	tax, err := LoadTaxonomy(path)
	if err != nil {
		return Taxonomy{}, err
	}
	if err := record.CheckTaxonomyTenant(path, tax.Tenant, tenant); err != nil {
		return Taxonomy{}, err
	}
	return tax, nil
}

func shareOptions(t Taxonomy) share.Options {
	key := "App"
	if t.Tags != nil && t.Tags.App != "" {
		key = t.Tags.App
	}
	codes := make([]string, 0, len(t.Apps))
	for _, a := range t.Apps {
		codes = append(codes, a.Code)
	}
	return share.Options{AppTagKey: key, AppCodes: codes}
}

func hasInventory(recs []record.Record) bool {
	for _, r := range recs {
		if r.Kind() == "ou" || r.Kind() == "account" {
			return true
		}
	}
	return false
}
