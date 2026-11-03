package enrich

import (
	"fmt"
	"sort"
	"strings"

	"github.com/PopGreenTh/OpenXOps/record"
	"github.com/PopGreenTh/OpenXOps/record/taxonomy"
	"github.com/PopGreenTh/OpenXOps/record/urn"
)

const cmdValidate = "finops enrich validate"

// validator collects the findings of one run.
type validator struct {
	t        taxonomy.Taxonomy
	ix       taxonomy.Index
	findings []record.Record
	aliases  map[string]map[string]string // node type -> alias -> first owner
}

func (v *validator) err(code, subject, msg string) {
	v.findings = append(v.findings, record.Finding("error", code, subject, msg, cmdValidate))
}

func (v *validator) warn(code, subject, msg string) {
	v.findings = append(v.findings, record.Finding("warning", code, subject, msg, cmdValidate))
}

// checkCode applies the code rule and the reserved ids to one code (D-047, D-056).
func (v *validator) checkCode(code, what string) {
	switch {
	case !urn.ValidCode(code):
		v.err("taxonomy-invalid-id", code, fmt.Sprintf("%s code breaks the code rule (3 to 50 characters: letters, digits and . + = @ _ -)", what))
	case reservedCode(code):
		v.err("taxonomy-reserved-code", code, fmt.Sprintf("%s code is a reserved id", what))
	}
}

// checkURN compares a `urn` written in the file with the one built from the codes (D-058).
func (v *validator) checkURN(written string, built urn.URN) {
	if written != "" && written != built.String() {
		v.err("taxonomy-urn-mismatch", written, fmt.Sprintf("urn does not match the one built from the codes, %s", built))
	}
}

// checkAlias reports an alias already used by another node of the same type in the tenant (D-057).
func (v *validator) checkAlias(typ, owner, alias string) {
	if alias == "" {
		return
	}
	if v.aliases[typ] == nil {
		v.aliases[typ] = map[string]string{}
	}
	key := strings.ToLower(strings.TrimSpace(alias))
	if first, dup := v.aliases[typ][key]; dup {
		v.err("taxonomy-duplicate-alias", typ+" "+alias, fmt.Sprintf("duplicate %s alias %q on %s (first on %s)", typ, alias, owner, first))
		return
	}
	v.aliases[typ][key] = owner
}

// dup returns a check for the codes of one parent: codes that differ only in case collide (D-056),
// because App tags already match codes in any case. It reports the collision itself.
func (v *validator) dup(parent urn.URN) func(code string) {
	seen := map[string]struct{}{}
	return func(code string) {
		k := strings.ToLower(code)
		if _, dup := seen[k]; dup {
			v.err("taxonomy-duplicate-code", parent.String()+" "+code, fmt.Sprintf("two codes collide under %s: %q", parent, code))
		}
		seen[k] = struct{}{}
	}
}

func (v *validator) missing(subject, what string) {
	v.err("taxonomy-missing-field", subject, what)
}

// ValidateTaxonomy checks a taxonomy and returns finding records.
func ValidateTaxonomy(t taxonomy.Taxonomy) []record.Record {
	v := &validator{t: t, ix: t.Index(), aliases: map[string]map[string]string{}}
	v.checkURN(t.Tenant.URN, urn.Tenant(t.Tenant.Code))
	v.checkAlias("tnt", t.Tenant.Code, t.Tenant.Alias)
	v.businessUnits()
	v.projects()
	v.lzs()
	v.defaults(t.Defaults, "defaults")
	v.unused()

	sort.Slice(v.findings, func(i, j int) bool {
		a, b := v.findings[i], v.findings[j]
		if a["code"] != b["code"] {
			return strID(a["code"]) < strID(b["code"])
		}
		return strID(a["subject"]) < strID(b["subject"])
	})
	return v.findings
}

func (v *validator) businessUnits() {
	t := v.t
	dupBU := v.dup(urn.Tenant(t.Tenant.Code))
	for _, bu := range t.BusinessUnits {
		if bu.Code == "" {
			v.missing("business_units", "business unit missing code")
			continue
		}
		if bu.Name == "" {
			v.missing(bu.Code, "business unit missing name")
		}
		v.checkCode(bu.Code, "business unit")
		dupBU(bu.Code)
		v.checkURN(bu.URN, t.BUURN(bu.Code))
		v.checkAlias("bzu", bu.Code, bu.Alias)

		dupCC := v.dup(t.BUURN(bu.Code))
		for _, cc := range bu.CostCenters {
			if cc.Code == "" {
				v.missing(bu.Code, "cost center missing code")
				continue
			}
			if cc.Name == "" {
				v.missing(cc.Code, "cost center missing name")
			}
			v.checkCode(cc.Code, "cost center")
			dupCC(cc.Code)
			v.checkURN(cc.URN, t.CCURN(bu.Code, cc.Code))
			v.checkAlias("cct", cc.Code, cc.Alias)

			dupONT := v.dup(t.CCURN(bu.Code, cc.Code))
			for _, ont := range cc.OwnerTeams {
				if ont.Code == "" {
					v.missing(cc.Code, "owner team missing code")
					continue
				}
				v.checkCode(ont.Code, "owner team")
				dupONT(ont.Code)
				v.checkAlias("ont", ont.Code, ont.Alias)
			}
		}
	}
}

func (v *validator) projects() {
	t := v.t
	tnt := urn.Tenant(t.Tenant.Code)
	dupPrj, dupApp := v.dup(tnt), v.dup(tnt)
	for _, pr := range t.Projects {
		if pr.Code == "" {
			v.missing("projects", "project missing code")
			continue
		}
		if pr.Name == "" {
			v.missing(pr.Code, "project missing name")
		}
		v.checkCode(pr.Code, "project")
		dupPrj(pr.Code)
		v.checkURN(pr.URN, t.ProjectURN(pr.Code))
		v.checkAlias("prj", pr.Code, pr.Alias)
		if pr.RefURNBU == "" {
			v.missing(pr.Code, "project missing ref_urn_bu")
		} else if _, ok := v.ix.LookupBU(pr.RefURNBU); !ok {
			v.err("taxonomy-unknown-reference", pr.RefURNBU, fmt.Sprintf("ref_urn_bu of project %s names no business unit", pr.Code))
		}

		for _, app := range pr.Apps {
			if app.Code == "" {
				v.missing(pr.Code, "app missing code")
				continue
			}
			if app.Name == "" {
				v.missing(app.Code, "app missing name")
			}
			v.checkCode(app.Code, "app")
			dupApp(app.Code)
			v.checkURN(app.URN, t.AppURN(pr.Code, app.Code))
			v.checkAlias("app", app.Code, app.Alias)
			if app.RefURNCCT == "" {
				v.missing(app.Code, "app missing ref_urn_cct")
			} else if _, ok := v.ix.LookupCC(app.RefURNCCT); !ok {
				v.err("taxonomy-unknown-reference", app.RefURNCCT, fmt.Sprintf("ref_urn_cct of app %s names no cost center", app.Code))
			}

			dupEnv := v.dup(t.AppURN(pr.Code, app.Code))
			for _, env := range app.Environments {
				if env.Code == "" {
					v.missing(app.Code, "environment missing code")
					continue
				}
				v.checkCode(env.Code, "environment")
				dupEnv(env.Code)
				v.checkURN(env.URN, t.EnvURN(pr.Code, app.Code, env.Code))
				v.checkAlias("env", env.Code, env.Alias)
			}
		}
	}
}

func (v *validator) lzs() {
	t := v.t
	if len(t.LZs) != 1 {
		v.err("taxonomy-lz-count", "lzs", fmt.Sprintf("a tenant has exactly one LZ, the taxonomy has %d", len(t.LZs)))
	}
	dupLZ := v.dup(urn.Tenant(t.Tenant.Code))
	for _, lz := range t.LZs {
		if lz.Code == "" {
			v.missing("lzs", "LZ missing code")
			continue
		}
		if lz.Name == "" || lz.CSP == "" || lz.Currency == "" || lz.Management() == "" {
			v.missing(lz.Code, "LZ missing required field")
		}
		v.checkCode(lz.Code, "LZ")
		dupLZ(lz.Code)
		v.checkURN(lz.URN, t.LZURN(lz.Code))
		v.checkAlias("lzc", lz.Code, lz.Alias)
		if lz.RefID != "" && lz.ManagementAccount != "" && lz.RefID != lz.ManagementAccount {
			v.err("taxonomy-lz-ref-id", t.LZURN(lz.Code).String(),
				fmt.Sprintf("ref_id %q differs from management_account %q", lz.RefID, lz.ManagementAccount))
		}
		v.defaults(lz.Default, lz.Code+" default")

		dupOU := v.dup(t.LZURN(lz.Code))
		seenPath, seenID := map[string]struct{}{}, map[string]struct{}{}
		for _, ou := range lz.OUPaths {
			if ou.Code == "" {
				v.missing(lz.Code, "OU entry missing code")
				continue
			}
			if ou.Path == "" && ou.RefID == "" {
				v.missing(ou.Code, "OU entry needs a path or a ref_id")
			}
			v.checkCode(ou.Code, "OU")
			dupOU(ou.Code)
			v.checkURN(ou.URN, t.LZURN(lz.Code).Child("org", ou.Code))
			v.checkAlias("org", ou.Code, ou.Alias)
			v.duplicateOU(ou, seenPath, seenID)
			v.refs(ou.Code, ou.RefURNApps, ou.RefURNEnv)

			dupAcct := v.dup(t.LZURN(lz.Code).Child("org", ou.Code))
			for _, a := range ou.Accounts {
				if a.Code == "" {
					v.missing(ou.Code, "account entry missing code")
					continue
				}
				if a.RefID == "" {
					v.missing(a.Code, "account entry missing ref_id (the account id)")
				}
				v.checkCode(a.Code, "account")
				dupAcct(a.Code)
				v.checkURN(a.URN, t.LZURN(lz.Code).Child("org", ou.Code).Child("acc", a.Code))
				v.checkAlias("acc", a.Code, a.Alias)
				v.refs(a.Code, a.RefURNApps, a.RefURNEnv)
				v.envInApps(a)
			}
		}
	}
}

// duplicateOU reports two entries of one LZ that name the same OU.
func (v *validator) duplicateOU(ou taxonomy.OUPath, seenPath, seenID map[string]struct{}) {
	if ou.Path != "" {
		if _, dup := seenPath[ou.Path]; dup {
			v.err("taxonomy-duplicate-ou-link", ou.Path, fmt.Sprintf("two OU entries name the path %s", ou.Path))
		}
		seenPath[ou.Path] = struct{}{}
	}
	if ou.RefID != "" {
		if _, dup := seenID[ou.RefID]; dup {
			v.err("taxonomy-duplicate-ou-link", ou.RefID, fmt.Sprintf("two OU entries name the OU %s", ou.RefID))
		}
		seenID[ou.RefID] = struct{}{}
	}
}

// refs checks the App and Environment references of an OU or account entry.
func (v *validator) refs(owner string, apps []string, env string) {
	for _, u := range apps {
		if _, ok := v.ix.LookupApp(u); !ok {
			v.err("taxonomy-unknown-reference", u, fmt.Sprintf("ref_urn_apps of %s names no app", owner))
		}
	}
	if env != "" {
		if _, ok := v.ix.LookupEnv(env); !ok {
			v.err("taxonomy-unknown-reference", env, fmt.Sprintf("ref_urn_env of %s names no environment", owner))
		}
	}
}

// envInApps reports an account whose Environment belongs to an App other than the ones it lists.
func (v *validator) envInApps(a taxonomy.Account) {
	if len(a.RefURNApps) == 0 || a.RefURNEnv == "" {
		return
	}
	env, ok := v.ix.LookupEnv(a.RefURNEnv)
	if !ok {
		return
	}
	for _, u := range a.RefURNApps {
		if ref, ok := v.ix.LookupApp(u); ok && ref.App == env.App {
			return
		}
	}
	v.err("taxonomy-environment-not-in-app", a.Code, fmt.Sprintf("environment %s is not under any app the account lists", a.RefURNEnv))
}

// defaults checks a `default` (an App and an Environment, as codes).
func (v *validator) defaults(d *taxonomy.Defaults, subject string) {
	if d == nil || d.App == "" {
		return
	}
	ref, ok := v.ix.AppByCode(d.App)
	if !ok {
		v.err("taxonomy-unknown-reference", subject, fmt.Sprintf("unknown app %q", d.App))
		return
	}
	if d.Environment != "" && !ref.App.HasEnv(d.Environment) {
		v.err("taxonomy-environment-not-in-app", subject, fmt.Sprintf("environment %q not defined under app %q", d.Environment, d.App))
	}
}

// unused warns about a business unit, cost center or project no App leads to.
func (v *validator) unused() {
	t := v.t
	if v.noApps() {
		return
	}
	usedBU, usedCC, usedPrj := map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}
	for _, pr := range t.Projects {
		if len(pr.Apps) == 0 {
			continue
		}
		usedPrj[pr.Code] = struct{}{}
		if bu, ok := v.ix.LookupBU(pr.RefURNBU); ok {
			usedBU[bu.Code] = struct{}{}
		}
		for _, app := range pr.Apps {
			if cc, ok := v.ix.LookupCC(app.RefURNCCT); ok {
				usedCC[t.CCURN(cc.BU.Code, cc.CC.Code).String()] = struct{}{}
			}
		}
	}
	for _, bu := range t.BusinessUnits {
		if _, ok := usedBU[bu.Code]; !ok {
			v.warn("taxonomy-unused", bu.Code, "unused business unit")
		}
		for _, cc := range bu.CostCenters {
			if _, ok := usedCC[t.CCURN(bu.Code, cc.Code).String()]; !ok {
				v.warn("taxonomy-unused", cc.Code, "unused cost center")
			}
		}
	}
	for _, pr := range t.Projects {
		if _, ok := usedPrj[pr.Code]; !ok {
			v.warn("taxonomy-unused", pr.Code, "unused project")
		}
	}
}

func (v *validator) noApps() bool {
	for _, pr := range v.t.Projects {
		if len(pr.Apps) > 0 {
			return false
		}
	}
	return true
}

// reservedCode reports whether code is one of the reserved system ids, in any case (D-056).
func reservedCode(code string) bool {
	switch strings.ToLower(code) {
	case "unalloc", "unspec", "invalid":
		return true
	}
	return false
}

// HasErrorSeverity reports whether any finding is an error.
func HasErrorSeverity(findings []record.Record) bool {
	for _, f := range findings {
		if f.Kind() == "finding" && f["severity"] == "error" {
			return true
		}
	}
	return false
}
