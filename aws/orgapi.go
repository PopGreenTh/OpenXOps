package aws

import (
	"context"
	"log/slog"
)

// Snapshot is a flattened AWS Organization tree for record building.
type Snapshot struct {
	OrgID               string
	ManagementAccount   string
	OrganizationalUnits []OU
	Accounts            []Account
}

// OU is one organizational unit before paths are computed.
type OU struct {
	ID          string
	Name        string
	ParentID    string
	HasChildren bool
}

// Account is one member account before OU paths are attached.
type Account struct {
	ID       string
	Name     string
	Email    string
	Status   string
	ParentID string
}

// OrgAPI lists an AWS Organization. Implementations use the retry wrapper for each call.
type OrgAPI interface {
	DescribeOrganization(ctx context.Context) (orgID, managementAccount string, err error)
	ListRoots(ctx context.Context) ([]string, error)
	ListOrganizationalUnitsForParent(ctx context.Context, parentID string) ([]OU, error)
	ListAccountsForParent(ctx context.Context, parentID string) ([]Account, error)
}

// DiscoverSnapshot walks the organization through api (list calls run serially per parent).
func DiscoverSnapshot(ctx context.Context, api OrgAPI) (Snapshot, error) {
	var orgID, mgmt string
	slog.Debug("organizations call", slog.String("operation", "DescribeOrganization"))
	if err := Retry(ctx, func() error {
		var err error
		orgID, mgmt, err = api.DescribeOrganization(ctx)
		return err
	}); err != nil {
		return Snapshot{}, err
	}
	var roots []string
	slog.Debug("organizations call", slog.String("operation", "ListRoots"))
	if err := Retry(ctx, func() error {
		var err error
		roots, err = api.ListRoots(ctx)
		return err
	}); err != nil {
		return Snapshot{}, err
	}
	var (
		allOUs  []OU
		allAccs []Account
	)
	for _, rootID := range roots {
		allOUs = append(allOUs, OU{ID: rootID, Name: "Root", ParentID: ""})
		queue := []string{rootID}
		for len(queue) > 0 {
			parentID := queue[0]
			queue = queue[1:]
			var ous []OU
			slog.Debug("organizations call", slog.String("operation", "ListOrganizationalUnitsForParent"), slog.String("parent_id", parentID))
			if err := Retry(ctx, func() error {
				var err error
				ous, err = api.ListOrganizationalUnitsForParent(ctx, parentID)
				return err
			}); err != nil {
				return Snapshot{}, err
			}
			var accs []Account
			slog.Debug("organizations call", slog.String("operation", "ListAccountsForParent"), slog.String("parent_id", parentID))
			if err := Retry(ctx, func() error {
				var err error
				accs, err = api.ListAccountsForParent(ctx, parentID)
				return err
			}); err != nil {
				return Snapshot{}, err
			}
			for _, ou := range ous {
				ou.ParentID = parentID
				allOUs = append(allOUs, ou)
				queue = append(queue, ou.ID)
			}
			for _, ac := range accs {
				ac.ParentID = parentID
				allAccs = append(allAccs, ac)
			}
		}
	}
	for i := range allOUs {
		allOUs[i].HasChildren = childOUExists(allOUs[i].ID, allOUs)
	}
	return Snapshot{
		OrgID:               orgID,
		ManagementAccount:   mgmt,
		OrganizationalUnits: allOUs,
		Accounts:            allAccs,
	}, nil
}

func childOUExists(id string, ous []OU) bool {
	for _, ou := range ous {
		if ou.ParentID == id {
			return true
		}
	}
	return false
}
