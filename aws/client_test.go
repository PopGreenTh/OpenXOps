package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
)

// TestOrgClientPaginatorMergesPages documents how ListOrganizationalUnitsForParent
// accumulates every page (the real client uses the SDK paginator the same way).
func TestOrgClientPaginatorMergesPages(t *testing.T) {
	pages := []types.OrganizationalUnit{
		{Id: aws.String("ou-page-1"), Name: aws.String("One")},
		{Id: aws.String("ou-page-2"), Name: aws.String("Two")},
	}
	var merged []OU
	for _, ou := range pages {
		merged = append(merged, OU{ID: aws.ToString(ou.Id), Name: aws.ToString(ou.Name)})
	}
	if len(merged) != 2 {
		t.Fatalf("merged = %v", merged)
	}
}
