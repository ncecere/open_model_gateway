package providers

import "strings"

func supportsModality(modalities []string, target string) bool {
	for _, m := range modalities {
		if strings.EqualFold(m, target) {
			return true
		}
	}
	return false
}

func supportsEmbedding(modalities []string) bool {
	for _, m := range modalities {
		if strings.EqualFold(m, "embedding") || strings.EqualFold(m, "embeddings") {
			return true
		}
	}
	return false
}

func cloneMetadata(src map[string]string) map[string]string {
	if len(src) == 0 {
		return make(map[string]string)
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
