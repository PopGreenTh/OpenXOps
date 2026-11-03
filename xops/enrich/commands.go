package enrich

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/PopGreenTh/OpenXOps/record"
	"github.com/PopGreenTh/OpenXOps/record/taxonomy"
	"github.com/spf13/cobra"
)

// Commands returns Enrich-stage commands for xops.
func Commands() []*cobra.Command {
	enrich := &cobra.Command{
		Use:   "enrich",
		Short: "Turn raw records into higher-level records",
	}
	enrich.AddCommand(lzTreeCommand())
	return []*cobra.Command{enrich}
}

func lzTreeCommand() *cobra.Command {
	var taxPath, tenant string
	from := record.From{Name: "accounts", Kinds: []string{"ou", "account"}}
	cmd := &cobra.Command{
		Use:   "lz-tree",
		Short: "Turn ou and account records into one node record per LZ, OU and account",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			record.LogStart("xops enrich lz-tree", map[string]string{"taxonomy": taxPath, "accounts": from.File},
				map[string]string{"tenant": tenant})
			lz, err := loadLZ(taxPath, tenant)
			if err != nil {
				return err
			}
			recs, err := record.AssembleTenant(os.Stdin, tenant, []record.From{from})
			if err != nil {
				return err
			}
			if !hasInventory(recs) {
				return fmt.Errorf("--from-accounts or piped input with ou and account records is required")
			}
			source := from.File
			if source == "" || source == "-" {
				source = "stdin"
			}
			if err := CheckManagement(source, recs, lz); err != nil {
				return err
			}
			nodes := LZTree(tenant, lz, recs)
			warnings, err := CheckNodes(nodes)
			for _, w := range warnings {
				slog.Warn(w)
			}
			if err != nil {
				return err
			}
			return record.Write(cmd.OutOrStdout(), nodes)
		},
	}
	record.AddTenantFlag(cmd, &tenant)
	cmd.Flags().StringVar(&taxPath, "taxonomy", "", "taxonomy YAML file: its one LZ is the top of the tree (its tenant must be --tenant)")
	_ = cmd.MarkFlagRequired("taxonomy")
	record.AddFromFlags(cmd, []*record.From{&from})
	return cmd
}

// loadLZ reads the one LZ of the taxonomy at path (D-052) and checks the tenant. An OU entry's
// alias is found by its ref_id when it has one, else by its path, the way `finops enrich map`
// matches the entry to an OU (D-061); an account entry's alias by its ref_id.
func loadLZ(path, tenant string) (LZ, error) {
	tax, err := taxonomy.Load(path)
	if err != nil {
		return LZ{}, err
	}
	if err := record.CheckTaxonomyTenant(path, tax.Tenant.Code, tenant); err != nil {
		return LZ{}, err
	}
	if len(tax.LZs) != 1 {
		return LZ{}, fmt.Errorf("taxonomy %s has %d LZs, want exactly one", path, len(tax.LZs))
	}
	l := tax.LZs[0]
	ouAliases := map[string]string{}
	accountAliases := map[string]string{}
	for _, ou := range l.OUPaths {
		if ou.Alias != "" {
			if ou.RefID != "" {
				ouAliases[ou.RefID] = ou.Alias
			} else if ou.Path != "" {
				ouAliases[ou.Path] = ou.Alias
			}
		}
		for _, acc := range ou.Accounts {
			if acc.Alias != "" && acc.RefID != "" {
				accountAliases[acc.RefID] = acc.Alias
			}
		}
	}
	return LZ{
		Code:              l.Code,
		Name:              l.Name,
		CSP:               strings.ToLower(l.CSP),
		ManagementAccount: l.Management(),
		Alias:             l.Alias,
		OUAliases:         ouAliases,
		AccountAliases:    accountAliases,
	}, nil
}

func hasInventory(recs []record.Record) bool {
	for _, r := range recs {
		if r.Kind() == "ou" || r.Kind() == "account" {
			return true
		}
	}
	return false
}
