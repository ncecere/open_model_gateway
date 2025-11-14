package httpserver

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/ncecere/open_model_gateway/backend/internal/app"
	"github.com/ncecere/open_model_gateway/backend/internal/config"
	adminroutes "github.com/ncecere/open_model_gateway/backend/internal/httpserver/admin"
	publicroutes "github.com/ncecere/open_model_gateway/backend/internal/httpserver/public"
	userroutes "github.com/ncecere/open_model_gateway/backend/internal/httpserver/user"
)

// Server wraps the Fiber app and configuration.
type Server struct {
	app       *fiber.App
	cfg       *config.Config
	container *app.Container
}

// New constructs a server with baseline middleware ready.
func New(container *app.Container) (*Server, error) {
	if container == nil {
		return nil, fmt.Errorf("dependency container is required")
	}

	cfg := container.Config
	if cfg == nil {
		return nil, fmt.Errorf("container missing config")
	}

	bodyLimit := cfg.Server.BodyLimitMB * 1024 * 1024
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ServerHeader:          "open-model-gateway",
		BodyLimit:             bodyLimit,
		ReadTimeout:           cfg.Server.SyncTimeout,
		IdleTimeout:           cfg.Server.StreamMaxDuration,
		ReadBufferSize:        4 * 1024,
		WriteBufferSize:       4 * 1024,
	})

	app.Use(requestid.New())
	app.Use(logger.New())
	app.Use(recover.New())

	if container.Observability != nil {
		app.Use(func(c *fiber.Ctx) error {
			start := time.Now()
			err := c.Next()
			route := ""
			if r := c.Route(); r != nil {
				route = r.Path
			}
			if route == "" {
				route = c.Path()
			}
			container.Observability.RecordHTTPRequest(c.UserContext(), c.Method(), route, c.Response().StatusCode(), time.Since(start))
			return err
		})
	}

	if container.Observability != nil && container.Observability.TracerProvider() != nil {
		tracer := otel.Tracer("open-model-gateway/http")
		app.Use(func(c *fiber.Ctx) error {
			spanCtx, span := tracer.Start(c.UserContext(), c.Method()+" "+c.Path())
			c.SetUserContext(spanCtx)
			err := c.Next()
			route := ""
			if r := c.Route(); r != nil {
				route = r.Path
			}
			span.SetAttributes(
				attribute.String("http.method", c.Method()),
				attribute.String("http.route", route),
				attribute.Int("http.status_code", c.Response().StatusCode()),
			)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			} else if status := c.Response().StatusCode(); status >= 500 {
				span.SetStatus(codes.Error, fmt.Sprintf("status %d", status))
			} else {
				span.SetStatus(codes.Ok, "OK")
			}
			span.End()
			return err
		})
	}

	if container.Observability != nil {
		if handler := container.Observability.PrometheusHandler(); handler != nil {
			app.Get("/metrics", adaptor.HTTPHandler(handler))
		}
	}

	registerHealthRoutes(app, container)
	mountAdminUISubpath(app)
	adminroutes.Register(app, container)
	userroutes.Register(app, container)
	publicroutes.Register(app, container)
	mountEmbeddedUI(app)

	return &Server{
		app:       app,
		cfg:       cfg,
		container: container,
	}, nil
}

// Listen blocks until context cancellation or a fatal listen error occurs.
func (s *Server) Listen(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.app.Listen(s.cfg.Server.ListenAddr)
	}()

	select {
	case <-ctx.Done():
		timeout := s.cfg.Server.GracefulShutdownDelay
		if timeout <= 0 {
			timeout = 5 * time.Second
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := s.app.ShutdownWithContext(shutdownCtx)
		if err == nil {
			err = <-errCh
		}
		return err
	case err := <-errCh:
		return err
	}
}

func registerHealthRoutes(app *fiber.App, container *app.Container) {
	app.Get("/healthz", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
		defer cancel()

		checks := make(map[string]fiber.Map)
		overall := "ok"

		if container != nil && container.DBPool != nil {
			start := time.Now()
			err := container.DBPool.Ping(ctx)
			latency := time.Since(start)
			status := "ok"
			check := fiber.Map{
				"status":     status,
				"latency_ms": latency.Milliseconds(),
			}
			if err != nil {
				check["status"] = "error"
				check["error"] = err.Error()
				overall = "degraded"
			}
			checks["postgres"] = check
		}

		if container != nil && container.Redis != nil {
			start := time.Now()
			err := container.Redis.Ping(ctx).Err()
			latency := time.Since(start)
			check := fiber.Map{
				"status":     "ok",
				"latency_ms": latency.Milliseconds(),
			}
			if err != nil {
				check["status"] = "error"
				check["error"] = err.Error()
				overall = "degraded"
			}
			checks["redis"] = check
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": overall,
			"checks": checks,
		})
	})
}
