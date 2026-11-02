package enrich

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Taxonomy is the landing-zone and allocation configuration file.
type Taxonomy struct {
	// Tenant is the tenant id the file belongs to; it must equal --tenant (D-045).
	Tenant        string         `yaml:"tenant"`
	Alias         string         `yaml:"alias,omitempty"`
	BusinessUnits []BusinessUnit `yaml:"business_units"`
	CostCenters   []CostCenter   `yaml:"cost_centers"`
	OwnerTeams    []OwnerTeam    `yaml:"owner_teams,omitempty"`
	Projects      []Project      `yaml:"projects"`
	Apps          []App          `yaml:"apps"`
	LZs           []LZ           `yaml:"lzs"`
	OUs           []OU           `yaml:"ous,omitempty"`
	OULinks       []OULink       `yaml:"ou_links"`
	Accounts      []AccountLink  `yaml:"accounts"`
	Defaults      *Defaults      `yaml:"defaults"`
	Tags          *TagKeys       `yaml:"tags"`
}

type BusinessUnit struct {
	Code  string `yaml:"code"`
	Name  string `yaml:"name"`
	Alias string `yaml:"alias,omitempty"`
}

type CostCenter struct {
	Code         string `yaml:"code"`
	Name         string `yaml:"name"`
	BusinessUnit string `yaml:"business_unit"`
	Alias        string `yaml:"alias,omitempty"`
}

type OwnerTeam struct {
	Code       string `yaml:"code"`
	Name       string `yaml:"name"`
	CostCenter string `yaml:"cost_center,omitempty"`
	Alias      string `yaml:"alias,omitempty"`
}

type Project struct {
	Code         string `yaml:"code"`
	Name         string `yaml:"name"`
	BusinessUnit string `yaml:"business_unit"`
	Alias        string `yaml:"alias,omitempty"`
}

type Environment struct {
	Code  string `yaml:"code"`
	Name  string `yaml:"name"`
	Alias string `yaml:"alias,omitempty"`
}

type App struct {
	Code         string        `yaml:"code"`
	Name         string        `yaml:"name"`
	Project      string        `yaml:"project"`
	CostCenter   string        `yaml:"cost_center"`
	Environments []Environment `yaml:"environments"`
	Alias        string        `yaml:"alias,omitempty"`
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
	Alias             string     `yaml:"alias,omitempty"`
}

type OU struct {
	Code  string `yaml:"code"`
	Name  string `yaml:"name"`
	Path  string `yaml:"path"`
	Alias string `yaml:"alias,omitempty"`
}

type OULink struct {
	LZ          string `yaml:"lz"`
	OU          string `yaml:"ou"`
	App         string `yaml:"app"`
	Environment string `yaml:"environment"`
	Alias       string `yaml:"alias,omitempty"`
}

type AccountLink struct {
	Account     string   `yaml:"account"`
	Apps        []string `yaml:"apps"`
	Environment string   `yaml:"environment"`
	Alias       string   `yaml:"alias,omitempty"`
}

type Defaults struct {
	App         string `yaml:"app"`
	Environment string `yaml:"environment"`
}

type TagKeys struct {
	App     string `yaml:"app"`
	Project string `yaml:"project"`
}

// LoadTaxonomy reads and parses a taxonomy YAML file; unknown keys at any depth are rejected for flat schemas.
func LoadTaxonomy(path string) (Taxonomy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Taxonomy{}, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var tax Taxonomy
	if err := dec.Decode(&tax); err == nil {
		return tax, nil
	}

	nestedTax, nestedErr := loadNestedTaxonomy(data)
	if nestedErr == nil {
		return nestedTax, nil
	}

	return Taxonomy{}, fmt.Errorf("parse taxonomy %s: %w", path, err)
}

type nestedDoc struct {
	Tenant        yaml.Node         `yaml:"tenant"`
	Alias         string            `yaml:"alias"`
	BusinessUnits []nestedBU        `yaml:"business_units"`
	CostCenters   []nestedCC        `yaml:"cost_centers"`
	OwnerTeams    []nestedOwnerTeam `yaml:"owner_teams"`
	Projects      []nestedProject   `yaml:"projects"`
	Apps          []nestedApp       `yaml:"apps"`
	LZs           []nestedLZ        `yaml:"lzs"`
	LZConns       []nestedLZ        `yaml:"lz_conns"`
	OULinks       []OULink          `yaml:"ou_links"`
	Accounts      []AccountLink     `yaml:"accounts"`
	Defaults      *Defaults         `yaml:"defaults"`
	Tags          *TagKeys          `yaml:"tags"`
}

type nestedOwnerTeam struct {
	Code  string `yaml:"code"`
	URN   string `yaml:"urn"`
	Name  string `yaml:"name"`
	Alias string `yaml:"alias"`
}

type nestedBU struct {
	Code        string     `yaml:"code"`
	URN         string     `yaml:"urn"`
	Name        string     `yaml:"name"`
	Alias       string     `yaml:"alias"`
	CostCenters []nestedCC `yaml:"cost_centers"`
}

type nestedCC struct {
	Code         string            `yaml:"code"`
	URN          string            `yaml:"urn"`
	Name         string            `yaml:"name"`
	Alias        string            `yaml:"alias"`
	BusinessUnit string            `yaml:"business_unit"`
	OwnerTeams   []nestedOwnerTeam `yaml:"owner_teams"`
}

type nestedProject struct {
	Code         string      `yaml:"code"`
	URN          string      `yaml:"urn"`
	Name         string      `yaml:"name"`
	Alias        string      `yaml:"alias"`
	BusinessUnit string      `yaml:"business_unit"`
	RefURNBU     string      `yaml:"ref_urn_bu"`
	Apps         []nestedApp `yaml:"apps"`
}

type nestedApp struct {
	Code         string              `yaml:"code"`
	URN          string              `yaml:"urn"`
	Name         string              `yaml:"name"`
	Alias        string              `yaml:"alias"`
	Project      string              `yaml:"project"`
	CostCenter   string              `yaml:"cost_center"`
	RefURNCCT    string              `yaml:"ref_urn_cct"`
	Environments []nestedEnvironment `yaml:"environments"`
}

type nestedEnvironment struct {
	Code      string `yaml:"code"`
	URN       string `yaml:"urn"`
	Name      string `yaml:"name"`
	Alias     string `yaml:"alias"`
	RefURNOnt string `yaml:"ref_urn_ont"`
}

type nestedLZ struct {
	Code              string         `yaml:"code"`
	URN               string         `yaml:"urn"`
	Name              string         `yaml:"name"`
	Alias             string         `yaml:"alias"`
	CSP               string         `yaml:"csp"`
	Currency          string         `yaml:"currency"`
	ManagementAccount string         `yaml:"management_account"`
	RefID             string         `yaml:"ref_id"`
	Default           *LZDefault     `yaml:"default"`
	RefURNPrjs        []string       `yaml:"ref_urn_prjs"`
	OUPaths           []nestedOUPath `yaml:"ou_paths"`
}

type nestedOUPath struct {
	Code        string          `yaml:"code"`
	URN         string          `yaml:"urn"`
	Name        string          `yaml:"name"`
	Alias       string          `yaml:"alias"`
	RefID       string          `yaml:"ref_id"`
	Path        string          `yaml:"path"`
	RefURNApps  []string        `yaml:"ref_urn_apps"`
	RefURNEnv   string          `yaml:"ref_urn_env"`
	Environment string          `yaml:"environment"`
	Accounts    []nestedAccount `yaml:"accounts"`
}

type nestedAccount struct {
	Code        string `yaml:"code"`
	URN         string `yaml:"urn"`
	Name        string `yaml:"name"`
	Alias       string `yaml:"alias"`
	RefID       string `yaml:"ref_id"`
	RefURNEnv   string `yaml:"ref_urn_env"`
	Environment string `yaml:"environment"`
	Contact     string `yaml:"contact"`
}

func loadNestedTaxonomy(data []byte) (Taxonomy, error) {
	var doc nestedDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return Taxonomy{}, err
	}
	isNested := doc.Tenant.Kind == yaml.MappingNode || len(doc.LZConns) > 0
	for _, bu := range doc.BusinessUnits {
		if len(bu.CostCenters) > 0 {
			isNested = true
			break
		}
	}
	for _, prj := range doc.Projects {
		if len(prj.Apps) > 0 {
			isNested = true
			break
		}
	}
	for _, lz := range doc.LZs {
		if len(lz.OUPaths) > 0 {
			isNested = true
			break
		}
	}
	if !isNested {
		return Taxonomy{}, fmt.Errorf("not a nested taxonomy")
	}

	tenantStr, err := parseTenantNode(&doc.Tenant)
	if err != nil {
		return Taxonomy{}, err
	}

	var tax Taxonomy
	tax.Tenant = tenantStr
	if doc.Tenant.Kind == yaml.MappingNode {
		var tm struct {
			Alias string `yaml:"alias"`
		}
		_ = doc.Tenant.Decode(&tm)
		if tm.Alias != "" {
			tax.Alias = tm.Alias
		}
	}
	if doc.Alias != "" {
		tax.Alias = doc.Alias
	}
	tax.Defaults = doc.Defaults
	tax.Tags = doc.Tags
	if tax.Tags == nil {
		tax.Tags = &TagKeys{App: "App", Project: "Project"}
	}

	for _, bu := range doc.BusinessUnits {
		buCode := urnSegment(bu.URN, "bzu")
		if buCode == "" {
			buCode = strings.ToLower(bu.Code)
		}
		tax.BusinessUnits = append(tax.BusinessUnits, BusinessUnit{
			Code:  buCode,
			Name:  bu.Name,
			Alias: bu.Alias,
		})
		for _, cc := range bu.CostCenters {
			ccCode := urnSegment(cc.URN, "cct")
			if ccCode == "" {
				ccCode = strings.ToLower(cc.Code)
			}
			tax.CostCenters = append(tax.CostCenters, CostCenter{
				Code:         ccCode,
				Name:         cc.Name,
				BusinessUnit: buCode,
				Alias:        cc.Alias,
			})
			for _, ont := range cc.OwnerTeams {
				ontCode := urnSegment(ont.URN, "ont")
				if ontCode == "" {
					ontCode = strings.ToLower(ont.Code)
				}
				tax.OwnerTeams = append(tax.OwnerTeams, OwnerTeam{
					Code:       ontCode,
					Name:       ont.Name,
					CostCenter: ccCode,
					Alias:      ont.Alias,
				})
			}
		}
	}
	for _, cc := range doc.CostCenters {
		ccCode := urnSegment(cc.URN, "cct")
		if ccCode == "" {
			ccCode = strings.ToLower(cc.Code)
		}
		tax.CostCenters = append(tax.CostCenters, CostCenter{
			Code:         ccCode,
			Name:         cc.Name,
			BusinessUnit: strings.ToLower(cc.BusinessUnit),
			Alias:        cc.Alias,
		})
		for _, ont := range cc.OwnerTeams {
			ontCode := urnSegment(ont.URN, "ont")
			if ontCode == "" {
				ontCode = strings.ToLower(ont.Code)
			}
			tax.OwnerTeams = append(tax.OwnerTeams, OwnerTeam{
				Code:       ontCode,
				Name:       ont.Name,
				CostCenter: ccCode,
				Alias:      ont.Alias,
			})
		}
	}
	for _, ont := range doc.OwnerTeams {
		ontCode := urnSegment(ont.URN, "ont")
		if ontCode == "" {
			ontCode = strings.ToLower(ont.Code)
		}
		tax.OwnerTeams = append(tax.OwnerTeams, OwnerTeam{
			Code:  ontCode,
			Name:  ont.Name,
			Alias: ont.Alias,
		})
	}

	for _, prj := range doc.Projects {
		prjCode := urnSegment(prj.URN, "prj")
		if prjCode == "" {
			prjCode = strings.ToLower(prj.Code)
		}
		buRef := urnSegment(prj.RefURNBU, "bzu")
		if buRef == "" {
			buRef = strings.ToLower(prj.BusinessUnit)
		}
		tax.Projects = append(tax.Projects, Project{
			Code:         prjCode,
			Name:         prj.Name,
			BusinessUnit: buRef,
			Alias:        prj.Alias,
		})
		for _, app := range prj.Apps {
			appCode := urnSegment(app.URN, "app")
			if appCode == "" {
				appCode = strings.ToLower(app.Code)
			}
			ccRef := urnSegment(app.RefURNCCT, "cct")
			if ccRef == "" {
				ccRef = strings.ToLower(app.CostCenter)
			}
			var envs []Environment
			for _, env := range app.Environments {
				envCode := urnSegment(env.URN, "env")
				if envCode == "" {
					envCode = strings.ToLower(env.Code)
				}
				envs = append(envs, Environment{
					Code:  envCode,
					Name:  env.Name,
					Alias: env.Alias,
				})
			}
			tax.Apps = append(tax.Apps, App{
				Code:         appCode,
				Name:         app.Name,
				Project:      prjCode,
				CostCenter:   ccRef,
				Environments: envs,
				Alias:        app.Alias,
			})
		}
	}
	for _, app := range doc.Apps {
		appCode := urnSegment(app.URN, "app")
		if appCode == "" {
			appCode = strings.ToLower(app.Code)
		}
		var envs []Environment
		for _, env := range app.Environments {
			envCode := urnSegment(env.URN, "env")
			if envCode == "" {
				envCode = strings.ToLower(env.Code)
			}
			envs = append(envs, Environment{
				Code:  envCode,
				Name:  env.Name,
				Alias: env.Alias,
			})
		}
		tax.Apps = append(tax.Apps, App{
			Code:         appCode,
			Name:         app.Name,
			Project:      strings.ToLower(app.Project),
			CostCenter:   strings.ToLower(app.CostCenter),
			Environments: envs,
			Alias:        app.Alias,
		})
	}

	rawLZs := doc.LZs
	if len(rawLZs) == 0 {
		rawLZs = doc.LZConns
	}
	for _, lz := range rawLZs {
		lzCode := urnSegment(lz.URN, "lzc")
		if lzCode == "" {
			lzCode = strings.ToLower(lz.Code)
		}
		mgmt := lz.ManagementAccount
		if mgmt == "" {
			mgmt = lz.RefID
		}
		tax.LZs = append(tax.LZs, LZ{
			Code:              lzCode,
			Name:              lz.Name,
			CSP:               strings.ToLower(lz.CSP),
			Currency:          lz.Currency,
			ManagementAccount: mgmt,
			Default:           lz.Default,
			Alias:             lz.Alias,
		})

		for _, ou := range lz.OUPaths {
			ouPath := ou.Path
			if ouPath == "" {
				ouPath = ou.RefID
			}
			if ouPath == "" {
				ouPath = ou.Code
			}

			tax.OUs = append(tax.OUs, OU{
				Code:  ou.Code,
				Name:  ou.Name,
				Path:  ouPath,
				Alias: ou.Alias,
			})

			ouApp := ""
			if len(ou.RefURNApps) > 0 {
				ouApp = urnSegment(ou.RefURNApps[0], "app")
				if ouApp == "" {
					ouApp = strings.ToLower(ou.RefURNApps[0])
				}
			}

			ouEnv := ""
			if ou.RefURNEnv != "" {
				ouEnv = urnSegment(ou.RefURNEnv, "env")
			} else if ou.Environment != "" {
				ouEnv = strings.ToLower(ou.Environment)
			} else if len(ou.Accounts) > 0 {
				var firstEnv string
				allSame := true
				for _, a := range ou.Accounts {
					e := urnSegment(a.RefURNEnv, "env")
					if e == "" {
						e = strings.ToLower(a.Environment)
					}
					if e == "" {
						allSame = false
						break
					}
					if firstEnv == "" {
						firstEnv = e
					} else if firstEnv != e {
						allSame = false
						break
					}
				}
				if allSame && firstEnv != "" {
					ouIdentifier := strings.ToLower(ou.Code + " " + ou.Name + " " + ou.Path)
					for _, a := range tax.Apps {
						for _, envDef := range a.Environments {
							if envDef.Code == firstEnv {
								if strings.Contains(ouIdentifier, strings.ToLower(envDef.Name)) ||
									strings.Contains(ouIdentifier, strings.ToLower(envDef.Code)) {
									ouEnv = firstEnv
									break
								}
							}
						}
						if ouEnv != "" {
							break
						}
					}
				}
			}

			if ouApp != "" || ouEnv != "" {
				tax.OULinks = append(tax.OULinks, OULink{
					LZ:          lzCode,
					OU:          ouPath,
					App:         ouApp,
					Environment: ouEnv,
					Alias:       ou.Alias,
				})
			}

			for _, a := range ou.Accounts {
				acctID := a.RefID
				if acctID == "" {
					acctID = a.Code
				}
				aApp := urnSegment(a.RefURNEnv, "app")
				if aApp == "" && len(ou.RefURNApps) > 0 {
					aApp = urnSegment(ou.RefURNApps[0], "app")
				}
				aEnv := urnSegment(a.RefURNEnv, "env")
				if aEnv == "" {
					aEnv = strings.ToLower(a.Environment)
				}
				var acctApps []string
				if aApp != "" {
					acctApps = []string{aApp}
				}
				tax.Accounts = append(tax.Accounts, AccountLink{
					Account:     acctID,
					Apps:        acctApps,
					Environment: aEnv,
					Alias:       a.Alias,
				})
			}
		}
	}

	if len(doc.OULinks) > 0 {
		tax.OULinks = append(tax.OULinks, doc.OULinks...)
	}
	if len(doc.Accounts) > 0 {
		tax.Accounts = append(tax.Accounts, doc.Accounts...)
	}

	return tax, nil
}

func parseTenantNode(n *yaml.Node) (string, error) {
	if n == nil || n.Kind == 0 {
		return "", nil
	}
	if n.Kind == yaml.ScalarNode {
		return n.Value, nil
	}
	if n.Kind == yaml.MappingNode {
		var m struct {
			Code string `yaml:"code"`
			URN  string `yaml:"urn"`
		}
		if err := n.Decode(&m); err != nil {
			return "", err
		}
		if m.URN != "" {
			if t := urnSegment(m.URN, "tnt"); t != "" {
				return t, nil
			}
		}
		if m.Code != "" {
			return strings.ToLower(m.Code), nil
		}
	}
	return "", fmt.Errorf("unexpected tenant node kind")
}

func urnSegment(u, typ string) string {
	parts := strings.Split(u, ":")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == typ {
			return parts[i+1]
		}
	}
	return ""
}

// reservedCode reports whether code is one of the reserved system ids (D-047).
func reservedCode(code string) bool {
	return code == "unalloc" || code == "unspec" || code == "invalid"
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
