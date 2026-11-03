package sense

import (
	"context"
	"fmt"
	"strings"

	"github.com/PopGreenTh/OpenXOps/aws"
	"github.com/PopGreenTh/OpenXOps/aws/ce"
	"github.com/PopGreenTh/OpenXOps/record"
	"github.com/PopGreenTh/OpenXOps/record/taxonomy"
	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/spf13/cobra"
)

var newCEClient = func(cfg awssdk.Config) ce.API {
	return ce.NewClient(cfg)
}

// Commands returns Sense-stage commands for finops.
func Commands() []*cobra.Command {
	sense := &cobra.Command{
		Use:   "sense",
		Short: "Read clouds and emit raw cost records",
	}
	sense.AddCommand(awsCostCommand())
	return []*cobra.Command{sense}
}

func awsCostCommand() *cobra.Command {
	var mock, refresh bool
	var profile, region, lz, taxPath, granularity, metric, from, to, appTag, projectTag, tenant string
	cmd := &cobra.Command{
		Use:   "aws-cost",
		Short: "Read AWS Cost Explorer and emit cost records",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			record.LogStart("finops sense aws-cost", map[string]string{"taxonomy": taxPath}, map[string]string{
				"mock":        fmt.Sprintf("%t", mock),
				"profile":     profile,
				"region":      region,
				"lz":          lz,
				"granularity": granularity,
				"metric":      metric,
				"from":        from,
				"to":          to,
				"app-tag":     appTag,
				"project-tag": projectTag,
				"refresh":     fmt.Sprintf("%t", refresh),
				"tenant":      tenant,
			})
			if lz != "" && taxPath == "" {
				return fmt.Errorf("--lz requires --taxonomy")
			}
			if taxPath != "" {
				tax, err := loadTaxonomy(taxPath, tenant)
				if err != nil {
					return err
				}
				if len(tax.LZs) != 1 {
					return fmt.Errorf("taxonomy %s has %d LZs, want exactly one", taxPath, len(tax.LZs))
				}
				code := tax.LZs[0].Code
				if lz != "" && !strings.EqualFold(lz, code) {
					return fmt.Errorf("--lz %q is not the LZ %q of taxonomy %s", lz, code, taxPath)
				}
				lz = code
			}
			var api ce.API
			if mock {
				api = ce.NewMock()
			} else {
				cfg, err := aws.LoadConfig(ctx, aws.Config{Profile: profile, Region: region})
				if err != nil {
					return err
				}
				api = newCEClient(cfg)
			}
			var calls int
			return RunAwsCost(ctx, CostOptions{
				Tenant:      tenant,
				Mock:        mock,
				Profile:     profile,
				Region:      region,
				LZ:          lz,
				Taxonomy:    taxPath,
				Granularity: granularity,
				Metric:      metric,
				From:        from,
				To:          to,
				AppTag:      appTag,
				ProjectTag:  projectTag,
				Refresh:     refresh,
				Out:         cmd.OutOrStdout(),
				CE:          api,
				CallCount:   &calls,
			})
		},
	}
	record.AddTenantFlag(cmd, &tenant)
	cmd.Flags().BoolVar(&mock, "mock", false, "use embedded deterministic Cost Explorer fixtures")
	cmd.Flags().BoolVar(&refresh, "refresh", false, "refetch all periods in range (never overwrite closed cache files)")
	cmd.Flags().StringVar(&profile, "profile", "", "AWS shared config profile")
	cmd.Flags().StringVar(&region, "region", "", "AWS region (default us-east-1)")
	cmd.Flags().StringVar(&lz, "lz", "", "landing zone code (requires --taxonomy; defaults to the taxonomy's one LZ)")
	cmd.Flags().StringVar(&taxPath, "taxonomy", "", "taxonomy YAML for --lz and default tag keys (its tenant must be --tenant)")
	cmd.Flags().StringVar(&granularity, "granularity", "monthly", "monthly or daily")
	cmd.Flags().StringVar(&metric, "metric", "", "Cost Explorer metric (default by granularity)")
	cmd.Flags().StringVar(&from, "from", "", "start period inclusive")
	cmd.Flags().StringVar(&to, "to", "", "end period inclusive")
	cmd.Flags().StringVar(&appTag, "app-tag", "", "App cost allocation tag key")
	cmd.Flags().StringVar(&projectTag, "project-tag", "", "Project cost allocation tag key")
	return cmd
}

// loadTaxonomy reads the taxonomy at path and stops unless it is for tenant.
func loadTaxonomy(path, tenant string) (taxonomy.Taxonomy, error) {
	tax, err := taxonomy.Load(path)
	if err != nil {
		return taxonomy.Taxonomy{}, err
	}
	if err := record.CheckTaxonomyTenant(path, tax.Tenant.Code, tenant); err != nil {
		return taxonomy.Taxonomy{}, err
	}
	return tax, nil
}

// resolveTags returns the App and Project tag keys: the flags, else the taxonomy's `tags`, else
// `App` and `Project`.
func resolveTags(opt CostOptions) (app, project string, err error) {
	app, project = opt.AppTag, opt.ProjectTag
	taxApp, taxProject := "App", "Project"
	if opt.Taxonomy != "" {
		tax, err := taxonomy.Load(opt.Taxonomy)
		if err != nil {
			return "", "", err
		}
		taxApp, taxProject = tax.TagKeys()
	}
	if app == "" {
		app = taxApp
	}
	if project == "" {
		project = taxProject
	}
	return app, project, nil
}

// managementAccountForLZ returns the management account of the LZ with this code (any case) in the
// taxonomy at path.
func managementAccountForLZ(path, code string) (string, error) {
	tax, err := taxonomy.Load(path)
	if err != nil {
		return "", err
	}
	for _, lz := range tax.LZs {
		if strings.EqualFold(lz.Code, code) {
			return lz.Management(), nil
		}
	}
	return "", fmt.Errorf("LZ %q not found in taxonomy %s", code, path)
}
