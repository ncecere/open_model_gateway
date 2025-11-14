package usagepipeline

import (
	"context"
	"errors"
	"testing"
)

type stubSink struct {
	err      error
	calls    int
	payloads []AlertPayload
}

func (s *stubSink) Notify(ctx context.Context, payload AlertPayload) error {
	s.calls++
	s.payloads = append(s.payloads, payload)
	return s.err
}

func TestCompositeSinkNotify(t *testing.T) {
	p := AlertPayload{}
	okSink := &stubSink{}
	errSink := &stubSink{err: errors.New("boom")}

	sink := NewCompositeSink(okSink, errSink).(*CompositeSink)
	if err := sink.Notify(context.Background(), p); err == nil {
		t.Fatalf("expected error from composite sink")
	}
	if okSink.calls != 1 || errSink.calls != 1 {
		t.Fatalf("expected sinks to be invoked once each")
	}
}

func TestCompositeSinkSkipsNil(t *testing.T) {
	sink := NewCompositeSink(nil)
	if sink != nil {
		t.Fatalf("expected nil sink when no entries provided")
	}
}
