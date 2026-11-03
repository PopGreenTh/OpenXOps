package taxonomy

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// rawDoc is the file as written. `lzs` and `lz_conns` are two names for the same list.
type rawDoc struct {
	Tenant        Node           `yaml:"tenant"`
	BusinessUnits []BusinessUnit `yaml:"business_units"`
	Projects      []Project      `yaml:"projects"`
	LZs           []LZ           `yaml:"lzs"`
	LZConns       []LZ           `yaml:"lz_conns"`
	Defaults      *Defaults      `yaml:"defaults"`
	Tags          *TagKeys       `yaml:"tags"`
}

// flatKeys are the top-level keys of the flat layout, which is no longer read (D-060).
var flatKeys = []string{"cost_centers", "apps", "ou_links", "accounts", "owner_teams"}

// Load reads the taxonomy file at path. Unknown keys at any depth are errors, and so is a file in
// the flat layout.
func Load(path string) (Taxonomy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Taxonomy{}, fmt.Errorf("read taxonomy %s: %w", path, err)
	}
	t, err := Parse(data)
	if err != nil {
		return Taxonomy{}, fmt.Errorf("parse taxonomy %s: %w", path, err)
	}
	return t, nil
}

// Parse reads a taxonomy from YAML. Codes are kept as written (D-056).
func Parse(data []byte) (Taxonomy, error) {
	var top map[string]yaml.Node
	if err := yaml.Unmarshal(data, &top); err != nil {
		return Taxonomy{}, err
	}
	if len(top) == 0 {
		return Taxonomy{}, errors.New("the file is empty")
	}
	for _, k := range flatKeys {
		if _, ok := top[k]; ok {
			return Taxonomy{}, fmt.Errorf("key %q is the flat layout, which is no longer read; nest it in the tenant's hierarchies (docs/user/taxonomy.md)", k)
		}
	}
	if n, ok := top["tenant"]; ok && n.Kind == yaml.ScalarNode {
		return Taxonomy{}, errors.New("tenant must be a mapping with a code and a name, for example tenant: { code: acme, name: Acme }")
	}
	_, hasLZs := top["lzs"]
	_, hasConns := top["lz_conns"]
	if hasLZs && hasConns {
		return Taxonomy{}, errors.New("both lzs and lz_conns are set; use one of them")
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var d rawDoc
	if err := dec.Decode(&d); err != nil && !errors.Is(err, io.EOF) {
		return Taxonomy{}, err
	}
	lzs := d.LZs
	if hasConns {
		lzs = d.LZConns
	}
	return Taxonomy{
		Tenant:        d.Tenant,
		BusinessUnits: d.BusinessUnits,
		Projects:      d.Projects,
		LZs:           lzs,
		Defaults:      d.Defaults,
		Tags:          d.Tags,
	}, nil
}

// UnmarshalYAML lets a taxonomy sit inside another YAML document (the reference cases do): the
// node is read with the same strict rules as a file.
func (t *Taxonomy) UnmarshalYAML(value *yaml.Node) error {
	data, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	parsed, err := Parse(data)
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}
