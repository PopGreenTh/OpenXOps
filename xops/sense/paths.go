package sense

import (
	"fmt"
	"sort"

	"github.com/PopGreenTh/OpenXOps/aws"
	"github.com/PopGreenTh/OpenXOps/record"
)

type ouView struct {
	ou     aws.OU
	path   string
	idPath string
}

// Paths computes name paths and id paths for every OU and detects ambiguous sibling names.
func Paths(ous []aws.OU) (map[string]ouView, []record.Record) {
	byID := make(map[string]aws.OU, len(ous))
	children := make(map[string][]aws.OU)
	for _, ou := range ous {
		byID[ou.ID] = ou
		children[ou.ParentID] = append(children[ou.ParentID], ou)
	}
	views := make(map[string]ouView, len(ous))
	var findings []record.Record
	var walk func(id string)
	walk = func(id string) {
		ou := byID[id]
		path := "/"
		idPath := ou.ID
		if ou.ParentID != "" {
			parent := views[ou.ParentID]
			if parent.path == "/" {
				path = "/" + ou.Name
			} else {
				path = parent.path + "/" + ou.Name
			}
			idPath = parent.idPath + "/" + ou.ID
		}
		views[id] = ouView{ou: ou, path: path, idPath: idPath}
		kids := children[id]
		sort.Slice(kids, func(i, j int) bool { return kids[i].ID < kids[j].ID })
		names := make(map[string]int)
		for _, k := range kids {
			names[k.Name]++
		}
		for name, n := range names {
			if n > 1 {
				subject := path
				if path == "/" {
					subject = "/" + name
				} else {
					subject = path + "/" + name
				}
				findings = append(findings, record.Finding(
					"warning", "ou-path-ambiguous", subject,
					fmt.Sprintf("%d organizational units named %q share parent %s", n, name, path),
					"xops sense aws-accounts",
				))
			}
		}
		for _, k := range kids {
			walk(k.ID)
		}
	}
	for _, ou := range ous {
		if ou.ParentID == "" {
			walk(ou.ID)
		}
	}
	return views, findings
}
