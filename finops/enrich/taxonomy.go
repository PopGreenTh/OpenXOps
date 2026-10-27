package enrich

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/PopGreenTh/OpenXOps/record"
	"gopkg.in/yaml.v3"
)

// Taxonomy is the landing-zone and allocation configuration file.
type Taxonomy struct {
	BusinessUnits []BusinessUnit `yaml:"business_units"`
	CostCenters   []CostCenter   `yaml:"cost_centers"`
	Projects      []Project      `yaml:"projects"`
	Apps          []App          `yaml:"apps"`
	LZs           []LZ           `yaml:"lzs"`
	OULinks       []OULink       `yaml:"ou_links"`
	Accounts      []AccountLink  `yaml:"accounts"`
	Defaults      *Defaults      `yaml:"defaults"`
	Tags          *TagKeys       `yaml:"tags"`
}

type BusinessUnit struct {
	Code string `yaml:"code"`
	Name string `yaml:"name"`
}

type CostCenter struct {
	Code         string `yaml:"code"`
	Name         string `yaml:"name"`
	BusinessUnit string `yaml:"business_unit"`
}

type Project struct {
	Code         string `yaml:"code"`
	Name         string `yaml:"name"`
	BusinessUnit string `yaml:"business_unit"`
}

type Environment struct {
	Code string `yaml:"code"`
	Name string `yaml:"name"`
}

type App struct {
	Code         string        `yaml:"code"`
	Name         string        `yaml:"name"`
	Project      string        `yaml:"project"`
	CostCenter   string        `yaml:"cost_center"`
	Environments []Environment `yaml:"environments"`
}

type LZDefault struct {
	App         string `yaml:"app"`
	Environment string `yaml:"environment"`
}

type LZ struct {
	Code              string     `yaml:"code"`
	Name              string     `yaml:"name"`
	CSP               string     `yaml:"csp"`
	Currency          string     `yaml:"currency"`
	ManagementAccount string     `yaml:"management_account"`
	Default           *LZDefault `yaml:"default"`
}

type OULink struct {
	LZ          string `yaml:"lz"`
	OU          string `yaml:"ou"`
	App         string `yaml:"app"`
	Environment string `yaml:"environment"`
}

type AccountLink struct {
	Account     string   `yaml:"account"`
	Apps        []string `yaml:"apps"`
	Environment string   `yaml:"environment"`
}

type Defaults struct {
	App         string `yaml:"app"`
	Environment string `yaml:"environment"`
}

type TagKeys struct {
	App     string `yaml:"app"`
	Project string `yaml:"project"`
}

// LoadTaxonomy reads and parses a taxonomy YAML file; unknown keys at any depth are rejected.
func LoadTaxonomy(path string) (Taxonomy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Taxonomy{}, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var tax Taxonomy
	if err := dec.Decode(&tax); err != nil {
		return Taxonomy{}, fmt.Errorf("parse taxonomy %s: %w", path, err)
	}
	return tax, nil
}

func reservedCode(code string) bool {
	return strings.HasSuffix(code, "-UNALLOC") || strings.HasSuffix(code, "-UNSPEC") || strings.HasSuffix(code, "-INVALID")
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

const cmdValidate = "finops enrich validate"

// ValidateTaxonomy checks a taxonomy and returns finding records.
func ValidateTaxonomy(t Taxonomy) []record.Record {
	var findings []record.Record
	addErr := func(code, subject, msg string) {
		findings = append(findings, record.Finding("error", code, subject, msg, cmdValidate))
	}
	addWarn := func(code, subject, msg string) {
		findings = append(findings, record.Finding("warning", code, subject, msg, cmdValidate))
	}

	buCodes := map[string]struct{}{}
	for _, bu := range t.BusinessUnits {
		if bu.Code == "" {
			addErr("taxonomy-missing-field", "business_units", "business unit missing code")
			continue
		}
		if bu.Name == "" {
			addErr("taxonomy-missing-field", bu.Code, "business unit missing name")
		}
		if reservedCode(bu.Code) {
			addErr("taxonomy-reserved-code", bu.Code, "reserved code suffix")
		}
		if _, dup := buCodes[bu.Code]; dup {
			addErr("taxonomy-duplicate-code", bu.Code, "duplicate business unit code")
		}
		buCodes[bu.Code] = struct{}{}
	}

	ccCodes := map[string]struct{}{}
	for _, cc := range t.CostCenters {
		if cc.Code == "" {
			addErr("taxonomy-missing-field", "cost_centers", "cost center missing code")
			continue
		}
		if cc.Name == "" || cc.BusinessUnit == "" {
			addErr("taxonomy-missing-field", cc.Code, "cost center missing required field")
		}
		if reservedCode(cc.Code) {
			addErr("taxonomy-reserved-code", cc.Code, "reserved code suffix")
		}
		if _, dup := ccCodes[cc.Code]; dup {
			addErr("taxonomy-duplicate-code", cc.Code, "duplicate cost center code")
		}
		ccCodes[cc.Code] = struct{}{}
		if cc.BusinessUnit != "" {
			if _, ok := buCodes[cc.BusinessUnit]; !ok {
				addErr("taxonomy-unknown-reference", cc.Code, fmt.Sprintf("unknown business_unit %q", cc.BusinessUnit))
			}
		}
	}

	prjCodes := map[string]struct{}{}
	for _, pr := range t.Projects {
		if pr.Code == "" {
			addErr("taxonomy-missing-field", "projects", "project missing code")
			continue
		}
		if pr.Name == "" || pr.BusinessUnit == "" {
			addErr("taxonomy-missing-field", pr.Code, "project missing required field")
		}
		if reservedCode(pr.Code) {
			addErr("taxonomy-reserved-code", pr.Code, "reserved code suffix")
		}
		if _, dup := prjCodes[pr.Code]; dup {
			addErr("taxonomy-duplicate-code", pr.Code, "duplicate project code")
		}
		prjCodes[pr.Code] = struct{}{}
		if pr.BusinessUnit != "" {
			if _, ok := buCodes[pr.BusinessUnit]; !ok {
				addErr("taxonomy-unknown-reference", pr.Code, fmt.Sprintf("unknown business_unit %q", pr.BusinessUnit))
			}
		}
	}

	appCodes := map[string]struct{}{}
	appEnvs := map[string]map[string]struct{}{}
	for _, app := range t.Apps {
		if app.Code == "" {
			addErr("taxonomy-missing-field", "apps", "app missing code")
			continue
		}
		if app.Name == "" || app.Project == "" || app.CostCenter == "" {
			addErr("taxonomy-missing-field", app.Code, "app missing required field")
		}
		if reservedCode(app.Code) {
			addErr("taxonomy-reserved-code", app.Code, "reserved code suffix")
		}
		if _, dup := appCodes[app.Code]; dup {
			addErr("taxonomy-duplicate-code", app.Code, "duplicate app code")
		}
		appCodes[app.Code] = struct{}{}
		if app.Project != "" {
			if _, ok := prjCodes[app.Project]; !ok {
				addErr("taxonomy-unknown-reference", app.Code, fmt.Sprintf("unknown project %q", app.Project))
			}
		}
		if app.CostCenter != "" {
			if _, ok := ccCodes[app.CostCenter]; !ok {
				addErr("taxonomy-unknown-reference", app.Code, fmt.Sprintf("unknown cost_center %q", app.CostCenter))
			}
		}
		envSet := map[string]struct{}{}
		for _, env := range app.Environments {
			if env.Code == "" {
				addErr("taxonomy-missing-field", app.Code, "environment missing code")
				continue
			}
			if reservedCode(env.Code) {
				addErr("taxonomy-reserved-code", env.Code, "reserved environment code")
			}
			envSet[env.Code] = struct{}{}
		}
		appEnvs[app.Code] = envSet
	}

	lzCodes := map[string]struct{}{}
	for _, lz := range t.LZs {
		if lz.Code == "" {
			addErr("taxonomy-missing-field", "lzs", "LZ missing code")
			continue
		}
		if lz.Name == "" || lz.CSP == "" || lz.Currency == "" || lz.ManagementAccount == "" {
			addErr("taxonomy-missing-field", lz.Code, "LZ missing required field")
		}
		if reservedCode(lz.Code) {
			addErr("taxonomy-reserved-code", lz.Code, "reserved code suffix")
		}
		if _, dup := lzCodes[lz.Code]; dup {
			addErr("taxonomy-duplicate-code", lz.Code, "duplicate LZ code")
		}
		lzCodes[lz.Code] = struct{}{}
		if lz.Default != nil {
			checkEnvUnderApps(lz.Default.Environment, []string{lz.Default.App}, appEnvs, addErr, lz.Code+" default")
		}
	}

	type ouKey struct {
		lz, ou string
	}
	ouApps := map[ouKey]string{}
	ouEnvs := map[ouKey]string{}
	for _, link := range t.OULinks {
		if link.LZ == "" || link.OU == "" {
			addErr("taxonomy-missing-field", "ou_links", "ou_link missing lz or ou")
			continue
		}
		if link.App == "" && link.Environment == "" {
			addErr("taxonomy-missing-field", link.OU, "ou_link must set app and/or environment")
		}
		if link.LZ != "" {
			if _, ok := lzCodes[link.LZ]; !ok {
				addErr("taxonomy-unknown-reference", link.OU, fmt.Sprintf("unknown lz %q", link.LZ))
			}
		}
		if link.App != "" {
			if _, ok := appCodes[link.App]; !ok {
				addErr("taxonomy-unknown-reference", link.OU, fmt.Sprintf("unknown app %q", link.App))
			}
		}
		if link.Environment != "" {
			if reservedCode(link.Environment) {
				addErr("taxonomy-reserved-code", link.Environment, "reserved environment code")
			}
		}
		k := ouKey{link.LZ, link.OU}
		if link.App != "" {
			if prev, ok := ouApps[k]; ok {
				addErr("taxonomy-duplicate-ou-link", link.OU, fmt.Sprintf("duplicate app for lz %q ou %q (was %q)", link.LZ, link.OU, prev))
			}
			ouApps[k] = link.App
		}
		if link.Environment != "" {
			if prev, ok := ouEnvs[k]; ok {
				addErr("taxonomy-duplicate-ou-link", link.OU, fmt.Sprintf("duplicate environment for lz %q ou %q (was %q)", link.LZ, link.OU, prev))
			}
			ouEnvs[k] = link.Environment
		}
	}

	for _, ac := range t.Accounts {
		if ac.Account == "" {
			addErr("taxonomy-missing-field", "accounts", "account entry missing account id")
			continue
		}
		if len(ac.Apps) == 0 && ac.Environment == "" {
			addErr("taxonomy-missing-field", ac.Account, "account entry must set apps and/or environment")
		}
		for _, app := range ac.Apps {
			if _, ok := appCodes[app]; !ok {
				addErr("taxonomy-unknown-reference", ac.Account, fmt.Sprintf("unknown app %q", app))
			}
		}
		checkEnvUnderApps(ac.Environment, ac.Apps, appEnvs, addErr, ac.Account)
	}

	if t.Defaults != nil {
		checkEnvUnderApps(t.Defaults.Environment, []string{t.Defaults.App}, appEnvs, addErr, "defaults")
		if t.Defaults.App != "" {
			if _, ok := appCodes[t.Defaults.App]; !ok {
				addErr("taxonomy-unknown-reference", "defaults", fmt.Sprintf("unknown app %q", t.Defaults.App))
			}
		}
	}

	usedBU, usedCC, usedPrj := map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}
	for _, app := range t.Apps {
		if pr, ok := prjByCode(t, app.Project); ok {
			usedPrj[pr.Code] = struct{}{}
			usedBU[pr.BusinessUnit] = struct{}{}
		}
		if cc, ok := ccByCode(t, app.CostCenter); ok {
			usedCC[cc.Code] = struct{}{}
		}
	}
	for code := range buCodes {
		if _, ok := usedBU[code]; !ok && len(t.Apps) > 0 {
			addWarn("taxonomy-unused", code, "unused business unit")
		}
	}
	for code := range ccCodes {
		if _, ok := usedCC[code]; !ok && len(t.Apps) > 0 {
			addWarn("taxonomy-unused", code, "unused cost center")
		}
	}
	for code := range prjCodes {
		if _, ok := usedPrj[code]; !ok && len(t.Apps) > 0 {
			addWarn("taxonomy-unused", code, "unused project")
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a["code"] != b["code"] {
			return strID(a["code"]) < strID(b["code"])
		}
		return strID(a["subject"]) < strID(b["subject"])
	})
	return findings
}

func checkEnvUnderApps(env string, apps []string, appEnvs map[string]map[string]struct{}, addErr func(string, string, string), subject string) {
	if env == "" {
		return
	}
	if len(apps) == 0 {
		return
	}
	for _, app := range apps {
		set, ok := appEnvs[app]
		if !ok {
			continue
		}
		if _, has := set[env]; !has {
			addErr("taxonomy-environment-not-in-app", subject, fmt.Sprintf("environment %q not defined under app %q", env, app))
		}
	}
}

func prjByCode(t Taxonomy, code string) (Project, bool) {
	for _, p := range t.Projects {
		if p.Code == code {
			return p, true
		}
	}
	return Project{}, false
}

func ccByCode(t Taxonomy, code string) (CostCenter, bool) {
	for _, c := range t.CostCenters {
		if c.Code == code {
			return c, true
		}
	}
	return CostCenter{}, false
}

func HasErrorSeverity(findings []record.Record) bool {
	for _, f := range findings {
		if f.Kind() == "finding" && f["severity"] == "error" {
			return true
		}
	}
	return false
}
