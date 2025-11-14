package vertex

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type vertexPart struct {
	Text string `json:"text,omitempty"`
}

type vertexContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []vertexPart `json:"parts"`
}

func (c vertexContent) Text() string {
	var builder strings.Builder
	for i, part := range c.Parts {
		if i > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(part.Text)
	}
	return builder.String()
}

type vertexGenerationConfig struct {
	MaxOutputTokens *int32   `json:"maxOutputTokens,omitempty"`
	Temperature     *float32 `json:"temperature,omitempty"`
	TopP            *float32 `json:"topP,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

type vertexGenerateRequest struct {
	Contents          []vertexContent         `json:"contents"`
	SystemInstruction *vertexContent          `json:"systemInstruction,omitempty"`
	GenerationConfig  *vertexGenerationConfig `json:"generationConfig,omitempty"`
}

type vertexUsageMetadata struct {
	PromptTokens     int32 `json:"promptTokenCount,omitempty"`
	CandidatesTokens int32 `json:"candidatesTokenCount,omitempty"`
	TotalTokens      int32 `json:"totalTokenCount,omitempty"`
}

type vertexCandidate struct {
	Content      vertexContent        `json:"content"`
	FinishReason string               `json:"finishReason"`
	Usage        *vertexUsageMetadata `json:"usageMetadata,omitempty"`
}

type vertexGenerateResponse struct {
	Candidates    []vertexCandidate    `json:"candidates"`
	UsageMetadata *vertexUsageMetadata `json:"usageMetadata,omitempty"`
}

func (r vertexGenerateResponse) Usage() *vertexUsageMetadata {
	if r.UsageMetadata != nil {
		return r.UsageMetadata
	}
	if len(r.Candidates) > 0 {
		return r.Candidates[0].Usage
	}
	return nil
}

func (r vertexGenerateResponse) FirstCandidate() *vertexCandidate {
	if len(r.Candidates) == 0 {
		return nil
	}
	return &r.Candidates[0]
}

type vertexPredictInstance struct {
	Content string `json:"content"`
}

type vertexPredictRequest struct {
	Instances []vertexPredictInstance `json:"instances"`
}

type vertexPrediction struct {
	Values []float64 `json:"values"`
}

type vertexPredictResponse struct {
	Predictions []vertexPrediction   `json:"predictions"`
	Metadata    *vertexUsageMetadata `json:"metadata,omitempty"`
}

type vertexAPIError struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

func decodeAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	var apiErr vertexAPIError
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error.Message != "" {
		return fmt.Errorf("vertex api error %d (%s): %s", apiErr.Error.Code, apiErr.Error.Status, apiErr.Error.Message)
	}
	return fmt.Errorf("vertex api error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
}
