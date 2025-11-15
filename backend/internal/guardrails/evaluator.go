package guardrails

import "strings"

// Action enumerates evaluator outcomes.
type Action string

const (
	ActionAllow Action = "allow"
	ActionWarn  Action = "warn"
	ActionBlock Action = "block"
)

// PreCheckInput captures text about to be sent upstream.
type PreCheckInput struct {
	Prompt string
}

// PostCheckInput captures the provider response.
type PostCheckInput struct {
	Completion string
}

// Result represents the evaluator decision.
type Result struct {
	Action     Action
	Violations []string
	Category   string
}

// Evaluator runs guardrail rules with a parsed config.
type Evaluator struct {
	config Config
}

func NewEvaluator(cfg Config) *Evaluator {
	return &Evaluator{config: cfg}
}

func (e *Evaluator) PreCheck(input PreCheckInput) Result {
	if !e.config.Enabled {
		return Result{Action: ActionAllow}
	}
	if violation := matchKeyword(e.config.Prompt.BlockedKeywords, input.Prompt); violation != "" {
		return Result{Action: ActionBlock, Violations: []string{violation}}
	}
	return Result{Action: ActionAllow}
}

func (e *Evaluator) PostCheck(input PostCheckInput) Result {
	if !e.config.Enabled {
		return Result{Action: ActionAllow}
	}
	if violation := matchKeyword(e.config.Response.BlockedKeywords, input.Completion); violation != "" {
		return Result{Action: ActionBlock, Violations: []string{violation}}
	}
	return Result{Action: ActionAllow}
}

func matchKeyword(keywords []string, text string) string {
	lower := strings.ToLower(text)
	for _, keyword := range keywords {
		kw := strings.ToLower(strings.TrimSpace(keyword))
		if kw == "" {
			continue
		}
		if strings.Contains(lower, kw) {
			return keyword
		}
	}
	return ""
}
