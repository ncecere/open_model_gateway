package models

import (
	"io"
	"strings"
)

// AudioInput wraps the uploaded audio payload.
type AudioInput struct {
	Reader      io.Reader
	Filename    string
	ContentType string
	Bytes       int64
	FormField   string
}

type AudioTranscriptionTask string

const (
	AudioTranscriptionTaskTranscribe AudioTranscriptionTask = "transcribe"
	AudioTranscriptionTaskTranslate  AudioTranscriptionTask = "translate"
)

type AudioResponseFormat string

const (
	AudioResponseFormatJSON        AudioResponseFormat = "json"
	AudioResponseFormatText        AudioResponseFormat = "text"
	AudioResponseFormatSRT         AudioResponseFormat = "srt"
	AudioResponseFormatVTT         AudioResponseFormat = "vtt"
	AudioResponseFormatVerboseJSON AudioResponseFormat = "verbose_json"
	AudioResponseFormatDiarized    AudioResponseFormat = "diarized_json"
)

type AudioTimestampGranularity string

const (
	AudioTimestampGranularityWord    AudioTimestampGranularity = "word"
	AudioTimestampGranularitySegment AudioTimestampGranularity = "segment"
)

// AudioTranscriptionRequest captures transcription/translation parameters.
type AudioTranscriptionRequest struct {
	Model                  string
	Task                   AudioTranscriptionTask
	Input                  AudioInput
	Prompt                 string
	Temperature            *float32
	Language               string
	ResponseFormat         AudioResponseFormat
	TimestampGranularities []AudioTimestampGranularity
	User                   string
	Stream                 bool
}

// AudioTranscriptionResponse is a normalized transcription payload.
type AudioTranscriptionResponse struct {
	Format      AudioResponseFormat
	ContentType string
	Payload     []byte
	Text        string
	Usage       Usage
}

// AudioTranscriptionStreamChunk is emitted when providers support SSE streaming responses.
type AudioTranscriptionStreamChunk struct {
	Payload []byte
	Usage   *Usage
	Done    bool
	Err     error
}

// AudioSpeechRequest drives text-to-speech generation.
type AudioSpeechRequest struct {
	Model        string
	Input        string
	Voice        string
	Format       string
	Stream       bool
	StreamFormat string
}

// AudioSpeechResponse returns generated audio bytes (non-streaming).
type AudioSpeechResponse struct {
	Audio []byte
	Usage Usage
}

// AudioSpeechChunk represents a streaming speech fragment.
type AudioSpeechChunk struct {
	Audio []byte
	Done  bool
}

// ParseAudioResponseFormat normalizes a string into a known response format.
// Returns false when the format is not recognized.
func ParseAudioResponseFormat(val string) (AudioResponseFormat, bool) {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "":
		return AudioResponseFormatJSON, true
	case "json":
		return AudioResponseFormatJSON, true
	case "text":
		return AudioResponseFormatText, true
	case "srt":
		return AudioResponseFormatSRT, true
	case "vtt":
		return AudioResponseFormatVTT, true
	case "verbose_json":
		return AudioResponseFormatVerboseJSON, true
	case "diarized_json":
		return AudioResponseFormatDiarized, true
	default:
		return "", false
	}
}

// ContentType returns the HTTP content-type for the response format.
func (f AudioResponseFormat) ContentType() string {
	switch f {
	case AudioResponseFormatText:
		return "text/plain; charset=utf-8"
	case AudioResponseFormatSRT:
		return "application/x-subrip; charset=utf-8"
	case AudioResponseFormatVTT:
		return "text/vtt; charset=utf-8"
	case AudioResponseFormatVerboseJSON, AudioResponseFormatJSON, AudioResponseFormatDiarized:
		return "application/json"
	default:
		return "application/json"
	}
}

// IsJSONFormat indicates whether the format yields a JSON payload.
func (f AudioResponseFormat) IsJSONFormat() bool {
	switch f {
	case "", AudioResponseFormatJSON, AudioResponseFormatVerboseJSON, AudioResponseFormatDiarized:
		return true
	default:
		return false
	}
}

// ParseAudioGranularity normalizes raw form inputs into known constants.
func ParseAudioGranularity(val string) (AudioTimestampGranularity, bool) {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "word":
		return AudioTimestampGranularityWord, true
	case "segment":
		return AudioTimestampGranularitySegment, true
	default:
		return "", false
	}
}
