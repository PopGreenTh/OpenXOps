package sense

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/PopGreenTh/OpenXOps/aws"
	"github.com/PopGreenTh/OpenXOps/record"
	awssdk "github.com/aws/aws-sdk-go-v2/aws"
)

var errSimulatedDiscovery = errors.New("simulated discovery failure")

type listRootsErrOrg struct{}

func (listRootsErrOrg) DescribeOrganization(ctx context.Context) (string, string, error) {
	return "o-fake", "111111111111", nil
}

func (listRootsErrOrg) ListRoots(ctx context.Context) ([]string, error) {
	return nil, errSimulatedDiscovery
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
	cmd.SetArgs([]string{"--tenant", "testtnt", "--region", "us-east-1"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected discovery error")
	}
	if !errors.Is(err, errSimulatedDiscovery) {
		t.Fatalf("error = %v, want %v", err, errSimulatedDiscovery)
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
	cmd.SetArgs([]string{"--tenant", "testtnt", "--profile", "openxops-no-such-profile-7589"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing profile")
	}
}

func TestAWSAccountsCommandWithoutTenantFailsBeforeReading(t *testing.T) {
	old := newOrgAPI
	t.Cleanup(func() { newOrgAPI = old })
	called := false
	newOrgAPI = func(cfg awssdk.Config) aws.OrgAPI {
		called = true
		return listRootsErrOrg{}
	}
	for _, args := range [][]string{
		{"--mock"},
		{"--region", "us-east-1"},
		{"--tenant", "Not_Valid", "--mock"},
	} {
		var out bytes.Buffer
		cmd := awsAccountsCommand()
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		cmd.SetContext(context.Background())
		cmd.SetOut(&out)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs(args)
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "tenant") {
			t.Fatalf("args %v: error = %v, want a --tenant error", args, err)
		}
		if called || out.Len() != 0 {
			t.Fatalf("args %v: command read the organization or wrote %q", args, out.String())
		}
	}
}

func TestAWSAccountsMockEveryRecordHasTenant(t *testing.T) {
	var out bytes.Buffer
	cmd := awsAccountsCommand()
	cmd.SetContext(context.Background())
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--tenant", "testtnt", "--mock"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	recs, err := record.Read(&out, "stdout", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) == 0 {
		t.Fatal("no records")
	}
	for i, r := range recs {
		if r["tenant"] != "testtnt" {
			t.Fatalf("record %d has tenant %v: %v", i+1, r["tenant"], r)
		}
	}
}
