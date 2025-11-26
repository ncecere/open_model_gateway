package provider

import (
	"context"
	"testing"
	"time"
)

type countingSink struct {
	count int
	last  Incident
}

func (s *countingSink) Notify(_ context.Context, inc Incident) error {
	s.count++
	s.last = inc
	return nil
}

func TestIncidentDispatcherCooldown(t *testing.T) {
	sink := &countingSink{}
	dispatcher := NewIncidentDispatcher(AlertConfig{Cooldown: time.Hour}, sink)

	inc := Incident{
		Identifier: Identifier{Provider: "openai", ModelAlias: "gpt-5"},
		Type:       "error_rate",
		OpenedAt:   time.Now().UTC(),
	}
	if err := dispatcher.Dispatch(context.Background(), inc); err != nil {
		t.Fatalf("first dispatch failed: %v", err)
	}
	if err := dispatcher.Dispatch(context.Background(), inc); err != nil {
		t.Fatalf("second dispatch failed: %v", err)
	}
	if sink.count != 1 {
		t.Fatalf("expected 1 notification due to cooldown, got %d", sink.count)
	}

	inc2 := inc
	inc2.OpenedAt = inc.OpenedAt.Add(time.Hour + time.Second)
	if err := dispatcher.Dispatch(context.Background(), inc2); err != nil {
		t.Fatalf("third dispatch failed: %v", err)
	}
	if sink.count != 2 {
		t.Fatalf("expected second notification after cooldown, got %d", sink.count)
	}
	if sink.last.OpenedAt != inc2.OpenedAt {
		t.Fatalf("expected last incident to match newer opened_at")
	}
}
