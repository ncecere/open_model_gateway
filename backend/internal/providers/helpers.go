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

func setDefaultAudioMetadata(md map[string]string, enableStreaming bool, includeDiarized bool) {
	if md == nil {
		return
	}
	if _, ok := md["audio_formats"]; !ok {
		formats := []string{"json", "text", "srt", "vtt", "verbose_json"}
		if includeDiarized {
			formats = append(formats, "diarized_json")
		}
		md["audio_formats"] = strings.Join(formats, ",")
	}
	if _, ok := md["audio_timestamp_granularities"]; !ok {
		md["audio_timestamp_granularities"] = "word,segment"
	}
	if _, ok := md["audio_streaming"]; !ok {
		if enableStreaming {
			md["audio_streaming"] = "true"
		} else {
			md["audio_streaming"] = "false"
		}
	}
}
