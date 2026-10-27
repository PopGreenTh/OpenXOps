package ce

import (
	"strings"
)

// ParseTagGroup splits Cost Explorer tag groups like "App$APP-CHK" or "App$".
func ParseTagGroup(key string, groupKey string) string {
	prefix := key + "$"
	if !strings.HasPrefix(groupKey, prefix) {
		return strings.TrimSpace(groupKey)
	}
	return strings.TrimSpace(strings.TrimPrefix(groupKey, prefix))
}
