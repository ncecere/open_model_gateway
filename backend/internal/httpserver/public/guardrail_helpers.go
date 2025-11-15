package public

import (
	"context"
	"log/slog"
	"strings"

	guardrails "github.com/ncecere/open_model_gateway/backend/internal/guardrails"
	"github.com/ncecere/open_model_gateway/backend/internal/models"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	usagepipeline "github.com/ncecere/open_model_gateway/backend/internal/services/usagepipeline"
)

const (
	guardrailStagePrompt   = "prompt"
	guardrailStageResponse = "response"

	guardrailPromptMessage   = "prompt rejected by guardrails"
	guardrailResponseMessage = "response blocked by guardrails"
	guardrailErrorCode       = "guardrail_blocked"
)

type guardrailRuntime struct {
	evaluator *guardrails.Evaluator
	cfg       guardrails.Config
	ctx       context.Context
	moderator *guardrails.WebhookModerator
}

func (h *openAIHandler) loadGuardrailRuntime(ctx context.Context, rc *requestctx.Context) (guardrailRuntime, error) {
	var runtime guardrailRuntime
	if h == nil || h.container == nil || h.container.Guardrails == nil || rc == nil {
		return runtime, nil
	}
	cfgMap, err := h.container.Guardrails.EffectivePolicy(ctx, rc.TenantID, rc.APIKeyID)
	if err != nil {
		return runtime, err
	}
	cfg := guardrails.ParseConfig(cfgMap)
	if !cfg.Enabled {
		return runtime, nil
	}
	runtime.cfg = cfg
	runtime.evaluator = guardrails.NewEvaluator(cfg)
	runtime.ctx = ctx
	if cfg.Moderation.Enabled && strings.EqualFold(cfg.Moderation.Provider, "webhook") {
		moderator, err := guardrails.NewWebhookModerator(cfg.Moderation)
		if err != nil {
			slog.Warn("guardrail webhook init", slog.String("error", err.Error()))
		} else {
			runtime.moderator = moderator
		}
	}
	return runtime, nil
}

func (rt guardrailRuntime) Enabled() bool {
	return rt.evaluator != nil && rt.cfg.Enabled
}

func (rt guardrailRuntime) PreCheck(prompt string) guardrails.Result {
	if !rt.Enabled() || strings.TrimSpace(prompt) == "" {
		return guardrails.Result{Action: guardrails.ActionAllow}
	}
	if res := rt.evaluator.PreCheck(guardrails.PreCheckInput{Prompt: prompt}); res.Action == guardrails.ActionBlock {
		return res
	}
	if res := rt.evaluateModeration(guardrailStagePrompt, prompt); res.Action != guardrails.ActionAllow {
		return res
	}
	return guardrails.Result{Action: guardrails.ActionAllow}
}

func (rt guardrailRuntime) PostCheck(output string) guardrails.Result {
	if !rt.Enabled() || strings.TrimSpace(output) == "" {
		return guardrails.Result{Action: guardrails.ActionAllow}
	}
	if res := rt.evaluator.PostCheck(guardrails.PostCheckInput{Completion: output}); res.Action == guardrails.ActionBlock {
		return res
	}
	if res := rt.evaluateModeration(guardrailStageResponse, output); res.Action != guardrails.ActionAllow {
		return res
	}
	return guardrails.Result{Action: guardrails.ActionAllow}
}

func (rt guardrailRuntime) evaluateModeration(stage, content string) guardrails.Result {
	if rt.moderator == nil {
		return guardrails.Result{Action: guardrails.ActionAllow}
	}
	ctx := rt.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	res, err := rt.moderator.Evaluate(ctx, stage, content)
	if err != nil {
		slog.Warn("guardrail webhook", slog.String("stage", stage), slog.String("error", err.Error()))
		return guardrails.Result{Action: guardrails.ActionAllow}
	}
	return res
}

func (h *openAIHandler) recordGuardrailEvent(ctx context.Context, rc *requestctx.Context, alias, stage string, result guardrails.Result, metadata map[string]any) {
	if result.Action == guardrails.ActionAllow || h == nil || h.container == nil || h.container.Guardrails == nil || rc == nil {
		return
	}
	details := map[string]any{
		"stage":      stage,
		"violations": result.Violations,
	}
	if strings.TrimSpace(result.Category) != "" {
		details["category"] = result.Category
	}
	for k, v := range metadata {
		details[k] = v
	}
	if err := h.container.Guardrails.RecordEvent(ctx, rc.TenantID, rc.APIKeyID, alias, string(result.Action), stage, details); err != nil {
		slog.Warn("record guardrail event", slog.String("alias", alias), slog.String("stage", stage), slog.String("error", err.Error()))
	}
	h.dispatchGuardrailAlert(ctx, rc, alias, stage, result, details)
}

func chatPromptText(messages []models.ChatMessage) string {
	var builder strings.Builder
	for _, msg := range messages {
		if strings.TrimSpace(msg.Content) == "" {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(msg.Content)
	}
	return builder.String()
}

func chatResponseText(resp models.ChatResponse) string {
	var builder strings.Builder
	for _, choice := range resp.Choices {
		if strings.TrimSpace(choice.Message.Content) == "" {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(choice.Message.Content)
	}
	return builder.String()
}

func embeddingPromptText(inputs []string) string {
	var builder strings.Builder
	for _, input := range inputs {
		if strings.TrimSpace(input) == "" {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(input)
	}
	return builder.String()
}

func embeddingResponseText(models.EmbeddingsResponse) string {
	return ""
}

func imageResponseText(resp models.ImageResponse) string {
	var builder strings.Builder
	for _, item := range resp.Data {
		if strings.TrimSpace(item.RevisedPrompt) == "" {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(item.RevisedPrompt)
	}
	return builder.String()
}

type guardrailStreamMonitor struct {
	runtime guardrailRuntime
	buffers map[int]*strings.Builder
}

func newGuardrailStreamMonitor(runtime guardrailRuntime) *guardrailStreamMonitor {
	if !runtime.Enabled() {
		return nil
	}
	return &guardrailStreamMonitor{runtime: runtime, buffers: map[int]*strings.Builder{}}
}

func (m *guardrailStreamMonitor) Process(chunk models.ChatChunk) (guardrails.Result, bool) {
	if m == nil {
		return guardrails.Result{Action: guardrails.ActionAllow}, false
	}
	for _, choice := range chunk.Choices {
		content := choice.Delta.Content
		if strings.TrimSpace(content) == "" {
			continue
		}
		buf, ok := m.buffers[choice.Index]
		if !ok {
			buf = &strings.Builder{}
			m.buffers[choice.Index] = buf
		}
		buf.WriteString(content)
		res := m.runtime.PostCheck(buf.String())
		if res.Action == guardrails.ActionBlock {
			return res, true
		}
	}
	return guardrails.Result{Action: guardrails.ActionAllow}, false
}

func guardrailBlocked(res guardrails.Result) bool {
	return res.Action == guardrails.ActionBlock
}

func (h *openAIHandler) dispatchGuardrailAlert(ctx context.Context, rc *requestctx.Context, alias, stage string, result guardrails.Result, metadata map[string]any) {
	if result.Action != guardrails.ActionBlock || h == nil || h.container == nil || h.container.UsageLogger == nil {
		return
	}
	dispatcher := h.container.UsageLogger.AlertDispatcher()
	if dispatcher == nil {
		return
	}
	info := usagepipeline.GuardrailAlertInfo{
		Stage:      stage,
		Action:     string(result.Action),
		Violations: result.Violations,
		ModelAlias: alias,
	}
	if category, ok := metadata["category"].(string); ok {
		info.Category = category
	} else if endpoint, ok := metadata["endpoint"].(string); ok {
		info.Category = endpoint
	}
	_ = dispatcher.DispatchGuardrail(ctx, rc, info)
}
