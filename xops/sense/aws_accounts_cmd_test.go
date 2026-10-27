package sense

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"

	"github.com/PopGreenTh/OpenXOps/aws"
	awssdk "github.com/aws/aws-sdk-go-v2/aws"
)

type listRootsErrOrg struct{}

func (listRootsErrOrg) DescribeOrganization(ctx context.Context) (string, string, error) {
	return "o-fake", "111111111111", nil
}

func (listRootsErrOrg) ListRoots(ctx context.Context) ([]string, error) {
	return nil, errors.New("simulated discovery failure")
}

func (listRootsErrOrg) ListOrganizationalUnitsForParent(ctx context.Context, parentID string) ([]aws.OU, error) {
	return nil, nil
}

func (listRootsErrOrg) ListAccountsForParent(ctx context.Context, parentID string) ([]aws.Account, error) {
	return nil, nil
}

func TestAWSAccountsCommandDiscoveryError(t *testing.T) {
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	old := newOrgAPI
	t.Cleanup(func() { newOrgAPI = old })
	newOrgAPI = func(cfg awssdk.Config) aws.OrgAPI { return listRootsErrOrg{} }

	var out bytes.Buffer
	cmd := awsAccountsCommand()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetContext(context.Background())
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--region", "us-east-1"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected discovery error")
	}
	if out.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", out.String())
	}
}

func TestAWSAccountsCommandLoadConfigMissingProfile(t *testing.T) {
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_PROFILE", "openxops-no-such-profile-7589")
	cmd := awsAccountsCommand()
	cmd.SetContext(context.Background())
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)
	cmd.SetArgs([]string{"--profile", "openxops-no-such-profile-7589"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing profile")
	}
}
