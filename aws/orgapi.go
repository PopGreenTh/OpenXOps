package aws

import "context"

// OrgAPI lists an AWS Organization. Implementations use the retry wrapper for each call.
type OrgAPI interface {
	DescribeOrganization(ctx context.Context) (orgID, managementAccount string, err error)
	ListRoots(ctx context.Context) ([]string, error)
	ListOrganizationalUnitsForParent(ctx context.Context, parentID string) ([]OU, error)
	ListAccountsForParent(ctx context.Context, parentID string) ([]Account, error)
}

// DiscoverSnapshot walks the organization through api with at most maxConcurrent list calls.
func DiscoverSnapshot(ctx context.Context, api OrgAPI, maxConcurrent int) (Snapshot, error) {
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	var orgID, mgmt string
	if err := Retry(ctx, func() error {
		var err error
		orgID, mgmt, err = api.DescribeOrganization(ctx)
		return err
	}); err != nil {
		return Snapshot{}, err
	}
	var roots []string
	if err := Retry(ctx, func() error {
		var err error
		roots, err = api.ListRoots(ctx)
		return err
	}); err != nil {
		return Snapshot{}, err
	}
	sem := make(chan struct{}, maxConcurrent)
	type result struct {
		ous      []OU
		accounts []Account
		err      error
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
			sem <- struct{}{}
			resCh := make(chan result, 1)
			go func(pid string) {
				defer func() { <-sem }()
				var res result
				res.err = Retry(ctx, func() error {
					var err error
					res.ous, err = api.ListOrganizationalUnitsForParent(ctx, pid)
					return err
				})
				if res.err != nil {
					resCh <- res
					return
				}
				res.err = Retry(ctx, func() error {
					var err error
					res.accounts, err = api.ListAccountsForParent(ctx, pid)
					return err
				})
				resCh <- res
			}(parentID)
			res := <-resCh
			if res.err != nil {
				return Snapshot{}, res.err
			}
			for _, ou := range res.ous {
				ou.ParentID = parentID
				allOUs = append(allOUs, ou)
				queue = append(queue, ou.ID)
			}
			for _, ac := range res.accounts {
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
