package vertex

import (
	"net/http"
	"strings"

	"github.com/ncecere/open_model_gateway/backend/internal/adapters/base"
)

type vertexPart struct {
	Text             string                  `json:"text,omitempty"`
	InlineData       *vertexInlineData       `json:"inlineData,omitempty"`
	FileData         *vertexFileData         `json:"fileData,omitempty"`
	FunctionCall     *vertexFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *vertexFunctionResponse `json:"functionResponse,omitempty"`
}

// vertexFunctionCall represents a function call from the model.
type vertexFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args,omitempty"`
}

// vertexFunctionResponse represents the result of a function call sent back to the model.
type vertexFunctionResponse struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

type vertexInlineData struct {
	MimeType string `json:"mimeType,omitempty"`
	Data     string `json:"data,omitempty"`
}

type vertexFileData struct {
	FileURI  string `json:"fileUri,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
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
	Tools             []vertexTool            `json:"tools,omitempty"`
	ToolConfig        *vertexToolConfig       `json:"toolConfig,omitempty"`
}

// vertexTool wraps function declarations for the Gemini API.
type vertexTool struct {
	FunctionDeclarations []vertexFunctionDeclaration `json:"functionDeclarations,omitempty"`
}

// vertexFunctionDeclaration defines a callable function for the model.
type vertexFunctionDeclaration struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"` // JSON Schema object
}

// vertexToolConfig controls how tools are used.
type vertexToolConfig struct {
	FunctionCallingConfig *vertexFunctionCallingConfig `json:"functionCallingConfig,omitempty"`
}

// vertexFunctionCallingConfig specifies how the model should call functions.
type vertexFunctionCallingConfig struct {
	Mode                 string   `json:"mode,omitempty"` // "AUTO", "ANY", "NONE"
	AllowedFunctionNames []string `json:"allowedFunctionNames,omitempty"`
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

func decodeAPIError(resp *http.Response) error {
	return base.DecodeAPIError("vertex", resp)
}
