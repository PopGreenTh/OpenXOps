package aws

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
)

const pagingToken = "page-2"

// pagingOrganizationsFake returns two API pages joined by NextToken for OU and account lists.
type pagingOrganizationsFake struct{}

func (pagingOrganizationsFake) DescribeOrganization(ctx context.Context, params *organizations.DescribeOrganizationInput, optFns ...func(*organizations.Options)) (*organizations.DescribeOrganizationOutput, error) {
	return &organizations.DescribeOrganizationOutput{
		Organization: &types.Organization{
			Id:              aws.String("o-test"),
			MasterAccountId: aws.String("111111111111"),
		},
	}, nil
}

func (pagingOrganizationsFake) ListRoots(ctx context.Context, params *organizations.ListRootsInput, optFns ...func(*organizations.Options)) (*organizations.ListRootsOutput, error) {
	if params.NextToken != nil {
		return &organizations.ListRootsOutput{}, nil
	}
	return &organizations.ListRootsOutput{
		Roots: []types.Root{{Id: aws.String("r-1")}},
	}, nil
}

func (pagingOrganizationsFake) ListOrganizationalUnitsForParent(ctx context.Context, params *organizations.ListOrganizationalUnitsForParentInput, optFns ...func(*organizations.Options)) (*organizations.ListOrganizationalUnitsForParentOutput, error) {
	if params.NextToken == nil {
		return &organizations.ListOrganizationalUnitsForParentOutput{
			OrganizationalUnits: []types.OrganizationalUnit{
				{Id: aws.String("ou-page-1"), Name: aws.String("FromPageOne")},
			},
			NextToken: aws.String(pagingToken),
		}, nil
	}
	if aws.ToString(params.NextToken) != pagingToken {
		return nil, fmt.Errorf("unexpected NextToken %q", aws.ToString(params.NextToken))
	}
	return &organizations.ListOrganizationalUnitsForParentOutput{
		OrganizationalUnits: []types.OrganizationalUnit{
			{Id: aws.String("ou-page-2"), Name: aws.String("FromPageTwo")},
		},
	}, nil
}

func (pagingOrganizationsFake) ListAccountsForParent(ctx context.Context, params *organizations.ListAccountsForParentInput, optFns ...func(*organizations.Options)) (*organizations.ListAccountsForParentOutput, error) {
	if params.NextToken == nil {
		return &organizations.ListAccountsForParentOutput{
			Accounts: []types.Account{
				{Id: aws.String("200000000001"), Name: aws.String("a1"), Email: aws.String("a1@example.com"), Status: types.AccountStatusActive},
			},
			NextToken: aws.String(pagingToken),
		}, nil
	}
	if aws.ToString(params.NextToken) != pagingToken {
		return nil, fmt.Errorf("unexpected NextToken %q", aws.ToString(params.NextToken))
	}
	return &organizations.ListAccountsForParentOutput{
		Accounts: []types.Account{
			{Id: aws.String("200000000002"), Name: aws.String("a2"), Email: aws.String("a2@example.com"), Status: types.AccountStatusActive},
		},
	}, nil
}

func TestOrgClientListOrganizationalUnitsForParentPaging(t *testing.T) {
	fake := pagingOrganizationsFake{}
	client := &OrgClient{listOUs: fake}
	ous, err := client.ListOrganizationalUnitsForParent(context.Background(), "r-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(ous) != 2 || ous[0].ID != "ou-page-1" || ous[1].ID != "ou-page-2" {
		t.Fatalf("ous = %+v", ous)
	}
}

func TestOrgClientListAccountsForParentPaging(t *testing.T) {
	fake := pagingOrganizationsFake{}
	client := &OrgClient{listAccounts: fake}
	accs, err := client.ListAccountsForParent(context.Background(), "r-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(accs) != 2 || accs[0].ID != "200000000001" || accs[1].ID != "200000000002" {
		t.Fatalf("accs = %+v", accs)
	}
}
