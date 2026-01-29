// Package tokenizer provides a registry of BPE tokenizers keyed by provider name.
// It wraps tiktoken-go to supply accurate token counting for OpenAI, Anthropic,
// Bedrock, Vertex, Groq, OpenRouter, and vLLM models. Providers without a native
// public tokenizer (Anthropic, Vertex, Groq, Llama) use cl100k_base as a
// best-effort approximation (~5-10% variance on English text, far better than
// the previous chars/4 heuristic).
package tokenizer

import (
	"fmt"
	"strings"
	"sync"

	tiktoken "github.com/pkoukk/tiktoken-go"
)

// Encoder counts tokens for a given text string.
type Encoder interface {
	// CountTokens returns the number of BPE tokens in text.
	CountTokens(text string) int
}

// tiktokenEncoder wraps a tiktoken.Tiktoken instance.
type tiktokenEncoder struct {
	enc *tiktoken.Tiktoken
}

func (t *tiktokenEncoder) CountTokens(text string) int {
	return len(t.enc.Encode(text, nil, nil))
}

// registry holds lazily-initialized encoders keyed by encoding name.
var (
	mu       sync.RWMutex
	encoders = make(map[string]*tiktokenEncoder)
)

// Encoding names supported by tiktoken-go.
const (
	O200kBase  = "o200k_base"  // GPT-4o, GPT-4.1, GPT-4.5
	CL100kBase = "cl100k_base" // GPT-4, GPT-3.5-turbo, embeddings
)

// providerEncodings maps route.Tokenizer values to tiktoken encoding names.
// Unknown providers fall back to cl100k_base.
var providerEncodings = map[string]string{
	"openai":            CL100kBase,
	"openai-o200k":      O200kBase,
	"azure":             CL100kBase,
	"anthropic":         CL100kBase, // approximation — Claude uses a custom tokenizer
	"bedrock":           CL100kBase, // Bedrock Claude uses the same approximation
	"vertex":            CL100kBase, // Gemini uses SentencePiece; cl100k is close enough
	"groq":              CL100kBase, // Groq hosts Llama/Mixtral; cl100k approximation
	"openrouter":        CL100kBase,
	"openai-compatible": CL100kBase,
	"vllm":              CL100kBase,
}

// getOrCreate returns a cached encoder or creates one.
func getOrCreate(encoding string) (*tiktokenEncoder, error) {
	mu.RLock()
	if enc, ok := encoders[encoding]; ok {
		mu.RUnlock()
		return enc, nil
	}
	mu.RUnlock()

	mu.Lock()
	defer mu.Unlock()
	// double-check after acquiring write lock
	if enc, ok := encoders[encoding]; ok {
		return enc, nil
	}

	tk, err := tiktoken.GetEncoding(encoding)
	if err != nil {
		return nil, fmt.Errorf("tokenizer: load encoding %q: %w", encoding, err)
	}
	enc := &tiktokenEncoder{enc: tk}
	encoders[encoding] = enc
	return enc, nil
}

// ForProvider returns an Encoder for the given provider/tokenizer key.
// The key corresponds to the route.Tokenizer field set by provider builders
// (e.g. "openai", "anthropic", "vertex"). Unknown keys fall back to cl100k_base.
func ForProvider(tokenizerKey string) (Encoder, error) {
	key := strings.ToLower(strings.TrimSpace(tokenizerKey))
	encoding, ok := providerEncodings[key]
	if !ok {
		encoding = CL100kBase
	}
	return getOrCreate(encoding)
}

// ForEncoding returns an Encoder for a specific tiktoken encoding name
// (e.g. "o200k_base", "cl100k_base").
func ForEncoding(encoding string) (Encoder, error) {
	return getOrCreate(encoding)
}

// CountTokens is a convenience function that counts tokens using the
// provider's tokenizer. Returns (count, nil) on success or (0, error).
func CountTokens(tokenizerKey, text string) (int, error) {
	enc, err := ForProvider(tokenizerKey)
	if err != nil {
		return 0, err
	}
	return enc.CountTokens(text), nil
}

// MustCountTokens is like CountTokens but falls back to len(text)/4 on error.
func MustCountTokens(tokenizerKey, text string) int {
	n, err := CountTokens(tokenizerKey, text)
	if err != nil {
		return len(text) / 4
	}
	return n
}

// EstimateMessageTokens counts tokens for a chat message including role overhead.
// It adds 4 tokens per message (role, separators) matching OpenAI's formula.
func EstimateMessageTokens(tokenizerKey, role, content string) int {
	enc, err := ForProvider(tokenizerKey)
	if err != nil {
		return len(content)/4 + 4
	}
	return enc.CountTokens(role) + enc.CountTokens(content) + 4
}
