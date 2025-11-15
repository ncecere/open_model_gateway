package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/batchworker"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/database"
	"github.com/ncecere/open_model_gateway/backend/internal/executor"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver"
	"github.com/ncecere/open_model_gateway/backend/internal/redisclient"
	filesvc "github.com/ncecere/open_model_gateway/backend/internal/services/files"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(config.Options{})
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if err := database.RunMigrations(ctx, cfg.Database); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	dbPool, err := database.Connect(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer dbPool.Close()

	redisClient := redisclient.New(cfg.Redis)
	if err := redisclient.Ping(ctx, redisClient); err != nil {
		log.Fatalf("connect redis: %v", err)
	}
	defer redisClient.Close()

	container, err := app.NewContainer(ctx, cfg, dbPool, redisClient)
	if err != nil {
		log.Fatalf("build container: %v", err)
	}
	if container.Observability != nil {
		defer container.Observability.Shutdown(ctx)
	}

	if container.Batches != nil {
		go batchworker.New(container, executor.New(container)).Run(ctx)
	}
	if container.Files != nil {
		startFileSweeper(ctx, container.Files, cfg.Files)
	}

	server, err := httpserver.New(container)
	if err != nil {
		log.Fatalf("construct server: %v", err)
	}

	if err := server.Listen(ctx); err != nil && err != context.Canceled {
		log.Fatalf("server stopped: %v", err)
	}
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
