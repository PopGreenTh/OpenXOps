package sense

import (
	"bytes"
	"context"
	"os"
	"testing"
)

func TestAWSAccountsCommandDiscoveryError(t *testing.T) {
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_ACCESS_KEY_ID", "NOTAREALKEY")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "NOTAREALSECRET")
	cmd := awsAccountsCommand()
	cmd.SetContext(context.Background())
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--region", "us-east-1"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected live discovery to fail with invalid credentials")
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
