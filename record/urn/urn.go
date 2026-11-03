// Package urn holds the code rule, the URN rule, the slug rule, the OU path rule and a URN builder
// (D-046, D-053, D-056).
package urn

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	codeRule = regexp.MustCompile(`^[A-Za-z0-9.+=@_-]{3,50}$`)
	urnRule  = regexp.MustCompile(`^[a-z0-9.:+=@_/-]{11,250}$`)
)

// ValidCode reports whether code follows the code rule (D-056): 3 to 50 characters from letters,
// digits and `.` `+` `=` `@` `_` `-`. It binds --tenant, the taxonomy's tenant and every taxonomy
// code.
func ValidCode(code string) bool {
	return codeRule.MatchString(code)
}

// ValidURN reports whether u follows the URN rule (D-056): 11 to 250 characters from lowercase
// letters, digits and `.` `:` `+` `=` `@` `_` `/` `-`.
func ValidURN(u string) bool {
	return urnRule.MatchString(u)
}

// Slug turns a name from the cloud into an id (D-053). It first splits CamelCase: a word break
// goes between a lowercase letter or digit and the uppercase letter after it, and between two
// uppercase letters when the second is followed by a lowercase letter. It then lowercases, turns
// every run of characters outside [a-z0-9] into one `-`, and trims at both ends. An empty result
// is `unknown`.
func Slug(s string) string {
	rs := []rune(s)
	isUpper := func(r rune) bool { return r >= 'A' && r <= 'Z' }
	isLower := func(r rune) bool { return r >= 'a' && r <= 'z' }
	isDigit := func(r rune) bool { return r >= '0' && r <= '9' }
	var b strings.Builder
	dash := false
	for i, r := range rs {
		if i > 0 && isUpper(r) {
			prev := rs[i-1]
			if isLower(prev) || isDigit(prev) || (isUpper(prev) && i+1 < len(rs) && isLower(rs[i+1])) {
				dash = true
			}
		}
		r = unicode.ToLower(r)
		if isLower(r) || isDigit(r) {
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

// Tenant returns the URN of a tenant, `urn:tnt:<tenant>`. The tenant is lowercased: a URN id is
// the code in lowercase (D-056).
func Tenant(tenant string) URN {
	return URN("urn:tnt:" + strings.ToLower(tenant))
}

// Child returns u followed by `:<typ>:<id>`. The id is lowercased: a URN id is the code in
// lowercase (`PRJ_Shop` gives `prj_shop`), and a slug already is.
func (u URN) Child(typ, id string) URN {
	return u + URN(":"+typ+":"+strings.ToLower(id))
}

// String returns the URN as a string.
func (u URN) String() string {
	return string(u)
}
