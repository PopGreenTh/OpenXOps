package enrich

import (
	"fmt"
	"sort"
	"strings"

	"github.com/PopGreenTh/OpenXOps/record"
)

const (
	lzUnalloc  = "LZ-UNALLOC"
	appUnalloc = "APP-UNALLOC"
	envUnalloc = "ENV-UNALLOC"
	envUnspec  = "ENV-UNSPEC"
	prjUnalloc = "PRJ-UNALLOC"
	buUnalloc  = "BU-UNALLOC"
	ccUnalloc  = "CC-UNALLOC"
	cmdMap     = "finops enrich map"
)

type ouInfo struct {
	id, path string
	children bool
}

type inv struct {
	ous      map[string]ouInfo
	byPath   map[string]string
	accounts map[string]record.Record
	orgMgmt  map[string]string // org_id -> management_account
}

func buildInventory(recs []record.Record) inv {
	in := inv{
		ous:      map[string]ouInfo{},
		byPath:   map[string]string{},
		accounts: map[string]record.Record{},
		orgMgmt:  map[string]string{},
	}
	childCount := map[string]int{}
	for _, r := range recs {
		if r.Kind() != "ou" {
			continue
		}
		if pid := strID(r["parent_id"]); pid != "" {
			childCount[pid]++
		}
	}
	for _, r := range recs {
		switch r.Kind() {
		case "ou":
			id := strID(r["id"])
			path := strID(r["path"])
			in.ous[id] = ouInfo{id: id, path: path, children: childCount[id] > 0}
			in.byPath[path] = id
			org := strID(r["org_id"])
			if org != "" {
				in.orgMgmt[org] = strID(r["management_account"])
			}
		case "account":
			id := strID(r["id"])
			in.accounts[id] = r
			org := strID(r["org_id"])
			if org != "" {
				in.orgMgmt[org] = strID(r["management_account"])
			}
		}
	}
	return in
}

func lzByMgmt(t Taxonomy, mgmt string) (*LZ, bool) {
	for i := range t.LZs {
		if t.LZs[i].ManagementAccount == mgmt {
			return &t.LZs[i], true
		}
	}
	return nil, false
}

func appByCode(t Taxonomy, code string) (App, bool) {
	for _, a := range t.Apps {
		if a.Code == code {
			return a, true
		}
	}
	return App{}, false
}

func accountLink(t Taxonomy, id string) (*AccountLink, bool) {
	for i := range t.Accounts {
		if t.Accounts[i].Account == id {
			return &t.Accounts[i], true
		}
	}
	return nil, false
}

func ouChain(in inv, ac record.Record) []string {
	idPath := strID(ac["ou_id_path"])
	if idPath == "" {
		return nil
	}
	return strings.Split(idPath, "/")
}

func resolveOULinkTarget(in inv, ouRef string) (path string, id string, ok bool) {
	if strings.HasPrefix(ouRef, "/") {
		id, ok = in.byPath[ouRef]
		return ouRef, id, ok
	}
	info, ok := in.ous[ouRef]
	return info.path, ouRef, ok
}

type ouLinkEffect struct {
	app, env  string
	appPath   string
	envPath   string
	appOnLeaf bool
}

func applyOULinks(t Taxonomy, lzCode string, in inv, findings *[]record.Record) map[string]ouLinkEffect {
	effects := map[string]ouLinkEffect{}
	for _, link := range t.OULinks {
		if link.LZ != lzCode {
			continue
		}
		path, id, ok := resolveOULinkTarget(in, link.OU)
		if !ok {
			*findings = append(*findings, record.Finding("warning", "ou-not-found", link.OU,
				fmt.Sprintf("OU %q not found in inventory", link.OU), cmdMap))
			continue
		}
		eff := effects[id]
		if link.App != "" {
			if info, has := in.ous[id]; has && !info.children {
				*findings = append(*findings, record.Finding("warning", "app-on-leaf-ou", path,
					fmt.Sprintf("app link on leaf OU %s", path), cmdMap))
			} else {
				eff.app = link.App
				eff.appPath = path
			}
		}
		if link.Environment != "" {
			eff.env = link.Environment
			eff.envPath = path
		}
		effects[id] = eff
	}
	return effects
}

func deepestOURule(chain []string, effects map[string]ouLinkEffect, pickApp bool) (value, rule string) {
	for i := len(chain) - 1; i >= 0; i-- {
		id := chain[i]
		eff, ok := effects[id]
		if !ok {
			continue
		}
		if pickApp && eff.app != "" {
			return eff.app, "ou:" + eff.appPath
		}
		if !pickApp && eff.env != "" {
			return eff.env, "ou:" + eff.envPath
		}
	}
	return "", ""
}

func resolveAccount(t Taxonomy, in inv, ac record.Record, ouEffects map[string]ouLinkEffect) ([]record.Record, []record.Record) {
	var findings []record.Record
	acctID := strID(ac["id"])
	mgmt := strID(ac["management_account"])
	org := strID(ac["org_id"])
	ouPath := strID(ac["ou_path"])

	lz, ok := lzByMgmt(t, mgmt)
	if !ok {
		findings = append(findings, record.Finding("warning", "lz-unknown", org,
			fmt.Sprintf("no LZ for management account %s", mgmt), cmdMap))
		rec := allocationRecord(acctID, strID(ac["name"]), lzUnalloc, ouPath, prjUnalloc, appUnalloc, envUnalloc, buUnalloc, ccUnalloc, "unalloc", "unalloc", false)
		return []record.Record{rec}, findings
	}

	chain := ouChain(in, ac)
	link, hasLink := accountLink(t, acctID)

	var apps []string
	var appRule string
	if hasLink && len(link.Apps) > 0 {
		apps = append([]string{}, link.Apps...)
		appRule = "account"
	} else if app, rule := deepestOURule(chain, ouEffects, true); app != "" {
		apps = []string{app}
		appRule = rule
	} else if lz.Default != nil && lz.Default.App != "" {
		apps = []string{lz.Default.App}
		appRule = "lz-default"
	} else if t.Defaults != nil && t.Defaults.App != "" {
		apps = []string{t.Defaults.App}
		appRule = "default"
	} else {
		findings = append(findings, record.Finding("warning", "account-unmapped", acctID,
			fmt.Sprintf("account %s has no app mapping", acctID), cmdMap))
		apps = []string{appUnalloc}
		appRule = "unalloc"
	}

	env, envRule := resolveEnvironment(t, in, acctID, chain, ouEffects, link, hasLink, lz, apps)

	shared := len(apps) > 1
	var allocs []record.Record
	for _, appCode := range apps {
		project, bu, cc := deriveHierarchy(t, appCode)
		envOut, envRuleOut := env, envRule
		if appCode != appUnalloc {
			if !envUnderApp(t, appCode, env) {
				findings = append(findings, record.Finding("warning", "environment-not-in-app", acctID+"/"+appCode,
					fmt.Sprintf("environment %q not under app %q", env, appCode), cmdMap))
				envOut = envUnspec
				envRuleOut = "unspec"
			}
		}
		allocs = append(allocs, allocationRecord(acctID, strID(ac["name"]), lz.Code, ouPath, project, appCode, envOut, bu, cc, appRule, envRuleOut, shared))
	}
	return allocs, findings
}

func resolveEnvironment(t Taxonomy, in inv, acctID string, chain []string, effects map[string]ouLinkEffect, link *AccountLink, hasLink bool, lz *LZ, apps []string) (string, string) {
	if hasLink && link.Environment != "" {
		return link.Environment, "account"
	}
	if env, rule := deepestOURule(chain, effects, false); env != "" {
		return env, rule
	}
	if lz.Default != nil && lz.Default.Environment != "" {
		return lz.Default.Environment, "lz-default"
	}
	if t.Defaults != nil && t.Defaults.Environment != "" {
		return t.Defaults.Environment, "default"
	}
	appResolved := len(apps) == 1 && apps[0] != appUnalloc
	if appResolved {
		return envUnspec, "unspec"
	}
	return envUnalloc, "unalloc"
}

func envUnderApp(t Taxonomy, appCode, env string) bool {
	if env == "" || env == envUnspec || env == envUnalloc {
		return true
	}
	app, ok := appByCode(t, appCode)
	if !ok {
		return false
	}
	for _, e := range app.Environments {
		if e.Code == env {
			return true
		}
	}
	return false
}

func deriveHierarchy(t Taxonomy, appCode string) (project, bu, cc string) {
	if appCode == appUnalloc {
		return prjUnalloc, buUnalloc, ccUnalloc
	}
	app, ok := appByCode(t, appCode)
	if !ok {
		return prjUnalloc, buUnalloc, ccUnalloc
	}
	project = app.Project
	cc = app.CostCenter
	if pr, ok := prjByCode(t, app.Project); ok {
		bu = pr.BusinessUnit
	} else {
		bu = buUnalloc
	}
	return project, bu, cc
}

func allocationRecord(acctID, name, lz, ouPath, project, app, env, bu, cc, appRule, envRule string, shared bool) record.Record {
	return record.Record{
		"kind":             "allocation",
		"account_id":       acctID,
		"account_name":     name,
		"lz":               lz,
		"ou_path":          ouPath,
		"project":          project,
		"app":              app,
		"environment":      env,
		"business_unit":    bu,
		"cost_center":      cc,
		"app_rule":         appRule,
		"environment_rule": envRule,
		"shared":           shared,
	}
}

func enrichCost(t Taxonomy, in inv, cost record.Record, byAcct map[string][]record.Record) (record.Record, []record.Record) {
	var findings []record.Record
	out := copyRecord(cost)
	acctID := strID(cost["account_id"])
	mgmt := strID(cost["management_account"])

	allocs, inInv := byAcct[acctID]
	if !inInv {
		lzCode := lzUnalloc
		if lz, ok := lzByMgmt(t, mgmt); ok {
			lzCode = lz.Code
		}
		out["lz"] = lzCode
		out["project"] = prjUnalloc
		out["app"] = appUnalloc
		out["environment"] = envUnalloc
		out["business_unit"] = buUnalloc
		out["cost_center"] = ccUnalloc
		out["app_rule"] = "unalloc"
		out["environment_rule"] = "unalloc"
		out["shared"] = false
		findings = append(findings, record.Finding("warning", "account-not-in-inventory", acctID,
			fmt.Sprintf("cost for unknown account %s", acctID), cmdMap))
		return out, findings
	}

	if len(allocs) == 1 && allocs[0]["shared"] != true {
		a := allocs[0]
		out["lz"] = a["lz"]
		out["project"] = a["project"]
		out["app"] = a["app"]
		out["environment"] = a["environment"]
		out["business_unit"] = a["business_unit"]
		out["cost_center"] = a["cost_center"]
		out["app_rule"] = a["app_rule"]
		out["environment_rule"] = a["environment_rule"]
		out["shared"] = false
		return out, findings
	}

	// shared account
	out["lz"] = allocs[0]["lz"]
	out["environment"] = allocs[0]["environment"]
	out["environment_rule"] = allocs[0]["environment_rule"]
	out["shared"] = true
	out["project"] = ""
	out["app"] = ""
	out["business_unit"] = ""
	out["cost_center"] = ""
	out["app_rule"] = ""
	return out, findings
}

func copyRecord(r record.Record) record.Record {
	out := record.Record{}
	for k, v := range r {
		out[k] = v
	}
	return out
}

func checkProjectTags(t Taxonomy, in inv, rec record.Record, byAcct map[string][]record.Record) []record.Record {
	tag := strings.TrimSpace(strID(rec["project_tag"]))
	if tag == "" {
		return nil
	}
	acctID := strID(rec["account_id"])
	allocs := byAcct[acctID]
	if len(allocs) == 0 {
		return nil
	}
	allowed := map[string]struct{}{}
	for _, a := range allocs {
		allowed[strID(a["project"])] = struct{}{}
	}
	if _, ok := allowed[tag]; ok {
		return nil
	}
	return []record.Record{record.Finding("warning", "project-tag-mismatch", acctID+"/"+tag,
		fmt.Sprintf("project tag %q does not match taxonomy project(s) for account %s", tag, acctID), cmdMap)}
}

func staleAccountEntries(t Taxonomy, in inv) []record.Record {
	var out []record.Record
	for _, entry := range t.Accounts {
		if _, ok := in.accounts[entry.Account]; !ok {
			out = append(out, record.Finding("warning", "account-not-discovered", entry.Account,
				fmt.Sprintf("taxonomy account entry %s not in inventory", entry.Account), cmdMap))
		}
	}
	return out
}

func sortFindings(findings []record.Record) {
	sort.Slice(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a["code"] != b["code"] {
			return strID(a["code"]) < strID(b["code"])
		}
		return strID(a["subject"]) < strID(b["subject"])
	})
}
