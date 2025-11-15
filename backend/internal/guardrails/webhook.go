package guardrails

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

// WebhookModerator invokes a tenant-provided webhook to evaluate prompts/responses.
type WebhookModerator struct {
	url        string
	authHeader string
	authValue  string
	timeout    time.Duration
	client     *http.Client
}

// NewWebhookModerator creates a webhook moderator from the moderation config.
func NewWebhookModerator(cfg ModerationConfig) (*WebhookModerator, error) {
	url := strings.TrimSpace(cfg.WebhookURL)
	if url == "" {
		return nil, nil
	}
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &WebhookModerator{
		url:        url,
		authHeader: strings.TrimSpace(cfg.WebhookAuthHeader),
		authValue:  strings.TrimSpace(cfg.WebhookAuthValue),
		timeout:    timeout,
		client:     &http.Client{Timeout: timeout},
	}, nil
}

// Evaluate sends the content to the webhook and returns the resulting action.
func (w *WebhookModerator) Evaluate(ctx context.Context, stage, content string) (Result, error) {
	if w == nil {
		return Result{Action: ActionAllow}, nil
	}
	stage = strings.TrimSpace(stage)
	if stage == "" || strings.TrimSpace(content) == "" {
		return Result{Action: ActionAllow}, nil
	}
	payload := webhookRequest{
		Stage:   stage,
		Content: content,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Result{Action: ActionAllow}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url, bytes.NewReader(body))
	if err != nil {
		return Result{Action: ActionAllow}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if w.authHeader != "" && w.authValue != "" {
		req.Header.Set(w.authHeader, w.authValue)
	}
	resp, err := w.client.Do(req)
	if err != nil {
		return Result{Action: ActionAllow}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return Result{Action: ActionAllow}, errors.New(resp.Status)
	}
	var webhookResp webhookResponse
	if err := json.NewDecoder(resp.Body).Decode(&webhookResp); err != nil {
		return Result{Action: ActionAllow}, err
	}
	action := parseAction(webhookResp.Action)
	res := Result{Action: action, Violations: webhookResp.Violations, Category: strings.TrimSpace(webhookResp.Category)}
	return res, nil
}

func parseAction(val string) Action {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case string(ActionBlock):
		return ActionBlock
	case string(ActionWarn):
		return ActionWarn
	default:
		return ActionAllow
	}
}

type webhookRequest struct {
	Stage   string `json:"stage"`
	Content string `json:"content"`
}

type webhookResponse struct {
	Action     string   `json:"action"`
	Category   string   `json:"category"`
	Violations []string `json:"violations"`
}
