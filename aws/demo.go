package aws

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed testdata/demo-org.json
var demoOrgJSON []byte

type demoNode struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	OrganizationalUnits []demoNode `json:"organizational_units"`
	Accounts            []Account  `json:"accounts"`
}

type demoFile struct {
	OrgID             string   `json:"org_id"`
	ManagementAccount string   `json:"management_account"`
	Root              demoNode `json:"root"`
}

// DemoSnapshot returns the embedded demo organization as a snapshot.
func DemoSnapshot() (Snapshot, error) {
	var file demoFile
	if err := json.Unmarshal(demoOrgJSON, &file); err != nil {
		return Snapshot{}, fmt.Errorf("demo organization: %w", err)
	}
	var ous []OU
	var accounts []Account
	walkDemo(file.Root, "", &ous, &accounts)
	for i := range ous {
		ous[i].HasChildren = childOUExists(ous[i].ID, ous)
	}
	return Snapshot{
		OrgID:               file.OrgID,
		ManagementAccount:   file.ManagementAccount,
		OrganizationalUnits: ous,
		Accounts:            accounts,
	}, nil
}

func walkDemo(node demoNode, parentID string, ous *[]OU, accounts *[]Account) {
	*ous = append(*ous, OU{
		ID:       node.ID,
		Name:     node.Name,
		ParentID: parentID,
	})
	for _, ac := range node.Accounts {
		ac.ParentID = node.ID
		*accounts = append(*accounts, ac)
	}
	for _, child := range node.OrganizationalUnits {
		walkDemo(child, node.ID, ous, accounts)
	}
}
