package bedrock

import (
	"testing"

	"github.com/ncecere/open_model_gateway/backend/internal/providers/fixtures"
)

func TestAnthropicStopReasonFixture(t *testing.T) {
	var evt anthropicStreamEvent
	if err := fixtures.Load("bedrock_stream_delta.json", &evt); err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	if got := mapAnthropicStopReason(evt.StopReason()); got != "stop" {
		t.Fatalf("expected stop reason remapped to stop, got %s", got)
	}
}

func TestParseTitanEmbeddingFixtures(t *testing.T) {
	primary, err := fixtures.Read("titan_embed_primary.json")
	if err != nil {
		t.Fatalf("read primary: %v", err)
	}
	vec, tokens, err := parseTitanEmbedding(primary)
	if err != nil {
		t.Fatalf("parse primary: %v", err)
	}
	if tokens != 27 {
		t.Fatalf("expected 27 tokens, got %d", tokens)
	}
	if len(vec) != 3 || vec[0] != float32(0.12) || vec[1] != float32(-0.34) || vec[2] != float32(0.56) {
		t.Fatalf("unexpected primary vector: %v", vec)
	}

	alt, err := fixtures.Read("titan_embed_alt.json")
	if err != nil {
		t.Fatalf("read alt: %v", err)
	}
	vec, tokens, err = parseTitanEmbedding(alt)
	if err != nil {
		t.Fatalf("parse alt: %v", err)
	}
	if tokens != 14 {
		t.Fatalf("expected 14 tokens, got %d", tokens)
	}
	if len(vec) != 2 || vec[0] != float32(0.9) || vec[1] != float32(0.1) {
		t.Fatalf("unexpected alt vector: %v", vec)
	}
}

func TestClampImageCount(t *testing.T) {
	cases := map[int]int{
		-5: 1,
		0:  1,
		1:  1,
		3:  3,
		10: 4,
	}
	for input, want := range cases {
		if got := clampImageCount(input, 4); got != want {
			t.Fatalf("input %d: want %d got %d", input, want, got)
		}
	}
}
