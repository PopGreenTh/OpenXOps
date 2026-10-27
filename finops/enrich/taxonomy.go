package enrich

import (
	"fmt"
	"os"
	"strings"

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

var topLevelKeys = map[string]struct{}{
	"business_units": {}, "cost_centers": {}, "projects": {}, "apps": {}, "lzs": {},
	"ou_links": {}, "accounts": {}, "defaults": {}, "tags": {},
}

// LoadTaxonomy reads and parses a taxonomy YAML file with strict top-level keys.
func LoadTaxonomy(path string) (Taxonomy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Taxonomy{}, err
	}
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return Taxonomy{}, fmt.Errorf("parse taxonomy %s: %w", path, err)
	}
	if err := checkDocKeys(&root, topLevelKeys, "taxonomy"); err != nil {
		return Taxonomy{}, err
	}
	var tax Taxonomy
	if err := yaml.Unmarshal(data, &tax); err != nil {
		return Taxonomy{}, fmt.Errorf("parse taxonomy %s: %w", path, err)
	}
	return tax, nil
}

func checkDocKeys(root *yaml.Node, allowed map[string]struct{}, where string) error {
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return nil
	}
	m := root.Content[0]
	if m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(m.Content); i += 2 {
		key := m.Content[i].Value
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("%s: unknown key %q", where, key)
		}
	}
	return nil
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
