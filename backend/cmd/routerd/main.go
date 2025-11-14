package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/batchworker"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	"github.com/ncecere/open_model_gateway/backend/internal/database"
	"github.com/ncecere/open_model_gateway/backend/internal/executor"
	"github.com/ncecere/open_model_gateway/backend/internal/httpserver"
	"github.com/ncecere/open_model_gateway/backend/internal/redisclient"
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

	server, err := httpserver.New(container)
	if err != nil {
		log.Fatalf("construct server: %v", err)
	}

	if err := server.Listen(ctx); err != nil && err != context.Canceled {
		log.Fatalf("server stopped: %v", err)
	}
}
