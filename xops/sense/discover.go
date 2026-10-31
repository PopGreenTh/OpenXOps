package sense

import (
	"sort"

	"github.com/PopGreenTh/OpenXOps/aws"
	"github.com/PopGreenTh/OpenXOps/record"
)

// BuildRecords turns a snapshot into ou and account records plus path findings.
func BuildRecords(snap aws.Snapshot, lz string) []record.Record {
	views, findings := Paths(snap.OrganizationalUnits)
	ous := append([]aws.OU(nil), snap.OrganizationalUnits...)
	sort.Slice(ous, func(i, j int) bool { return ous[i].ID < ous[j].ID })
	accs := append([]aws.Account(nil), snap.Accounts...)
	sort.Slice(accs, func(i, j int) bool { return accs[i].ID < accs[j].ID })

	var out []record.Record
	for _, ou := range ous {
		v := views[ou.ID]
		rec := record.Record{
			"kind":               "ou",
			"csp":                "aws",
			"org_id":             snap.OrgID,
			"management_account": snap.ManagementAccount,
			"id":                 ou.ID,
			"name":               ou.Name,
			"parent_id":          ou.ParentID,
			"path":               v.path,
			"id_path":            v.idPath,
			"has_children":       ou.HasChildren,
		}
		if lz != "" {
			rec["lz"] = lz
		}
		out = append(out, rec)
	}
	for _, ac := range accs {
		parent := views[ac.ParentID]
		rec := record.Record{
			"kind":               "account",
			"csp":                "aws",
			"org_id":             snap.OrgID,
			"management_account": snap.ManagementAccount,
			"id":                 ac.ID,
			"name":               ac.Name,
			"email":              ac.Email,
			"status":             ac.Status,
			"parent_id":          ac.ParentID,
			"ou_path":            parent.path,
			"ou_id_path":         parent.idPath,
			"joined_at":          ac.JoinedAt,
			"joined_method":      ac.JoinedMethod,
		}
		if lz != "" {
			rec["lz"] = lz
		}
		out = append(out, rec)
	}
	out = append(out, findings...)
	return out
}
