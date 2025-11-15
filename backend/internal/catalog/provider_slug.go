package catalog

import "strings"

var providerAliases = map[string]string{
	"openai_compatible": "openai-compatible",
}

// NormalizeProviderSlug canonicalizes provider identifiers so backend + UI share the same names.
func NormalizeProviderSlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	if slug == "" {
		return ""
	}
	if canonical, ok := providerAliases[slug]; ok {
		return canonical
	}
	return slug
}
