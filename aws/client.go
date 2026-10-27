package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
)

// OrgClient calls AWS Organizations through the retry wrapper.
type OrgClient struct {
	svc *organizations.Client
}

// NewOrgClient returns an Organizations client.
func NewOrgClient(cfg aws.Config) *OrgClient {
	return &OrgClient{svc: organizations.NewFromConfig(cfg)}
}

func (c *OrgClient) DescribeOrganization(ctx context.Context) (string, string, error) {
	var orgID, mgmt string
	err := Retry(ctx, func() error {
		out, err := c.svc.DescribeOrganization(ctx, &organizations.DescribeOrganizationInput{})
		if err != nil {
			return err
		}
		if out.Organization == nil {
			return fmt.Errorf("DescribeOrganization: empty organization")
		}
		orgID = aws.ToString(out.Organization.Id)
		mgmt = aws.ToString(out.Organization.MasterAccountId)
		return nil
	})
	return orgID, mgmt, err
}

func (c *OrgClient) ListRoots(ctx context.Context) ([]string, error) {
	var ids []string
	err := Retry(ctx, func() error {
		ids = ids[:0]
		paginator := organizations.NewListRootsPaginator(c.svc, &organizations.ListRootsInput{})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return err
			}
			for _, r := range page.Roots {
				ids = append(ids, aws.ToString(r.Id))
			}
		}
		return nil
	})
	return ids, err
}

func (c *OrgClient) ListOrganizationalUnitsForParent(ctx context.Context, parentID string) ([]OU, error) {
	var out []OU
	err := Retry(ctx, func() error {
		out = out[:0]
		paginator := organizations.NewListOrganizationalUnitsForParentPaginator(c.svc, &organizations.ListOrganizationalUnitsForParentInput{
			ParentId: aws.String(parentID),
		})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return err
			}
			for _, ou := range page.OrganizationalUnits {
				out = append(out, OU{
					ID:   aws.ToString(ou.Id),
					Name: aws.ToString(ou.Name),
				})
			}
		}
		return nil
	})
	return out, err
}

func (c *OrgClient) ListAccountsForParent(ctx context.Context, parentID string) ([]Account, error) {
	var out []Account
	err := Retry(ctx, func() error {
		out = out[:0]
		paginator := organizations.NewListAccountsForParentPaginator(c.svc, &organizations.ListAccountsForParentInput{
			ParentId: aws.String(parentID),
		})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return err
			}
			for _, ac := range page.Accounts {
				out = append(out, accountFromAWS(ac))
			}
		}
		return nil
	})
	return out, err
}

func accountFromAWS(ac types.Account) Account {
	return Account{
		ID:     aws.ToString(ac.Id),
		Name:   aws.ToString(ac.Name),
		Email:  aws.ToString(ac.Email),
		Status: string(ac.Status),
	}
}
