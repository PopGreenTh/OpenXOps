package taxonomy

import (
	"strings"

	"github.com/PopGreenTh/OpenXOps/record/urn"
)

// Unknown is the OU code of an account that has cost but is not in the inventory (D-059).
const Unknown = "unknown"

// DiscoveredOU is an OU as the cloud names it: its id, path and display name.
type DiscoveredOU struct {
	ID, Path, Name string
}

// DiscoveredAccount is an account as the cloud names it: its id and display name. Name is "" when none is
// known, as for an account that has cost but is not in the inventory.
type DiscoveredAccount struct {
	ID, Name string
}

// Cloud holds the codes the taxonomy gives to the OUs and accounts of one LZ (D-059). The zero
// value lists none, as under an LZ the taxonomy does not know.
type Cloud struct {
	ouByID   map[string]string // OU id -> code, for entries with a ref_id
	ouByPath map[string]string // path -> code, for entries without one
	acctByID map[string]string // account id -> code
}

// Cloud returns the codes the taxonomy lists for the LZ with this code, in any case.
func (t Taxonomy) Cloud(lz string) Cloud {
	c := Cloud{ouByID: map[string]string{}, ouByPath: map[string]string{}, acctByID: map[string]string{}}
	for _, l := range t.LZs {
		if !Same(l.Code, lz) {
			continue
		}
		for _, ou := range l.OUPaths {
			switch {
			case ou.RefID != "":
				c.ouByID[ou.RefID] = ou.Code
			case ou.Path != "":
				c.ouByPath[ou.Path] = ou.Code
			}
			for _, a := range ou.Accounts {
				if a.RefID != "" {
					c.acctByID[a.RefID] = a.Code
				}
			}
		}
	}
	return c
}

// OUCode is the code of a discovered OU: the `code` of the `ou_paths` entry that matches it (by
// ref_id when the entry has one, else by path), else the slug of its name. The root, `/`, is
// `root` unless the taxonomy lists it.
func (c Cloud) OUCode(ou DiscoveredOU) string {
	if code, ok := c.ouByID[ou.ID]; ok && ou.ID != "" {
		return code
	}
	if code, ok := c.ouByPath[ou.Path]; ok {
		return code
	}
	if strings.Trim(ou.Path, "/") == "" {
		return "root"
	}
	return urn.Slug(ou.Name)
}

// AccountCode is the code of an account: the `code` of its `accounts` entry, else the slug of its
// name, else its id when no name is known.
func (c Cloud) AccountCode(a DiscoveredAccount) string {
	if code, ok := c.acctByID[a.ID]; ok {
		return code
	}
	if a.Name != "" {
		return urn.Slug(a.Name)
	}
	return a.ID
}

// OUURN is the URN of an OU: `<lz>:org:<OU code>` (D-059). lz is the LZ's URN.
func (c Cloud) OUURN(lz urn.URN, ou DiscoveredOU) urn.URN {
	return lz.Child("org", c.OUCode(ou))
}

// AccountURN is the URN of an account: `<URN of its OU>:acc:<account code>`.
func (c Cloud) AccountURN(lz urn.URN, ou DiscoveredOU, a DiscoveredAccount) urn.URN {
	return c.OUURN(lz, ou).Child("acc", c.AccountCode(a))
}

// UnknownOUURN is the URN of the OU an account outside the inventory sits under.
func UnknownOUURN(lz urn.URN) urn.URN { return lz.Child("org", Unknown) }

// UnknownAccountURN is the URN of an account that has cost but is not in the inventory: under
// `org:unknown`, with its taxonomy code, else its id.
func (c Cloud) UnknownAccountURN(lz urn.URN, id string) urn.URN {
	return UnknownOUURN(lz).Child("acc", c.AccountCode(DiscoveredAccount{ID: id}))
}

// PathName is the last segment of an OU path (`/Workloads/Prod` gives `Prod`), "" for the root.
func PathName(path string) string {
	path = strings.TrimRight(path, "/")
	return path[strings.LastIndex(path, "/")+1:]
}
