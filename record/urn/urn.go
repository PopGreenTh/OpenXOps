// Package urn holds the id rule, the slug rule, the OU path rule and a URN builder (D-046, D-047).
package urn

import (
	"regexp"
	"strings"
)

var idRule = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ValidID reports whether id follows the id rule: `^[a-z0-9]+(-[a-z0-9]+)*$`.
func ValidID(id string) bool {
	return idRule.MatchString(id)
}

// Slug lowercases s and turns every run of characters outside [a-z0-9] into one `-`, trimmed at
// both ends. An empty result is `unknown`.
func Slug(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			dash = false
			b.WriteRune(r)
			continue
		}
		dash = true
	}
	if b.Len() == 0 {
		return "unknown"
	}
	return b.String()
}

// OUPath turns an OU path into its segments, each slugged, joined by `/`, without the leading `/`
// (`/Workloads/Checkout/Prod` → `workloads/checkout/prod`). The root `/` is `root`.
func OUPath(path string) string {
	var segs []string
	for _, s := range strings.Split(path, "/") {
		if s != "" {
			segs = append(segs, Slug(s))
		}
	}
	if len(segs) == 0 {
		return "root"
	}
	return strings.Join(segs, "/")
}

// URN is a node name: `urn:tnt:<tenant>` followed by `:<type>:<id>` for each level.
type URN string

// Tenant returns the URN of a tenant, `urn:tnt:<tenant>`.
func Tenant(tenant string) URN {
	return URN("urn:tnt:" + tenant)
}

// Child returns u followed by `:<typ>:<id>`.
func (u URN) Child(typ, id string) URN {
	return u + URN(":"+typ+":"+id)
}

// String returns the URN as a string.
func (u URN) String() string {
	return string(u)
}
