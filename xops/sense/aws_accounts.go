package sense

import (
	"context"
	"fmt"
	"os"

	"github.com/PopGreenTh/OpenXOps/aws"
	"github.com/PopGreenTh/OpenXOps/record"
	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var newOrgAPI = func(cfg awssdk.Config) aws.OrgAPI {
	return aws.NewOrgClient(cfg)
}

func awsAccountsCommand() *cobra.Command {
	var mock bool
	var profile, region, lz, taxonomy, tenant string
	cmd := &cobra.Command{
		Use:   "aws-accounts",
		Short: "List AWS Organization OUs and accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			record.LogStart("xops sense aws-accounts", map[string]string{"taxonomy": taxonomy}, map[string]string{
				"mock":    fmt.Sprintf("%t", mock),
				"profile": profile,
				"region":  region,
				"lz":      lz,
				"tenant":  tenant,
			})
			lzCode, err := resolveLZ(lz, taxonomy)
			if err != nil {
				return err
			}
			var snap aws.Snapshot
			if mock {
				snap, err = aws.DemoSnapshot()
			} else {
				cfg, loadErr := aws.LoadConfig(ctx, aws.Config{Profile: profile, Region: region})
				if loadErr != nil {
					return loadErr
				}
				snap, err = aws.DiscoverSnapshot(ctx, newOrgAPI(cfg))
			}
			if err != nil {
				return err
			}
			if lzCode != "" {
				want, err := managementAccountForLZ(taxonomy, lzCode)
				if err != nil {
					return err
				}
				if snap.ManagementAccount != want {
					return fmt.Errorf("organization management account %q does not match LZ %q in taxonomy (want %q)", snap.ManagementAccount, lzCode, want)
				}
			}
			recs := BuildRecords(snap, lzCode)
			record.SetTenant(recs, tenant)
			return record.Write(cmd.OutOrStdout(), recs)
		},
	}
	record.AddTenantFlag(cmd, &tenant)
	cmd.Flags().BoolVar(&mock, "mock", false, "read the embedded demo organization")
	cmd.Flags().StringVar(&profile, "profile", "", "AWS shared config profile")
	cmd.Flags().StringVar(&region, "region", "", "AWS region (default us-east-1)")
	cmd.Flags().StringVar(&lz, "lz", "", "landing zone code (requires --taxonomy; defaults to the taxonomy's one LZ)")
	cmd.Flags().StringVar(&taxonomy, "taxonomy", "", "taxonomy YAML: its one LZ is checked against the organization's management account")
	return cmd
}

// resolveLZ returns the LZ code for the records. With --taxonomy it is the taxonomy's one LZ (D-052);
// a --lz that names another code is an error that names both.
func resolveLZ(lz, taxonomy string) (string, error) {
	if taxonomy == "" {
		if lz != "" {
			return "", fmt.Errorf("--lz requires --taxonomy")
		}
		return "", nil
	}
	code, err := taxonomyLZ(taxonomy)
	if err != nil {
		return "", err
	}
	if lz != "" && lz != code {
		return "", fmt.Errorf("--lz %q is not the LZ %q of taxonomy %s", lz, code, taxonomy)
	}
	return code, nil
}

// taxonomyLZ returns the code of the one LZ in the taxonomy at path.
func taxonomyLZ(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read taxonomy %s: %w", path, err)
	}
	var doc struct {
		LZs []struct {
			Code string `yaml:"code"`
		} `yaml:"lzs"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return "", fmt.Errorf("parse taxonomy %s: %w", path, err)
	}
	if len(doc.LZs) != 1 {
		return "", fmt.Errorf("taxonomy %s has %d LZs, want exactly one", path, len(doc.LZs))
	}
	return doc.LZs[0].Code, nil
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
