package ce

import "testing"

func TestParseTagGroup(t *testing.T) {
	tests := []struct {
		key, group, want string
	}{
		{"App", "App$APP-CHK", "APP-CHK"},
		{"App", "App$", ""},
		{"Project", "Project$PRJ-SHOP", "PRJ-SHOP"},
	}
	for _, tc := range tests {
		if got := ParseTagGroup(tc.key, tc.group); got != tc.want {
			t.Errorf("ParseTagGroup(%q, %q) = %q, want %q", tc.key, tc.group, got, tc.want)
		}
	}
}
