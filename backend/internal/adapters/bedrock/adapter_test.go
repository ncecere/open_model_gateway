package bedrock

import (
	"testing"

	"github.com/ncecere/open_model_gateway/backend/internal/providers/fixtures"
)

func TestAnthropicStreamUsageFixture(t *testing.T) {
	var evt anthropicStreamEvent
	if err := fixtures.Load("bedrock_stream_delta.json", &evt); err != nil {
		t.Fatalf("load fixture: %v", err)
	}

	if evt.Type != "message_delta" {
		t.Fatalf("unexpected type %q", evt.Type)
	}
	if evt.Usage.InputTokens != 27 || evt.Usage.OutputTokens != 580 {
		t.Fatalf("unexpected usage: %+v", evt.Usage)
	}
}
