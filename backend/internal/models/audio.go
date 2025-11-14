package models

import "io"

// AudioInput wraps the uploaded audio payload.
type AudioInput struct {
	Reader      io.Reader
	Filename    string
	ContentType string
	Bytes       int64
}

type AudioTranscriptionTask string

const (
	AudioTranscriptionTaskTranscribe AudioTranscriptionTask = "transcribe"
	AudioTranscriptionTaskTranslate  AudioTranscriptionTask = "translate"
)

// AudioTranscriptionRequest captures transcription/translation parameters.
type AudioTranscriptionRequest struct {
	Model       string
	Task        AudioTranscriptionTask
	Input       AudioInput
	Prompt      string
	Temperature *float32
	Language    string
}

// AudioTranscriptionResponse is a normalized transcription payload.
type AudioTranscriptionResponse struct {
	Text  string
	Usage Usage
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
