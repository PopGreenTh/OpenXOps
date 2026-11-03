// Package taxonomy reads a tenant's taxonomy file: the three hierarchies of D-046, nested the way
// people read them (D-060). Every command that takes --taxonomy reads it through this package.
package taxonomy

import "strings"

// Node is what every node has: a code and a name, and optionally the URN written in the file
// (checked against the one built from the codes, D-058) and an alias.
type Node struct {
	Code  string `yaml:"code"`
	Name  string `yaml:"name"`
	URN   string `yaml:"urn,omitempty"`
	Alias string `yaml:"alias,omitempty"`
}

// Taxonomy is the whole file.
type Taxonomy struct {
	Tenant        Node
	BusinessUnits []BusinessUnit
	Projects      []Project
	// LZs holds the entries of `lzs` or of `lz_conns`; a tenant has exactly one (D-052).
	LZs      []LZ
	Defaults *Defaults
	Tags     *TagKeys
}

// BusinessUnit is a business unit with the cost centers under it.
type BusinessUnit struct {
	Node        `yaml:",inline"`
	CostCenters []CostCenter `yaml:"cost_centers"`
}

// CostCenter is a cost center. OwnerTeams are read and not used (D-057).
type CostCenter struct {
	Node       `yaml:",inline"`
	OwnerTeams []Node `yaml:"owner_teams"`
}

// Project is a project. RefURNBU names its business unit.
type Project struct {
	Node     `yaml:",inline"`
	RefURNBU string `yaml:"ref_urn_bu"`
	Apps     []App  `yaml:"apps"`
}

// App is an App. RefURNCCT names its cost center.
type App struct {
	Node         `yaml:",inline"`
	RefURNCCT    string        `yaml:"ref_urn_cct"`
	Environments []Environment `yaml:"environments"`
}

// Environment is an Environment of one App. RefURNONT is read and not used (D-057).
type Environment struct {
	Node      `yaml:",inline"`
	RefURNONT string `yaml:"ref_urn_ont"`
}

// LZ is the landing zone, with the OUs the taxonomy knows about.
type LZ struct {
	Node              `yaml:",inline"`
	CSP               string    `yaml:"csp"`
	Currency          string    `yaml:"currency"`
	ManagementAccount string    `yaml:"management_account"`
	RefID             string    `yaml:"ref_id"`
	Default           *Defaults `yaml:"default"`
	RefURNPrjs        []string  `yaml:"ref_urn_prjs"` // read, not used (D-057)
	OUPaths           []OUPath  `yaml:"ou_paths"`
}

// Management returns the LZ's management account: management_account, else ref_id.
func (l LZ) Management() string {
	if l.ManagementAccount != "" {
		return l.ManagementAccount
	}
	return l.RefID
}

// OUPath is one OU of the LZ. It matches the discovered OU whose id equals RefID when it has one,
// else the one whose path equals Path (D-061).
type OUPath struct {
	Node       `yaml:",inline"`
	Path       string    `yaml:"path"`
	RefID      string    `yaml:"ref_id"`
	RefURNApps []string  `yaml:"ref_urn_apps"`
	RefURNEnv  string    `yaml:"ref_urn_env"`
	Accounts   []Account `yaml:"accounts"`
}

// Account is an account entry under its OU. RefID is the account id. Contact is read and not used.
type Account struct {
	Node       `yaml:",inline"`
	RefID      string   `yaml:"ref_id"`
	RefURNApps []string `yaml:"ref_urn_apps"`
	RefURNEnv  string   `yaml:"ref_urn_env"`
	Contact    string   `yaml:"contact"`
}

// Defaults are an App and an Environment, as codes.
type Defaults struct {
	App         string `yaml:"app"`
	Environment string `yaml:"environment"`
}

// TagKeys are the Cost Explorer tag keys for App and Project.
type TagKeys struct {
	App     string `yaml:"app"`
	Project string `yaml:"project"`
}

// TagKeys returns the App and Project tag keys, `App` and `Project` unless the file sets them.
func (t Taxonomy) TagKeys() (app, project string) {
	app, project = "App", "Project"
	if t.Tags != nil {
		if t.Tags.App != "" {
			app = t.Tags.App
		}
		if t.Tags.Project != "" {
			project = t.Tags.Project
		}
	}
	return app, project
}

// Same reports whether two codes are equal in any case, which is how uniqueness is compared.
func Same(a, b string) bool { return strings.EqualFold(a, b) }
