package usagepipeline

import (
	"context"
	"errors"
)

// CompositeSink fans out notifications to multiple sinks.
type CompositeSink struct {
	sinks []AlertSink
}

func NewCompositeSink(sinks ...AlertSink) AlertSink {
	filtered := make([]AlertSink, 0, len(sinks))
	for _, sink := range sinks {
		if sink == nil {
			continue
		}
		filtered = append(filtered, sink)
	}
	if len(filtered) == 0 {
		return nil
	}
	return &CompositeSink{sinks: filtered}
}

func (c *CompositeSink) Notify(ctx context.Context, payload AlertPayload) error {
	if c == nil {
		return nil
	}
	var err error
	for _, sink := range c.sinks {
		if sink == nil {
			continue
		}
		if notifyErr := sink.Notify(ctx, payload); notifyErr != nil {
			err = errorsJoin(err, notifyErr)
		}
	}
	return err
}

func errorsJoin(base error, next error) error {
	if base == nil {
		return next
	}
	if next == nil {
		return base
	}
	return errors.Join(base, next)
}
