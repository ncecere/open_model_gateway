package vertex

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/models"
)

const maxVertexInlineDataBytes = 20 * 1024 * 1024

func (a *Adapter) buildGenerateContentRequest(ctx context.Context, req models.ChatRequest) (vertexGenerateRequest, error) {
	if len(req.Messages) == 0 {
		return vertexGenerateRequest{}, errors.New("vertex: at least one message is required")
	}

	var systemParts []vertexPart
	contents := make([]vertexContent, 0, len(req.Messages))

	for idx, msg := range req.Messages {
		role := strings.ToLower(strings.TrimSpace(msg.Role))
		allowNonText := role != "system" && role != "developer"

		parts, err := a.convertMessageParts(ctx, idx, msg, allowNonText)
		if err != nil {
			return vertexGenerateRequest{}, err
		}
		if len(parts) == 0 {
			continue
		}

		switch role {
		case "system", "developer":
			systemParts = append(systemParts, parts...)
		case "assistant":
			contents = append(contents, vertexContent{Role: "model", Parts: parts})
		default:
			contents = append(contents, vertexContent{Role: "user", Parts: parts})
		}
	}

	if len(contents) == 0 {
		return vertexGenerateRequest{}, errors.New("vertex: no user/assistant messages provided")
	}

	var systemInstruction *vertexContent
	if len(systemParts) > 0 {
		systemInstruction = &vertexContent{Role: "system", Parts: systemParts}
	}

	cfg := &vertexGenerationConfig{}
	if req.MaxTokens != nil {
		cfg.MaxOutputTokens = req.MaxTokens
	}
	if req.Temperature != nil {
		cfg.Temperature = req.Temperature
	}
	if req.TopP != nil {
		cfg.TopP = req.TopP
	}
	if len(req.Stop) > 0 {
		cfg.StopSequences = append(cfg.StopSequences, req.Stop...)
	}
	if cfg.MaxOutputTokens == nil && cfg.Temperature == nil && cfg.TopP == nil && len(cfg.StopSequences) == 0 {
		cfg = nil
	}

	return vertexGenerateRequest{
		Contents:          contents,
		SystemInstruction: systemInstruction,
		GenerationConfig:  cfg,
	}, nil
}

func (a *Adapter) convertMessageParts(ctx context.Context, msgIndex int, msg models.ChatMessage, allowNonText bool) ([]vertexPart, error) {
	parts := msg.ContentParts
	if len(parts) == 0 {
		text := strings.TrimSpace(msg.Text())
		if text == "" {
			return nil, nil
		}
		parts = []models.MessageContentPart{{Type: models.MessageContentPartTypeText, Text: text}}
	}

	converted := make([]vertexPart, 0, len(parts))
	for partIdx, part := range parts {
		if part.IsTextual() {
			text := strings.TrimSpace(part.Text)
			if text != "" {
				converted = append(converted, vertexPart{Text: text})
			}
			continue
		}
		if !allowNonText {
			partType := part.Type
			if partType == "" {
				partType = "non-text"
			}
			return nil, fmt.Errorf("vertex: message %d does not support %s content", msgIndex, partType)
		}

		switch strings.ToLower(strings.TrimSpace(part.Type)) {
		case models.MessageContentPartTypeImageURL:
			inline, err := a.inlineDataFromImageURL(ctx, part.ImageURL)
			if err != nil {
				return nil, fmt.Errorf("vertex: message %d image_url: %w", msgIndex, err)
			}
			converted = append(converted, vertexPart{InlineData: inline})
		case models.MessageContentPartTypeImageFile:
			return nil, fmt.Errorf("vertex: message %d image_file parts are not supported yet", msgIndex)
		case models.MessageContentPartTypeInputAudio:
			inline, err := inlineDataFromAudio(part.InputAudio)
			if err != nil {
				return nil, fmt.Errorf("vertex: message %d input_audio: %w", msgIndex, err)
			}
			converted = append(converted, vertexPart{InlineData: inline})
		case "input_image":
			inline, err := inlineDataFromInlineImage(part.InputImage)
			if err != nil {
				return nil, fmt.Errorf("vertex: message %d input_image: %w", msgIndex, err)
			}
			converted = append(converted, vertexPart{InlineData: inline})
		default:
			return nil, fmt.Errorf("vertex: message %d unsupported content part %q (index %d)", msgIndex, part.Type, partIdx)
		}
	}
	return converted, nil
}

func (a *Adapter) inlineDataFromImageURL(ctx context.Context, image *models.MessageContentImageURL) (*vertexInlineData, error) {
	if image == nil || strings.TrimSpace(image.URL) == "" {
		return nil, errors.New("image_url missing url")
	}
	urlValue := strings.TrimSpace(image.URL)
	if strings.HasPrefix(strings.ToLower(urlValue), "data:") {
		mime, data, err := decodeDataURL(urlValue)
		if err != nil {
			return nil, err
		}
		return inlineDataFromBytes(mime, data)
	}
	mime, data, err := a.fetchExternalAsset(ctx, urlValue)
	if err != nil {
		return nil, err
	}
	return inlineDataFromBytes(mime, data)
}

func inlineDataFromAudio(audio *models.MessageContentInputAudio) (*vertexInlineData, error) {
	if audio == nil {
		return nil, errors.New("audio payload missing")
	}
	encoded := strings.TrimSpace(audio.Data)
	if encoded == "" {
		return nil, errors.New("audio data missing")
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode audio data: %w", err)
	}
	if len(data) > maxVertexInlineDataBytes {
		return nil, fmt.Errorf("audio data exceeds %d bytes", maxVertexInlineDataBytes)
	}
	mime := normalizeMime(audio.Format, "audio")
	return &vertexInlineData{
		MimeType: mime,
		Data:     base64.StdEncoding.EncodeToString(data),
	}, nil
}

func inlineDataFromInlineImage(image *models.MessageContentImageObject) (*vertexInlineData, error) {
	if image == nil {
		return nil, errors.New("image payload missing")
	}
	encoded := strings.TrimSpace(image.Data)
	if encoded == "" {
		return nil, errors.New("image data missing")
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode image data: %w", err)
	}
	if len(data) > maxVertexInlineDataBytes {
		return nil, fmt.Errorf("image data exceeds %d bytes", maxVertexInlineDataBytes)
	}
	mime := normalizeMime(image.Format, "image")
	return inlineDataFromBytes(mime, data)
}

func inlineDataFromBytes(mime string, data []byte) (*vertexInlineData, error) {
	if len(data) > maxVertexInlineDataBytes {
		return nil, fmt.Errorf("asset exceeds %d bytes", maxVertexInlineDataBytes)
	}
	mime = strings.TrimSpace(mime)
	if mime == "" {
		mime = "application/octet-stream"
	}
	return &vertexInlineData{
		MimeType: mime,
		Data:     base64.StdEncoding.EncodeToString(data),
	}, nil
}

func normalizeMime(format, kind string) string {
	fmtVal := strings.TrimSpace(format)
	if fmtVal == "" {
		return kind + "/" + "octet-stream"
	}
	if strings.Contains(fmtVal, "/") {
		return fmtVal
	}
	return kind + "/" + fmtVal
}

func (a *Adapter) fetchExternalAsset(ctx context.Context, raw string) (string, []byte, error) {
	client := a.assetClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return "", nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", nil, fmt.Errorf("remote asset status %d", resp.StatusCode)
	}
	reader := io.LimitReader(resp.Body, maxVertexInlineDataBytes+1)
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", nil, fmt.Errorf("read remote asset: %w", err)
	}
	if len(data) > maxVertexInlineDataBytes {
		return "", nil, fmt.Errorf("remote asset exceeds %d bytes", maxVertexInlineDataBytes)
	}
	mime := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if mime == "" {
		mime = http.DetectContentType(data)
	}
	return mime, data, nil
}

func decodeDataURL(raw string) (string, []byte, error) {
	payload := strings.TrimPrefix(raw, "data:")
	parts := strings.SplitN(payload, ",", 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("invalid data url")
	}
	meta := parts[0]
	dataPart := parts[1]
	segments := strings.Split(meta, ";")
	mime := strings.TrimSpace(segments[0])
	base64Encoded := false
	for _, seg := range segments[1:] {
		if strings.EqualFold(strings.TrimSpace(seg), "base64") {
			base64Encoded = true
		}
	}
	if mime == "" {
		mime = "application/octet-stream"
	}
	var decoded []byte
	var err error
	if base64Encoded {
		decoded, err = base64.StdEncoding.DecodeString(dataPart)
	} else {
		decodedString, decodeErr := url.QueryUnescape(dataPart)
		if decodeErr != nil {
			err = decodeErr
		} else {
			decoded = []byte(decodedString)
		}
	}
	if err != nil {
		return "", nil, fmt.Errorf("decode data url: %w", err)
	}
	if len(decoded) > maxVertexInlineDataBytes {
		return "", nil, fmt.Errorf("data url exceeds %d bytes", maxVertexInlineDataBytes)
	}
	return mime, decoded, nil
}
