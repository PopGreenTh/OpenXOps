package enrich

import (
	"fmt"
	"sort"
	"strings"

	"github.com/PopGreenTh/OpenXOps/record"
	"github.com/PopGreenTh/OpenXOps/record/taxonomy"
	"github.com/PopGreenTh/OpenXOps/record/urn"
)

// System node ids (D-047): the reserved ids that stand for "no such node".
const (
	lzUnalloc  = "unalloc"
	appUnalloc = "unalloc"
	envUnalloc = "unalloc"
	envUnspec  = "unspec"
	prjUnalloc = "unalloc"
	buUnalloc  = "unalloc"
	ccUnalloc  = "unalloc"
	cmdMap     = "finops enrich map"
)

type ouInfo struct {
	id, path, name string
	children       bool
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
			in.ous[id] = ouInfo{id: id, path: path, name: strID(r["name"]), children: childCount[id] > 0}
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

// resolver holds what resolution needs: the taxonomy, its reference index, the discovered
// inventory and the taxonomy's OU entries matched to it.
type resolver struct {
	t     taxonomy.Taxonomy
	ix    taxonomy.Index
	in    inv
	views map[string]*lzView // by LZ code
}

// appPick is an App chosen for an account. ref is nil when the taxonomy does not define it.
type appPick struct {
	code string
	ref  *taxonomy.AppRef
}

// place is where an allocation lands in the business and project trees. ccBU is the business unit
// the cost center sits under, which can differ from the project's.
type place struct {
	project, app, env, bu, cc, ccBU string
}

var unallocPlace = place{prjUnalloc, appUnalloc, envUnalloc, buUnalloc, ccUnalloc, buUnalloc}

func (r *resolver) lzByMgmt(mgmt string) (*taxonomy.LZ, bool) {
	for i := range r.t.LZs {
		if r.t.LZs[i].Management() == mgmt {
			return &r.t.LZs[i], true
		}
	}
	return nil, false
}

// apps looks up App URNs. A URN that names no App is skipped: `validate` reports it.
func (r *resolver) apps(urns []string) []appPick {
	var out []appPick
	for _, u := range urns {
		if ref, ok := r.ix.LookupApp(u); ok {
			ref := ref
			out = append(out, appPick{code: ref.App.Code, ref: &ref})
		}
	}
	return out
}

// env looks up an Environment URN and returns its code and its App.
func (r *resolver) env(u string) (string, taxonomy.EnvRef, bool) {
	if u == "" {
		return "", taxonomy.EnvRef{}, false
	}
	ref, ok := r.ix.LookupEnv(u)
	if !ok {
		return "", taxonomy.EnvRef{}, false
	}
	return ref.Env.Code, ref, true
}

func ouChain(ac record.Record) []string {
	idPath := strID(ac["ou_id_path"])
	if idPath == "" {
		return nil
	}
	return strings.Split(idPath, "/")
}

// deepestOU returns the nearest OU on the account's path that sets an App (pickApp) or an
// Environment, with the rule that names it.
func deepestOU(chain []string, effects map[string]ouEffect, pickApp bool) (e ouEffect, rule string, ok bool) {
	for i := len(chain) - 1; i >= 0; i-- {
		eff, has := effects[chain[i]]
		if !has {
			continue
		}
		if pickApp && len(eff.apps) > 0 {
			return eff, "ou:" + eff.appPath, true
		}
		if !pickApp && eff.env != "" {
			return eff, "ou:" + eff.envPath, true
		}
	}
	return ouEffect{}, "", false
}

// defaultApp finds the App a `default` names, by code. A code the taxonomy does not define is
// still used, with no tree behind it.
func (r *resolver) defaultApp(code string) appPick {
	if ref, ok := r.ix.AppByCode(code); ok {
		return appPick{code: ref.App.Code, ref: &ref}
	}
	return appPick{code: code}
}

func (r *resolver) resolveAccount(ac record.Record, lzUnknownOrgs map[string]struct{}) ([]record.Record, []record.Record) {
	var findings []record.Record
	acctID := strID(ac["id"])
	mgmt := strID(ac["management_account"])
	org := strID(ac["org_id"])
	ouPath := strID(ac["ou_path"])

	lz, ok := r.lzByMgmt(mgmt)
	if !ok {
		if org != "" {
			if _, dup := lzUnknownOrgs[org]; !dup {
				lzUnknownOrgs[org] = struct{}{}
				findings = append(findings, record.Finding("warning", "lz-unknown", org,
					fmt.Sprintf("no LZ for management account %s", mgmt), cmdMap))
			}
		} else {
			findings = append(findings, record.Finding("warning", "lz-unknown", org,
				fmt.Sprintf("no LZ for management account %s", mgmt), cmdMap))
		}
		rec := r.allocationRecord(ac, lzUnalloc, unallocPlace, "unalloc", "unalloc", false)
		return []record.Record{rec}, findings
	}

	view := r.views[lz.Code]
	chain := ouChain(ac)
	entry, hasEntry := view.accounts[acctID]
	if hasEntry && entry.ouFound && strID(ac["parent_id"]) != entry.ouID {
		findings = append(findings, record.Finding("warning", "account-ou-mismatch", acctID,
			fmt.Sprintf("account %s is under %s, the taxonomy lists it under %s", acctID, ouPath, entry.ouPath), cmdMap))
	}

	var acctApps []appPick
	var acctEnv string
	var acctEnvRef taxonomy.EnvRef
	var acctHasEnv bool
	if hasEntry {
		acctApps = r.apps(entry.acc.RefURNApps)
		acctEnv, acctEnvRef, acctHasEnv = r.env(entry.acc.RefURNEnv)
	}

	// App: the account's apps › the App of the account's environment › the deepest OU › the LZ
	// default › the global default › unalloc.
	var apps []appPick
	var appRule string
	ouEff, ouRule, ouHas := deepestOU(chain, view.effects, true)
	switch {
	case len(acctApps) > 0:
		apps, appRule = acctApps, "account"
	case acctHasEnv:
		app := acctEnvRef.App
		apps = []appPick{{code: app.Code, ref: &taxonomy.AppRef{App: app, Project: acctEnvRef.Project}}}
		appRule = "account"
	case ouHas:
		apps, appRule = ouEff.apps, ouRule
	case lz.Default != nil && lz.Default.App != "":
		apps, appRule = []appPick{r.defaultApp(lz.Default.App)}, "lz-default"
	case r.t.Defaults != nil && r.t.Defaults.App != "":
		apps, appRule = []appPick{r.defaultApp(r.t.Defaults.App)}, "default"
	default:
		findings = append(findings, record.Finding("warning", "account-unmapped", acctID,
			fmt.Sprintf("account %s has no app mapping", acctID), cmdMap))
		apps, appRule = []appPick{{code: appUnalloc}}, "unalloc"
	}

	env, envRule := r.resolveEnvironment(chain, view.effects, acctEnv, acctHasEnv, lz, apps)
	if env == envUnspec && envRule == "unspec" {
		findings = append(findings, record.Finding("warning", "environment-unresolved", acctID,
			fmt.Sprintf("no environment mapping for account %s", acctID), cmdMap))
	}

	shared := len(apps) > 1
	var allocs []record.Record
	for _, pick := range apps {
		p := r.placeOf(pick)
		envOut, envRuleOut := env, envRule
		if pick.code != appUnalloc && !envUnderApp(pick, env) {
			findings = append(findings, record.Finding("warning", "environment-not-in-app", acctID+"/"+pick.code,
				fmt.Sprintf("environment %q not under app %q", env, pick.code), cmdMap))
			envOut, envRuleOut = envUnspec, "unspec"
		}
		p.env = envOut
		allocs = append(allocs, r.allocationRecord(ac, lz.Code, p, appRule, envRuleOut, shared))
	}
	return allocs, findings
}

// resolveEnvironment picks the Environment code: the account's environment › the deepest OU's ›
// the LZ default › the global default › unspec or unalloc. Nothing is guessed from names.
func (r *resolver) resolveEnvironment(chain []string, effects map[string]ouEffect, acctEnv string, acctHasEnv bool, lz *taxonomy.LZ, apps []appPick) (string, string) {
	if acctHasEnv {
		return acctEnv, "account"
	}
	if eff, rule, ok := deepestOU(chain, effects, false); ok {
		return eff.env, rule
	}
	if lz.Default != nil && lz.Default.Environment != "" {
		return lz.Default.Environment, "lz-default"
	}
	if r.t.Defaults != nil && r.t.Defaults.Environment != "" {
		return r.t.Defaults.Environment, "default"
	}
	if len(apps) > 0 && apps[0].code != appUnalloc {
		return envUnspec, "unspec"
	}
	return envUnalloc, "unalloc"
}

func envUnderApp(pick appPick, env string) bool {
	if env == "" || env == envUnspec || env == envUnalloc {
		return true
	}
	if pick.ref == nil {
		return false
	}
	return pick.ref.App.HasEnv(env)
}

// placeOf derives the project, business unit and cost center of an App from the references in the
// taxonomy (D-061).
func (r *resolver) placeOf(pick appPick) place {
	if pick.ref == nil {
		p := unallocPlace
		p.app = pick.code
		return p
	}
	p := place{project: pick.ref.Project.Code, app: pick.ref.App.Code, bu: buUnalloc, cc: ccUnalloc, ccBU: buUnalloc}
	if bu, ok := r.ix.LookupBU(pick.ref.Project.RefURNBU); ok {
		p.bu = bu.Code
	}
	if cc, ok := r.ix.LookupCC(pick.ref.App.RefURNCCT); ok {
		p.cc, p.ccBU = cc.CC.Code, cc.BU.Code
	}
	return p
}

func (r *resolver) allocationRecord(ac record.Record, lz string, p place, appRule, envRule string, shared bool) record.Record {
	acctID, ouPath := strID(ac["id"]), strID(ac["ou_path"])
	rec := record.Record{
		"kind":             "allocation",
		"tenant":           r.t.Tenant.Code,
		"account_id":       acctID,
		"account_name":     strID(ac["name"]),
		"lz":               lz,
		"ou_path":          ouPath,
		"project":          p.project,
		"app":              p.app,
		"environment":      p.env,
		"business_unit":    p.bu,
		"cost_center":      p.cc,
		"app_rule":         appRule,
		"environment_rule": envRule,
		"shared":           shared,
	}
	ouURN, acctURN := r.cloudURNs(lz, ac)
	r.setURNs(rec, lz, ouURN, acctURN, p)
	return rec
}

// discoveredOU finds the OU an account sits in, from the inventory. An account whose OU is not in
// the inventory gets an OU made from its path.
func (in inv) discoveredOU(ac record.Record) taxonomy.DiscoveredOU {
	if info, ok := in.ous[strID(ac["parent_id"])]; ok {
		return taxonomy.DiscoveredOU{ID: info.id, Path: info.path, Name: info.name}
	}
	path := strID(ac["ou_path"])
	if id, ok := in.byPath[path]; ok {
		info := in.ous[id]
		return taxonomy.DiscoveredOU{ID: info.id, Path: info.path, Name: info.name}
	}
	return taxonomy.DiscoveredOU{Path: path, Name: taxonomy.PathName(path)}
}

// cloudURNs builds the OU and account URNs of a discovered account under the LZ (D-059). Under an
// LZ the taxonomy does not list, the codes come from the discovered names.
func (r *resolver) cloudURNs(lz string, ac record.Record) (ouURN, acctURN string) {
	cloud := r.t.Cloud(lz)
	ou := r.in.discoveredOU(ac)
	acct := taxonomy.DiscoveredAccount{ID: strID(ac["id"]), Name: strID(ac["name"])}
	return cloud.OUURN(r.t.LZURN(lz), ou).String(), cloud.AccountURN(r.t.LZURN(lz), ou, acct).String()
}

// setURNs sets the eight URN fields of an allocation on rec (D-046, D-048, D-059). The cost
// center's URN sits under the cost center's own business unit, which can differ from the project's.
func (r *resolver) setURNs(rec record.Record, lz, ouURN, acctURN string, p place) {
	t := r.t
	rec["business_unit_urn"] = t.BUURN(p.bu).String()
	rec["cost_center_urn"] = t.CCURN(p.ccBU, p.cc).String()
	rec["project_urn"] = t.ProjectURN(p.project).String()
	rec["app_urn"] = t.AppURN(p.project, p.app).String()
	rec["environment_urn"] = t.EnvURN(p.project, p.app, p.env).String()
	rec["lz_urn"] = t.LZURN(lz).String()
	rec["ou_urn"] = ouURN
	rec["account_urn"] = acctURN
}

// urnFields are the URN fields a cost row copies from its allocation.
var urnFields = []string{"business_unit_urn", "cost_center_urn", "project_urn", "app_urn", "environment_urn", "lz_urn", "ou_urn", "account_urn"}

// setServiceURN sets service_urn from the cost row's account_urn and service.
func setServiceURN(cost record.Record) {
	cost["service_urn"] = urn.URN(strID(cost["account_urn"])).Child("svc", urn.Slug(strID(cost["service"]))).String()
}

func (r *resolver) enrichCost(cost record.Record, byAcct map[string][]record.Record) (record.Record, []record.Record) {
	var findings []record.Record
	out := copyRecord(cost)
	out["tenant"] = r.t.Tenant.Code
	acctID := strID(cost["account_id"])
	mgmt := strID(cost["management_account"])

	allocs, inInv := byAcct[acctID]
	if !inInv {
		lzCode := lzUnalloc
		if lz, ok := r.lzByMgmt(mgmt); ok {
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
		lzURN := r.t.LZURN(lzCode)
		r.setURNs(out, lzCode, taxonomy.UnknownOUURN(lzURN).String(), r.t.Cloud(lzCode).UnknownAccountURN(lzURN, acctID).String(), unallocPlace)
		setServiceURN(out)
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
		for _, f := range urnFields {
			out[f] = a[f]
		}
		setServiceURN(out)
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
	for _, f := range []string{"lz_urn", "ou_urn", "account_urn"} {
		out[f] = allocs[0][f]
	}
	for _, f := range []string{"project_urn", "app_urn", "environment_urn", "business_unit_urn", "cost_center_urn"} {
		out[f] = "" // share fills these from the App each row lands on
	}
	setServiceURN(out)
	return out, findings
}

func copyRecord(r record.Record) record.Record {
	out := record.Record{}
	for k, v := range r {
		out[k] = v
	}
	return out
}

func checkProjectTags(rec record.Record, byAcct map[string][]record.Record) []record.Record {
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
	projs := make([]string, 0, len(allocs))
	seen := map[string]struct{}{}
	for _, a := range allocs {
		p := strID(a["project"])
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		projs = append(projs, p)
	}
	sort.Strings(projs)
	var msg string
	switch len(projs) {
	case 0:
		msg = fmt.Sprintf("project tag %q does not match account projects for account %s", tag, acctID)
	case 1:
		msg = fmt.Sprintf("project tag %q does not match account project %q", tag, projs[0])
	default:
		msg = fmt.Sprintf("project tag %q does not match account projects %s", tag, strings.Join(projs, ", "))
	}
	return []record.Record{record.Finding("warning", "project-tag-mismatch", acctID+"/"+tag, msg, cmdMap)}
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

func strID(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return fmt.Sprintf("%.0f", x)
	default:
		return fmt.Sprint(v)
	}
}
