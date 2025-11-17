package runtime

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
	filesvc "github.com/ncecere/open_model_gateway/backend/internal/services/files"
)

// jobsStage coordinates background worker startups.
type jobsStage struct {
	once  sync.Once
	start func(ctx context.Context)
}

func (j *jobsStage) Start(ctx context.Context) {
	if j == nil || j.start == nil {
		return
	}
	j.once.Do(func() {
		j.start(ctx)
	})
}

func startFileSweeper(ctx context.Context, svc *filesvc.Service, cfg config.FilesConfig) {
	if svc == nil {
		return
	}
	interval := cfg.SweepInterval
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	batchSize := cfg.SweepBatchSize
	if batchSize <= 0 {
		batchSize = 200
	}
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		run := func() {
			if err := svc.SweepExpired(ctx, int32(batchSize)); err != nil {
				log.Printf("files sweeper error: %v", err)
			}
		}
		run()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
}
