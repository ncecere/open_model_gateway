package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/httpserver"
	"github.com/ncecere/open_model_gateway/backend/internal/runtime"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	rt, err := runtime.New(ctx, runtime.Options{})
	if err != nil {
		log.Fatalf("init runtime: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := rt.Shutdown(shutdownCtx); err != nil {
			log.Printf("runtime shutdown: %v", err)
		}
	}()

	container := rt.Container
	if container == nil {
		log.Fatalf("runtime container missing")
	}

	// Launch background jobs (batch worker, file sweeper) via the runtime.
	rt.StartJobs(ctx)

	server, err := httpserver.New(container, rt.HealthReporter())
	if err != nil {
		log.Fatalf("construct server: %v", err)
	}

	if err := server.Listen(ctx); err != nil && err != context.Canceled {
		log.Fatalf("server stopped: %v", err)
	}
}
