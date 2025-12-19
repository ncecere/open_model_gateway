package providers

import (
	"strconv"
	"strings"
)

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

func deriveCapabilities(modalities []string, md map[string]string) RouteCapabilities {
	caps := RouteCapabilities{
		ImageInput: supportsModality(modalities, "image"),
		AudioInput: supportsModality(modalities, "audio"),
		VideoInput: supportsModality(modalities, "video"),
	}
	caps.ImageInput = capabilityOverride(md, "cap_image_input", caps.ImageInput)
	caps.AudioInput = capabilityOverride(md, "cap_audio_input", caps.AudioInput)
	caps.VideoInput = capabilityOverride(md, "cap_video_input", caps.VideoInput)
	return caps
}

func capabilityOverride(md map[string]string, key string, base bool) bool {
	if md == nil {
		return base
	}
	value, ok := md[key]
	if !ok {
		return base
	}
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return base
	}
	return parsed
}
