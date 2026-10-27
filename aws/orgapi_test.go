package aws

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/aws/smithy-go"
)

type throttleErr struct{}

func (throttleErr) Error() string                 { return "ThrottlingException" }
func (throttleErr) ErrorCode() string             { return "ThrottlingException" }
func (throttleErr) ErrorMessage() string          { return "rate exceeded" }
func (throttleErr) ErrorFault() smithy.ErrorFault { return smithy.FaultClient }

type fakeOrg struct {
	describeCalls atomic.Int32
}

func (f *fakeOrg) DescribeOrganization(ctx context.Context) (string, string, error) {
	n := f.describeCalls.Add(1)
	if n == 1 {
		return "", "", throttleErr{}
	}
	return "o-fake", "111111111111", nil
}

func (f *fakeOrg) ListRoots(ctx context.Context) ([]string, error) {
	return []string{"r-root"}, nil
}

func (f *fakeOrg) ListOrganizationalUnitsForParent(ctx context.Context, parentID string) ([]OU, error) {
	if parentID == "r-root" {
		return []OU{{ID: "ou-a", Name: "Alpha"}}, nil
	}
	return nil, nil
}

func (f *fakeOrg) ListAccountsForParent(ctx context.Context, parentID string) ([]Account, error) {
	if parentID == "r-root" {
		return []Account{{ID: "200000000001", Name: "one", Email: "a@example.com", Status: "ACTIVE"}}, nil
	}
	return nil, nil
}

func TestDiscoverSnapshotRetryOnThrottle(t *testing.T) {
	snap, err := DiscoverSnapshot(context.Background(), &fakeOrg{}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if snap.OrgID != "o-fake" || len(snap.Accounts) != 1 {
		t.Fatalf("snap = %+v", snap)
	}
}

type nestedOrg struct{}

func (nestedOrg) DescribeOrganization(ctx context.Context) (string, string, error) {
	return "o-nested", "111111111111", nil
}

func (nestedOrg) ListRoots(ctx context.Context) ([]string, error) {
	return []string{"r-1"}, nil
}

func (nestedOrg) ListOrganizationalUnitsForParent(ctx context.Context, parentID string) ([]OU, error) {
	switch parentID {
	case "r-1":
		return []OU{{ID: "ou-1", Name: "Workloads"}}, nil
	case "ou-1":
		return []OU{{ID: "ou-2", Name: "Leaf"}}, nil
	default:
		return nil, nil
	}
}

func (nestedOrg) ListAccountsForParent(ctx context.Context, parentID string) ([]Account, error) {
	if parentID == "ou-2" {
		return []Account{{ID: "200000000001", Name: "acct", Email: "a@example.com", Status: "ACTIVE"}}, nil
	}
	return nil, nil
}

func TestDiscoverSnapshotNestedOUsAndAccounts(t *testing.T) {
	snap, err := DiscoverSnapshot(context.Background(), nestedOrg{}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.OrganizationalUnits) != 3 || len(snap.Accounts) != 1 {
		t.Fatalf("ous=%d accounts=%d", len(snap.OrganizationalUnits), len(snap.Accounts))
	}
}

func TestRetryNotRetryable(t *testing.T) {
	err := Retry(context.Background(), func() error {
		return errors.New("permanent")
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
