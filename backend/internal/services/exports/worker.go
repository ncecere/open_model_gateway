package exports

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
)

// Worker processes queued usage exports asynchronously.
type Worker struct {
	service      *Service
	logger       *slog.Logger
	pollInterval time.Duration
}

func NewWorker(service *Service, logger *slog.Logger) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{
		service:      service,
		logger:       logger,
		pollInterval: 3 * time.Second,
	}
}

func (w *Worker) Run(ctx context.Context) {
	if w == nil || w.service == nil {
		return
	}
	for {
		if ctx.Err() != nil {
			return
		}
		export, err := w.service.ProcessNext(ctx)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) || IsNotFound(err) {
				select {
				case <-ctx.Done():
					return
				case <-time.After(w.pollInterval):
				}
				continue
			}
			w.logger.Error("usage export worker failed", slog.String("error", err.Error()))
			select {
			case <-ctx.Done():
				return
			case <-time.After(w.pollInterval):
			}
			continue
		}
		w.logger.Info("usage export completed", slog.String("export_id", export.ID.String()), slog.String("status", export.Status))
	}
}
