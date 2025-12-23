package batchworker

import (
	"context"
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/db"
	"github.com/ncecere/open_model_gateway/backend/internal/executor"
	"github.com/ncecere/open_model_gateway/backend/internal/requestctx"
	batchsvc "github.com/ncecere/open_model_gateway/backend/internal/services/batches"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Worker processes queued batch jobs and executes the corresponding /v1/* calls.
type Worker struct {
	container    *app.Container
	executor     *executor.Executor
	logger       *slog.Logger
	pollInterval time.Duration
}

func New(container *app.Container, exec *executor.Executor) *Worker {
	interval := 2 * time.Second
	return &Worker{
		container:    container,
		executor:     exec,
		logger:       slog.Default(),
		pollInterval: interval,
	}
}

// Run begins polling for queued batches until the context is canceled.

func (w *Worker) Run(ctx context.Context) {
	if w == nil || w.container == nil || w.container.Batches == nil || w.executor == nil {
		return
	}

	for {
		if ctx.Err() != nil {
			return
		}

		handled, err := w.processNextBatch(ctx)
		if err != nil {
			w.logger.Error("batch worker: process batch", slog.String("error", err.Error()))
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
			continue
		}

		if !handled {
			select {
			case <-ctx.Done():
				return
			case <-time.After(w.pollInterval):
			}
		}
	}
}

func (w *Worker) processNextBatch(ctx context.Context) (bool, error) {
	batch, err := w.container.Batches.ClaimNextBatch(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	w.logger.Info("batch worker: claimed batch", slog.String("batch_id", batch.ID.String()), slog.String("endpoint", batch.Endpoint))
	if err := w.processBatch(ctx, batch); err != nil {
		return true, err
	}
	return true, nil
}

func (w *Worker) processBatch(ctx context.Context, batch batchsvc.Batch) error {
	rc, err := w.buildRequestContext(ctx, batch)
	if err != nil {
		w.logger.Error("batch worker: build request context", slog.String("batch_id", batch.ID.String()), slog.String("error", err.Error()))
		return w.failEntireBatch(ctx, batch, "context_error", err.Error())
	}

	workerCount := batch.MaxConcurrency
	if workerCount <= 0 {
		workerCount = 1
	}

	tracePrefix := fmt.Sprintf("batch_%s_", batch.ID.String())
	writer := newResultWriter(w.container.Files, batch, fileTTL(batch))
	var completedCount atomic.Int64
	var failedCount atomic.Int64

	workCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 1)
	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if workCtx.Err() != nil {
					return
				}
				itemRow, err := w.container.Batches.ClaimNextItem(workCtx, batch.ID)
				if err != nil {
					if errors.Is(err, pgx.ErrNoRows) {
						return
					}
					w.sendWorkerError(errCh, err)
					cancel()
					return
				}
				itemID, err := fromPgUUID(itemRow.ID)
				if err != nil {
					w.sendWorkerError(errCh, err)
					cancel()
					return
				}
				item := batchItem{
					ID:       itemID,
					Index:    itemRow.ItemIndex,
					CustomID: strings.TrimSpace(itemRow.CustomID.String),
					Input:    itemRow.Input,
				}
				traceID := fmt.Sprintf("%s%d", tracePrefix, item.Index)
				result := w.executeItem(workCtx, batch, rc, traceID, item)

				if result.errPayload == nil {
					if err := w.container.Batches.CompleteItem(workCtx, item.ID, result.response); err != nil {
						w.sendWorkerError(errCh, err)
						cancel()
						return
					}
					completedCount.Add(1)
					if err := writer.AppendSuccess(item, result.statusCode, result.requestID, result.response); err != nil {
						w.sendWorkerError(errCh, err)
						cancel()
						return
					}
				} else {
					if err := w.container.Batches.FailItem(workCtx, item.ID, result.errPayload); err != nil {
						w.sendWorkerError(errCh, err)
						cancel()
						return
					}
					failedCount.Add(1)
					if err := writer.AppendError(item, result.statusCode, result.requestID, result.errPayload); err != nil {
						w.sendWorkerError(errCh, err)
						cancel()
						return
					}
				}
			}
		}()
	}

	wg.Wait()

	select {
	case workerErr := <-errCh:
		if workerErr != nil {
			return workerErr
		}
	default:
	}

	completed := int(completedCount.Load())
	failed := int(failedCount.Load())

	if err := w.container.Batches.IncrementCounts(ctx, batch.ID, completed, failed, 0); err != nil {
		return err
	}

	if _, err := w.container.Batches.FinalizeBatch(ctx, batch.ID, "finalizing", nil, nil, nil); err != nil {
		return err
	}

	resultFileID, errorFileID, err := writer.Flush(ctx)
	if err != nil {
		return err
	}

	latest, err := w.container.Batches.GetByID(ctx, batch.ID)
	if err != nil {
		return err
	}

	finalStatus := determineFinalStatus(latest.Status, completed, failed)
	cancelledCount := 0
	if finalStatus == "cancelled" {
		remaining := batch.RequestCountTotal - completed - failed
		if remaining > 0 {
			cancelledCount = remaining
		}
	}
	if cancelledCount > 0 {
		if err := w.container.Batches.IncrementCounts(ctx, batch.ID, 0, 0, cancelledCount); err != nil {
			return err
		}
	}

	_, err = w.container.Batches.FinalizeBatch(ctx, batch.ID, finalStatus, resultFileID, errorFileID, nil)
	return err
}

func determineFinalStatus(currentStatus string, completed, failed int) string {
	switch currentStatus {
	case "cancelling", "cancelled":
		return "cancelled"
	}
	if failed > 0 {
		return "failed"
	}
	if completed == 0 {
		return "failed"
	}
	return "completed"
}

func (w *Worker) sendWorkerError(ch chan<- error, err error) {
	if err == nil {
		return
	}
	select {
	case ch <- err:
	default:
	}
}

func (w *Worker) buildRequestContext(ctx context.Context, batch batchsvc.Batch) (*requestctx.Context, error) {
	keyRow, err := w.container.Queries.GetAPIKeyByID(ctx, toPgUUID(batch.APIKeyID))
	if err != nil {
		return nil, err
	}
	if keyRow.RevokedAt.Valid {
		return nil, fmt.Errorf("api key revoked")
	}

	tenantRow, err := w.container.Queries.GetTenantByID(ctx, keyRow.TenantID)
	if err != nil {
		return nil, err
	}
	if tenantRow.Status != db.TenantStatusActive {
		return nil, fmt.Errorf("tenant is not active")
	}

	return app.BuildRequestContext(ctx, w.container, keyRow)
}

func (w *Worker) failEntireBatch(ctx context.Context, batch batchsvc.Batch, code, message string) error {
	writer := newResultWriter(w.container.Files, batch, fileTTL(batch))
	var failed int
	for {
		itemRow, err := w.container.Batches.ClaimNextItem(ctx, batch.ID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				break
			}
			return err
		}
		itemID, err := fromPgUUID(itemRow.ID)
		if err != nil {
			return err
		}
		payload := encodeErrorPayload(code, message)
		if err := w.container.Batches.FailItem(ctx, itemID, payload); err != nil {
			return err
		}
		failed++
		traceID := fmt.Sprintf("batch_%s_%d", batch.ID.String(), itemRow.ItemIndex)
		_ = writer.AppendError(batchItem{
			ID:       itemID,
			CustomID: strings.TrimSpace(itemRow.CustomID.String),
			Index:    itemRow.ItemIndex,
		}, fiber.StatusInternalServerError, traceID, payload)
	}

	if failed > 0 {
		if err := w.container.Batches.IncrementCounts(ctx, batch.ID, 0, failed, 0); err != nil {
			return err
		}
	}

	resultFileID, errorFileID, err := writer.Flush(ctx)
	if err != nil {
		return err
	}

	errs := []batchsvc.BatchError{
		{Code: code, Message: message},
	}
	_, err = w.container.Batches.FinalizeBatch(ctx, batch.ID, "failed", resultFileID, errorFileID, errs)
	return err
}

func (w *Worker) executeItem(ctx context.Context, batch batchsvc.Batch, rc *requestctx.Context, traceID string, item batchItem) itemOutcome {
	switch batch.Endpoint {
	case "/v1/chat/completions":
		return w.runChatItem(ctx, rc, traceID, item)
	case "/v1/responses":
		return w.runResponsesItem(ctx, rc, traceID, item)
	case "/v1/embeddings":
		return w.runEmbeddingItem(ctx, rc, traceID, item)

	case "/v1/moderations":
		return w.runModerationItem(ctx, rc, traceID, item)
	case "/v1/images/generations":
		return w.runImageItem(ctx, rc, traceID, item)
	case "/v1/images/edits":
		return w.runImageEditItem(ctx, rc, traceID, item)
	case "/v1/images/variations":
		return w.runImageVariationItem(ctx, rc, traceID, item)
	default:
		errPayload := encodeErrorPayload("unsupported_endpoint", fmt.Sprintf("endpoint %s not supported yet", batch.Endpoint))
		return itemOutcome{
			statusCode: fiber.StatusBadRequest,
			requestID:  traceID,
			errPayload: errPayload,
		}
	}
}
