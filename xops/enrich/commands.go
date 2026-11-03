package enrich

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

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
	cmd.Flags().StringVar(&taxonomy, "taxonomy", "", "taxonomy YAML file: its one LZ is the top of the tree (its tenant must be --tenant)")
	_ = cmd.MarkFlagRequired("taxonomy")
	record.AddFromFlags(cmd, []*record.From{&from})
	return cmd
}

type rawOUPath struct {
	Code     string       `yaml:"code"`
	Name     string       `yaml:"name"`
	Path     string       `yaml:"path"`
	Alias    string       `yaml:"alias"`
	Accounts []rawAccount `yaml:"accounts"`
}

type rawAccount struct {
	Code    string `yaml:"code"`
	Name    string `yaml:"name"`
	RefID   string `yaml:"ref_id"`
	Account string `yaml:"account"`
	Alias   string `yaml:"alias"`
}

type rawOULink struct {
	OU    string `yaml:"ou"`
	Alias string `yaml:"alias"`
}

type rawAccountLink struct {
	Account string `yaml:"account"`
	Alias   string `yaml:"alias"`
}

type rawLZItem struct {
	Code              string      `yaml:"code"`
	URN               string      `yaml:"urn"`
	Name              string      `yaml:"name"`
	Alias             string      `yaml:"alias"`
	CSP               string      `yaml:"csp"`
	ManagementAccount string      `yaml:"management_account"`
	RefID             string      `yaml:"ref_id"`
	OUPaths           []rawOUPath `yaml:"ou_paths"`
}

// loadLZ reads the tenant and the one LZ of the taxonomy at path (D-052) and checks the tenant.
func loadLZ(path, tenant string) (LZ, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return LZ{}, fmt.Errorf("read taxonomy %s: %w", path, err)
	}
	var doc struct {
		Tenant   yaml.Node        `yaml:"tenant"`
		LZs      []rawLZItem      `yaml:"lzs"`
		LZConns  []rawLZItem      `yaml:"lz_conns"`
		OULinks  []rawOULink      `yaml:"ou_links"`
		Accounts []rawAccountLink `yaml:"accounts"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return LZ{}, fmt.Errorf("parse taxonomy %s: %w", path, err)
	}
	tnt, err := parseTenantNode(&doc.Tenant)
	if err != nil {
		return LZ{}, fmt.Errorf("parse taxonomy %s tenant: %w", path, err)
	}
	if err := record.CheckTaxonomyTenant(path, tnt, tenant); err != nil {
		return LZ{}, err
	}
	lzs := doc.LZs
	if len(lzs) == 0 {
		lzs = doc.LZConns
	}
	if len(lzs) != 1 {
		return LZ{}, fmt.Errorf("taxonomy %s has %d LZs, want exactly one", path, len(lzs))
	}
	l := lzs[0]
	mgmt := l.ManagementAccount
	if mgmt == "" {
		mgmt = l.RefID
	}
	code := l.Code
	if l.URN != "" {
		if u := urnSegment(l.URN, "lzc"); u != "" {
			code = u
		}
	}

	ouAliases := map[string]string{}
	accountAliases := map[string]string{}
	for _, ou := range l.OUPaths {
		if ou.Alias != "" {
			if ou.Path != "" {
				ouAliases[ou.Path] = ou.Alias
			}
			if ou.Code != "" {
				ouAliases[ou.Code] = ou.Alias
			}
			if ou.Name != "" {
				ouAliases[ou.Name] = ou.Alias
			}
		}
		for _, acc := range ou.Accounts {
			if acc.Alias != "" {
				if acc.RefID != "" {
					accountAliases[acc.RefID] = acc.Alias
				}
				if acc.Code != "" {
					accountAliases[acc.Code] = acc.Alias
				}
				if acc.Account != "" {
					accountAliases[acc.Account] = acc.Alias
				}
				if acc.Name != "" {
					accountAliases[acc.Name] = acc.Alias
				}
			}
		}
	}
	for _, link := range doc.OULinks {
		if link.Alias != "" && link.OU != "" {
			ouAliases[link.OU] = link.Alias
		}
	}
	for _, acc := range doc.Accounts {
		if acc.Alias != "" && acc.Account != "" {
			accountAliases[acc.Account] = acc.Alias
		}
	}

	return LZ{
		Code:              code,
		Name:              l.Name,
		CSP:               strings.ToLower(l.CSP),
		ManagementAccount: mgmt,
		Alias:             l.Alias,
		OUAliases:         ouAliases,
		AccountAliases:    accountAliases,
	}, nil
}

func parseTenantNode(n *yaml.Node) (string, error) {
	if n == nil || n.Kind == 0 {
		return "", nil
	}
	if n.Kind == yaml.ScalarNode {
		return n.Value, nil
	}
	if n.Kind == yaml.MappingNode {
		var m struct {
			Code string `yaml:"code"`
			URN  string `yaml:"urn"`
		}
		if err := n.Decode(&m); err != nil {
			return "", err
		}
		if m.URN != "" {
			if t := urnSegment(m.URN, "tnt"); t != "" {
				return t, nil
			}
		}
		if m.Code != "" {
			return strings.ToLower(m.Code), nil
		}
	}
	return "", fmt.Errorf("unexpected tenant node kind")
}

func urnSegment(u, typ string) string {
	parts := strings.Split(u, ":")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == typ {
			return parts[i+1]
		}
	}
	return ""
}

func hasInventory(recs []record.Record) bool {
	for _, r := range recs {
		if r.Kind() == "ou" || r.Kind() == "account" {
			return true
		}
	}
	return false
}
