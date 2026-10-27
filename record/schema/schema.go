// Package schema holds required-field and type checks for record kinds (D-037).
package schema

import (
	"fmt"
)

// FieldType is the JSON scalar type a field must have.
type FieldType int

const (
	TypeString FieldType = iota
	TypeNumber
	TypeBool
	TypeNumberOrNull
)

// Field is one schema entry.
type Field struct {
	Name     string
	Type     FieldType
	Required bool
}

// Schema is the check table for one record kind.
type Schema struct {
	Kind   string
	Fields []Field
}

// FieldError names the file, line and field that failed validation.
type FieldError struct {
	File  string
	Line  int
	Field string
	Err   error
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("%s:%d: field %q: %v", e.File, e.Line, e.Field, e.Err)
}

func (e *FieldError) Unwrap() error { return e.Err }

// ByKind maps record kind to its schema.
var ByKind = map[string]Schema{
	"ou":               ouSchema,
	"account":          accountSchema,
	"cost":             costSchema,
	"project_tag_cost": projectTagCostSchema,
	"allocation":       allocationSchema,
	"change":           changeSchema,
	"finding":          findingSchema,
}

var ouSchema = Schema{
	Kind: "ou",
	Fields: []Field{
		{Name: "kind", Type: TypeString, Required: true},
		{Name: "csp", Type: TypeString, Required: true},
		{Name: "org_id", Type: TypeString, Required: true},
		{Name: "management_account", Type: TypeString, Required: true},
		{Name: "id", Type: TypeString, Required: true},
		{Name: "name", Type: TypeString, Required: true},
		{Name: "parent_id", Type: TypeString, Required: true},
		{Name: "path", Type: TypeString, Required: true},
		{Name: "id_path", Type: TypeString, Required: true},
		{Name: "has_children", Type: TypeBool, Required: true},
	},
}

var accountSchema = Schema{
	Kind: "account",
	Fields: []Field{
		{Name: "kind", Type: TypeString, Required: true},
		{Name: "csp", Type: TypeString, Required: true},
		{Name: "org_id", Type: TypeString, Required: true},
		{Name: "management_account", Type: TypeString, Required: true},
		{Name: "id", Type: TypeString, Required: true},
		{Name: "name", Type: TypeString, Required: true},
		{Name: "email", Type: TypeString, Required: true},
		{Name: "status", Type: TypeString, Required: true},
		{Name: "parent_id", Type: TypeString, Required: true},
		{Name: "ou_path", Type: TypeString, Required: true},
		{Name: "ou_id_path", Type: TypeString, Required: true},
	},
}

var costSchema = Schema{
	Kind: "cost",
	Fields: []Field{
		{Name: "kind", Type: TypeString, Required: true},
		{Name: "csp", Type: TypeString, Required: true},
		{Name: "management_account", Type: TypeString, Required: true},
		{Name: "granularity", Type: TypeString, Required: true},
		{Name: "period", Type: TypeString, Required: true},
		{Name: "metric", Type: TypeString, Required: true},
		{Name: "amount", Type: TypeNumber, Required: true},
		{Name: "currency", Type: TypeString, Required: true},
		{Name: "estimated", Type: TypeBool, Required: true},
		{Name: "account_id", Type: TypeString, Required: true},
		{Name: "service", Type: TypeString, Required: true},
		{Name: "app_tag", Type: TypeString, Required: true},
	},
}

var projectTagCostSchema = Schema{
	Kind: "project_tag_cost",
	Fields: []Field{
		{Name: "kind", Type: TypeString, Required: true},
		{Name: "csp", Type: TypeString, Required: true},
		{Name: "management_account", Type: TypeString, Required: true},
		{Name: "granularity", Type: TypeString, Required: true},
		{Name: "period", Type: TypeString, Required: true},
		{Name: "metric", Type: TypeString, Required: true},
		{Name: "amount", Type: TypeNumber, Required: true},
		{Name: "currency", Type: TypeString, Required: true},
		{Name: "estimated", Type: TypeBool, Required: true},
		{Name: "account_id", Type: TypeString, Required: true},
		{Name: "project_tag", Type: TypeString, Required: true},
	},
}

var allocationSchema = Schema{
	Kind: "allocation",
	Fields: []Field{
		{Name: "kind", Type: TypeString, Required: true},
		{Name: "account_id", Type: TypeString, Required: true},
		{Name: "account_name", Type: TypeString, Required: true},
		{Name: "lz", Type: TypeString, Required: true},
		{Name: "ou_path", Type: TypeString, Required: true},
		{Name: "project", Type: TypeString, Required: true},
		{Name: "app", Type: TypeString, Required: true},
		{Name: "environment", Type: TypeString, Required: true},
		{Name: "business_unit", Type: TypeString, Required: true},
		{Name: "cost_center", Type: TypeString, Required: true},
		{Name: "app_rule", Type: TypeString, Required: true},
		{Name: "environment_rule", Type: TypeString, Required: true},
		{Name: "shared", Type: TypeBool, Required: true},
	},
}

var changeSchema = Schema{
	Kind: "change",
	Fields: []Field{
		{Name: "kind", Type: TypeString, Required: true},
		{Name: "subject_type", Type: TypeString, Required: true},
		{Name: "subject", Type: TypeString, Required: true},
		{Name: "change", Type: TypeString, Required: true},
		{Name: "field", Type: TypeString, Required: true},
		{Name: "old", Type: TypeString, Required: true},
		{Name: "new", Type: TypeString, Required: true},
		{Name: "cost_moved", Type: TypeNumberOrNull, Required: true},
		{Name: "currency", Type: TypeString, Required: true},
		{Name: "period_from", Type: TypeString, Required: true},
		{Name: "period_to", Type: TypeString, Required: true},
	},
}

var findingSchema = Schema{
	Kind: "finding",
	Fields: []Field{
		{Name: "kind", Type: TypeString, Required: true},
		{Name: "severity", Type: TypeString, Required: true},
		{Name: "code", Type: TypeString, Required: true},
		{Name: "subject", Type: TypeString, Required: true},
		{Name: "message", Type: TypeString, Required: true},
		{Name: "command", Type: TypeString, Required: true},
	},
}

// Kinds returns every kind that has a schema (for tests).
func Kinds() []string {
	out := make([]string, 0, len(ByKind))
	for k := range ByKind {
		out = append(out, k)
	}
	return out
}

// Validate checks rec when kind is in checkKinds. Unknown kinds and unchecked kinds pass.
func Validate(file string, line int, rec map[string]any, checkKinds map[string]struct{}) error {
	if checkKinds == nil {
		return nil
	}
	kind, _ := rec["kind"].(string)
	if kind == "" {
		return nil
	}
	if _, ok := checkKinds[kind]; !ok {
		return nil
	}
	sch, ok := ByKind[kind]
	if !ok {
		return nil
	}
	for _, f := range sch.Fields {
		v, present := rec[f.Name]
		if !present {
			if f.Required {
				return &FieldError{File: file, Line: line, Field: f.Name, Err: fmt.Errorf("missing required field")}
			}
			continue
		}
		if err := checkType(v, f.Type); err != nil {
			return &FieldError{File: file, Line: line, Field: f.Name, Err: err}
		}
	}
	return nil
}

func checkType(v any, typ FieldType) error {
	switch typ {
	case TypeString:
		if _, ok := v.(string); !ok {
			return fmt.Errorf("want string")
		}
	case TypeNumber:
		if !isNumber(v) {
			return fmt.Errorf("want number")
		}
	case TypeBool:
		if _, ok := v.(bool); !ok {
			return fmt.Errorf("want bool")
		}
	case TypeNumberOrNull:
		if v == nil {
			return nil
		}
		if !isNumber(v) {
			return fmt.Errorf("want number or null")
		}
	}
	return nil
}

func isNumber(v any) bool {
	_, ok := v.(float64)
	return ok
}
