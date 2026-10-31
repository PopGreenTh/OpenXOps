package aws

import (
	"context"
	"fmt"
	"testing"
	"time"

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
				{Id: aws.String("200000000001"), Name: aws.String("a1"), Email: aws.String("a1@example.com"), Status: types.AccountStatusActive,
					JoinedTimestamp: aws.Time(time.Date(2020, 1, 15, 8, 0, 31, 0, time.UTC)), JoinedMethod: types.AccountJoinedMethodCreated},
			},
			NextToken: aws.String(pagingToken),
		}, nil
	}
	if aws.ToString(params.NextToken) != pagingToken {
		return nil, fmt.Errorf("unexpected NextToken %q", aws.ToString(params.NextToken))
	}
	return &organizations.ListAccountsForParentOutput{
		Accounts: []types.Account{
			{Id: aws.String("200000000002"), Name: aws.String("a2"), Email: aws.String("a2@example.com"), Status: types.AccountStatusActive,
				JoinedTimestamp: aws.Time(time.Date(2021, 4, 12, 3, 15, 27, 0, time.UTC)), JoinedMethod: types.AccountJoinedMethodInvited},
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
	if accs[0].JoinedAt != "2020-01-15T08:00:31Z" || accs[0].JoinedMethod != "CREATED" ||
		accs[1].JoinedAt != "2021-04-12T03:15:27Z" || accs[1].JoinedMethod != "INVITED" {
		t.Fatalf("joined = %+v", accs)
	}
}

func TestJoinedAt(t *testing.T) {
	bangkok := time.FixedZone("ICT", 7*60*60)
	tests := []struct {
		name string
		in   *time.Time
		want string
	}{
		{"nil", nil, ""},
		{"utc", aws.Time(time.Date(2021, 4, 12, 3, 15, 27, 0, time.UTC)), "2021-04-12T03:15:27Z"},
		{"non-utc with fraction", aws.Time(time.Date(2021, 4, 12, 10, 15, 27, 987654321, bangkok)), "2021-04-12T03:15:27Z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := joinedAt(tt.in); got != tt.want {
				t.Fatalf("joinedAt = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAccountFromAWSMissingJoined(t *testing.T) {
	ac := accountFromAWS(types.Account{Id: aws.String("200000000001")})
	if ac.JoinedAt != "" || ac.JoinedMethod != "" {
		t.Fatalf("joined = %q %q, want empty", ac.JoinedAt, ac.JoinedMethod)
	}
}
