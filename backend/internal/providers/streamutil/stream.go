package streamutil

import (
	"context"
	"sync"

	"github.com/ncecere/open_model_gateway/backend/internal/models"
)

// YieldFunc receives converted chat chunks. Returning false stops further forwarding.
type YieldFunc func(models.ChatChunk) bool

// Forward wraps provider-specific streaming logic with a shared channel lifecycle so adapters follow the
// same contract when emitting chat chunks. The forward callback should invoke yield for every chunk until it
// returns false or the stream is exhausted.
func Forward(ctx context.Context, closer func() error, forward func(ctx context.Context, yield YieldFunc)) (<-chan models.ChatChunk, func() error) {
	chunks := make(chan models.ChatChunk)
	var once sync.Once
	callCloser := func() {
		if closer == nil {
			return
		}
		once.Do(func() {
			_ = closer()
		})
	}

	go func() {
		defer close(chunks)
		defer callCloser()

		forward(ctx, func(chunk models.ChatChunk) bool {
			select {
			case <-ctx.Done():
				return false
			case chunks <- chunk:
				return true
			}
		})
	}()

	return chunks, func() error {
		callCloser()
		return nil
	}
}
