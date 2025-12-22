package models

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"strconv"
	"strings"
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

// ParseImageSize splits a WxH size string into pixel dimensions.
func ParseImageSize(size string) (int64, int64, bool) {
	normalized := strings.ToLower(strings.TrimSpace(size))
	if normalized == "" {
		return 0, 0, false
	}
	parts := strings.Split(normalized, "x")
	if len(parts) != 2 {
		return 0, 0, false
	}
	width, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil || width <= 0 {
		return 0, 0, false
	}
	height, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil || height <= 0 {
		return 0, 0, false
	}
	return width, height, true
}

// ImagePixelCount computes total pixels for a WxH size string and image count.
func ImagePixelCount(size string, count int) int64 {
	if count <= 0 {
		count = 1
	}
	width, height, ok := ParseImageSize(size)
	if !ok {
		return 0
	}
	return width * height * int64(count)
}

// ImageSizeFromInput returns the WxH string for a decoded image payload.
func ImageSizeFromInput(input ImageInput) (string, bool) {
	width, height, ok := ImageDimensionsFromInput(input)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("%dx%d", width, height), true
}

// ImagePixelCountFromInput computes total pixels for the image payload.
func ImagePixelCountFromInput(input ImageInput, count int) int64 {
	if count <= 0 {
		count = 1
	}
	width, height, ok := ImageDimensionsFromInput(input)
	if !ok {
		return 0
	}
	return width * height * int64(count)
}

// ImageDimensionsFromInput decodes width/height from the image payload.
func ImageDimensionsFromInput(input ImageInput) (int64, int64, bool) {
	if len(input.Data) == 0 {
		return 0, 0, false
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(input.Data))
	if err != nil {
		return 0, 0, false
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return 0, 0, false
	}
	return int64(cfg.Width), int64(cfg.Height), true
}
