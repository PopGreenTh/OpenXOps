package sense

import (
	"context"
	"fmt"
	"os"

	"github.com/PopGreenTh/OpenXOps/aws"
	"github.com/PopGreenTh/OpenXOps/aws/ce"
	"github.com/PopGreenTh/OpenXOps/record"
	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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
	var profile, region, lz, taxonomy, granularity, metric, from, to, appTag, projectTag, tenant string
	cmd := &cobra.Command{
		Use:   "aws-cost",
		Short: "Read AWS Cost Explorer and emit cost records",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			record.LogStart("finops sense aws-cost", map[string]string{"taxonomy": taxonomy}, map[string]string{
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
			if lz != "" && taxonomy == "" {
				return fmt.Errorf("--lz requires --taxonomy")
			}
			if taxonomy != "" && lz == "" {
				return fmt.Errorf("--taxonomy requires --lz")
			}
			if taxonomy != "" {
				if err := checkTaxonomyTenant(taxonomy, tenant); err != nil {
					return err
				}
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
				Taxonomy:    taxonomy,
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
	cmd.Flags().StringVar(&lz, "lz", "", "landing zone code (requires --taxonomy)")
	cmd.Flags().StringVar(&taxonomy, "taxonomy", "", "taxonomy YAML for --lz and default tag keys (its tenant must be --tenant)")
	cmd.Flags().StringVar(&granularity, "granularity", "monthly", "monthly or daily")
	cmd.Flags().StringVar(&metric, "metric", "", "Cost Explorer metric (default by granularity)")
	cmd.Flags().StringVar(&from, "from", "", "start period inclusive")
	cmd.Flags().StringVar(&to, "to", "", "end period inclusive")
	cmd.Flags().StringVar(&appTag, "app-tag", "", "App cost allocation tag key")
	cmd.Flags().StringVar(&projectTag, "project-tag", "", "Project cost allocation tag key")
	return cmd
}

// checkTaxonomyTenant stops when the taxonomy at path is not for tenant.
func checkTaxonomyTenant(path, tenant string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read taxonomy %s: %w", path, err)
	}
	var doc struct {
		Tenant string `yaml:"tenant"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parse taxonomy %s: %w", path, err)
	}
	return record.CheckTaxonomyTenant(path, doc.Tenant, tenant)
}

func resolveTags(opt CostOptions) (app, project string, err error) {
	app = opt.AppTag
	project = opt.ProjectTag
	if opt.Taxonomy != "" {
		data, err := os.ReadFile(opt.Taxonomy)
		if err != nil {
			return "", "", fmt.Errorf("read taxonomy %s: %w", opt.Taxonomy, err)
		}
		var doc struct {
			Tags struct {
				App     string `yaml:"app"`
				Project string `yaml:"project"`
			} `yaml:"tags"`
		}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return "", "", fmt.Errorf("parse taxonomy %s: %w", opt.Taxonomy, err)
		}
		if app == "" && doc.Tags.App != "" {
			app = doc.Tags.App
		}
		if project == "" && doc.Tags.Project != "" {
			project = doc.Tags.Project
		}
	}
	if app == "" {
		app = "App"
	}
	if project == "" {
		project = "Project"
	}
	return app, project, nil
}

func managementAccountForLZ(path, code string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read taxonomy %s: %w", path, err)
	}
	var doc struct {
		LZs []struct {
			Code              string `yaml:"code"`
			ManagementAccount string `yaml:"management_account"`
		} `yaml:"lzs"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return "", fmt.Errorf("parse taxonomy %s: %w", path, err)
	}
	for _, lz := range doc.LZs {
		if lz.Code == code {
			return lz.ManagementAccount, nil
		}
	}
	return "", fmt.Errorf("LZ %q not found in taxonomy %s", code, path)
}
