package catalog

import "strings"

// NormalizeModelType trims, lowercases, and maps deprecated values onto the supported model types.
func NormalizeModelType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
