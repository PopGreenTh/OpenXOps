package aws

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
