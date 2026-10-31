package enrich

import (
	"fmt"
	"os"

	"github.com/PopGreenTh/OpenXOps/record"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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
	var taxonomy, tenant string
	from := record.From{Name: "accounts", Kinds: []string{"ou", "account"}}
	cmd := &cobra.Command{
		Use:   "lz-tree",
		Short: "Turn ou and account records into one node record per LZ, OU and account",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			record.LogStart("xops enrich lz-tree", map[string]string{"taxonomy": taxonomy, "accounts": from.File},
				map[string]string{"tenant": tenant})
			lz, err := loadLZ(taxonomy, tenant)
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
			return record.Write(cmd.OutOrStdout(), LZTree(tenant, lz, recs))
		},
	}
	record.AddTenantFlag(cmd, &tenant)
	cmd.Flags().StringVar(&taxonomy, "taxonomy", "", "taxonomy YAML file: its one LZ is the top of the tree (its tenant must be --tenant)")
	_ = cmd.MarkFlagRequired("taxonomy")
	record.AddFromFlags(cmd, []*record.From{&from})
	return cmd
}

// loadLZ reads the tenant and the one LZ of the taxonomy at path (D-052) and checks the tenant.
func loadLZ(path, tenant string) (LZ, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return LZ{}, fmt.Errorf("read taxonomy %s: %w", path, err)
	}
	var doc struct {
		Tenant string `yaml:"tenant"`
		LZs    []struct {
			Code              string `yaml:"code"`
			Name              string `yaml:"name"`
			CSP               string `yaml:"csp"`
			ManagementAccount string `yaml:"management_account"`
		} `yaml:"lzs"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return LZ{}, fmt.Errorf("parse taxonomy %s: %w", path, err)
	}
	if err := record.CheckTaxonomyTenant(path, doc.Tenant, tenant); err != nil {
		return LZ{}, err
	}
	if len(doc.LZs) != 1 {
		return LZ{}, fmt.Errorf("taxonomy %s has %d LZs, want exactly one", path, len(doc.LZs))
	}
	l := doc.LZs[0]
	return LZ{Code: l.Code, Name: l.Name, CSP: l.CSP, ManagementAccount: l.ManagementAccount}, nil
}

func hasInventory(recs []record.Record) bool {
	for _, r := range recs {
		if r.Kind() == "ou" || r.Kind() == "account" {
			return true
		}
	}
	return false
}
