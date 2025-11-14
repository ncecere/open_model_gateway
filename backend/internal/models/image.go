package models

import (
	"bytes"
	"errors"
	"io"
	"time"
)

// ErrImageOperationUnsupported indicates that the provider does not support a
// requested image workflow (edits, variations, etc.).
var ErrImageOperationUnsupported = errors.New("image operation unsupported")

// ImageInput stores a binary image payload in-memory so the same data can be
// re-read if multiple providers are attempted (e.g., failover routing).
type ImageInput struct {
	Data        []byte
	Filename    string
	ContentType string
}

// Reader returns a fresh ReadCloser for the stored image bytes.
func (in ImageInput) Reader() io.ReadCloser {
	return io.NopCloser(bytes.NewReader(in.Data))
}

// Size exposes the number of bytes in the image payload.
func (in ImageInput) Size() int64 {
	return int64(len(in.Data))
}

// ImageRequest captures parameters for generating images via provider adapters.
type ImageRequest struct {
	Model          string
	Prompt         string
	Size           string
	ResponseFormat string
	Quality        string
	N              int
	User           string
	Background     string
	Style          string
}

// ImageEditRequest captures the multipart-driven inputs for the
// `/v1/images/edits` endpoint.
type ImageEditRequest struct {
	Model          string
	Prompt         string
	Images         []ImageInput
	Mask           *ImageInput
	Size           string
	ResponseFormat string
	Quality        string
	Background     string
	Style          string
	N              int
	User           string
}

// ImageVariationRequest captures the payload for `/v1/images/variations`.
type ImageVariationRequest struct {
	Model          string
	Image          ImageInput
	Size           string
	ResponseFormat string
	Quality        string
	Background     string
	Style          string
	N              int
	User           string
}

// ImageData represents a single generated image payload.
type ImageData struct {
	B64JSON       string
	URL           string
	RevisedPrompt string
}

// ImageResponse wraps generated images along with creation metadata.
type ImageResponse struct {
	Created time.Time
	Data    []ImageData
	Usage   Usage
}
