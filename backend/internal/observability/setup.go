package observability

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	promreg "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

type Provider struct {
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *metric.MeterProvider
	promExporter   *prometheus.Exporter
	promHandler    http.Handler
	shutdownFuncs  []func(context.Context) error

	httpRequestCounter *promreg.CounterVec
	httpRequestLatency *promreg.HistogramVec
	apiLatencyHist     *promreg.HistogramVec
	apiTokensCounter   *promreg.CounterVec
}

func Setup(ctx context.Context, cfg config.ObservabilityConfig) (*Provider, error) {
	if !cfg.EnableOTLP && !cfg.EnableMetrics {
		return nil, nil
	}

	provider := &Provider{}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("open-model-gateway"),
		),
	)
	if err != nil {
		return nil, err
	}

	if cfg.EnableOTLP {
		rawEndpoint := strings.TrimSpace(cfg.OTLPEndpoint)
		endpoint := rawEndpoint
		if endpoint == "" {
			endpoint = "localhost:4317"
		}
		opts := []otlptracegrpc.Option{}
		switch {
		case strings.HasPrefix(endpoint, "http://"):
			endpoint = strings.TrimPrefix(endpoint, "http://")
			opts = append(opts, otlptracegrpc.WithInsecure())
		case strings.HasPrefix(endpoint, "https://"):
			endpoint = strings.TrimPrefix(endpoint, "https://")
		default:
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		opts = append(opts, otlptracegrpc.WithEndpoint(endpoint))

		client := otlptracegrpc.NewClient(opts...)
		exporter, err := otlptrace.New(ctx, client)
		if err != nil {
			return nil, err
		}
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
		)
		otel.SetTracerProvider(tp)
		provider.tracerProvider = tp
		provider.shutdownFuncs = append(provider.shutdownFuncs, tp.Shutdown)
	}

	if cfg.EnableMetrics {
		registry := promreg.NewRegistry()
		promExporter, err := prometheus.New(prometheus.WithRegisterer(registry))
		if err != nil {
			return nil, err
		}
		mp := metric.NewMeterProvider(
			metric.WithReader(promExporter),
			metric.WithResource(res),
		)
		otel.SetMeterProvider(mp)
		provider.meterProvider = mp
		provider.promExporter = promExporter
		provider.promHandler = promhttp.HandlerFor(registry, promhttp.HandlerOpts{EnableOpenMetrics: true})
		provider.shutdownFuncs = append(provider.shutdownFuncs, mp.Shutdown)

		httpRequests := promreg.NewCounterVec(
			promreg.CounterOpts{
				Namespace: "open_model_gateway",
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests processed.",
			},
			[]string{"method", "route", "status"},
		)
		latencyBuckets := []float64{0.05, 0.1, 0.2, 0.5, 1, 2, 5, 10}
		httpLatency := promreg.NewHistogramVec(
			promreg.HistogramOpts{
				Namespace: "open_model_gateway",
				Name:      "http_request_duration_seconds",
				Help:      "Duration of HTTP requests in seconds.",
				Buckets:   latencyBuckets,
			},
			[]string{"method", "route", "status"},
		)
		apiLatency := promreg.NewHistogramVec(
			promreg.HistogramOpts{
				Namespace: "open_model_gateway",
				Name:      "api_request_duration_seconds",
				Help:      "Duration of upstream model requests.",
				Buckets:   latencyBuckets,
			},
			[]string{"tenant", "model", "provider", "status"},
		)
		tokenCounter := promreg.NewCounterVec(
			promreg.CounterOpts{
				Namespace: "open_model_gateway",
				Name:      "api_tokens_total",
				Help:      "Total prompt/completion tokens processed.",
			},
			[]string{"tenant", "model", "provider", "type"},
		)
		if err := registry.Register(httpRequests); err != nil {
			return nil, err
		}
		if err := registry.Register(httpLatency); err != nil {
			return nil, err
		}
		if err := registry.Register(apiLatency); err != nil {
			return nil, err
		}
		if err := registry.Register(tokenCounter); err != nil {
			return nil, err
		}
		provider.httpRequestCounter = httpRequests
		provider.httpRequestLatency = httpLatency
		provider.apiLatencyHist = apiLatency
		provider.apiTokensCounter = tokenCounter
	}

	return provider, nil
}

func (p *Provider) PrometheusHandler() http.Handler {
	if p == nil || p.promHandler == nil {
		return nil
	}
	return p.promHandler
}

func (p *Provider) Shutdown(ctx context.Context) error {
	if p == nil {
		return nil
	}
	for _, fn := range p.shutdownFuncs {
		if err := fn(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (p *Provider) TracerProvider() *sdktrace.TracerProvider {
	if p == nil {
		return nil
	}
	return p.tracerProvider
}

func (p *Provider) RecordHTTPRequest(_ context.Context, method, route string, status int, duration time.Duration) {
	if p == nil {
		return
	}

	statusLabel := strconv.Itoa(status)

	if p.httpRequestCounter != nil {
		p.httpRequestCounter.WithLabelValues(method, route, statusLabel).Inc()
	}

	if p.httpRequestLatency != nil {
		p.httpRequestLatency.WithLabelValues(method, route, statusLabel).Observe(duration.Seconds())
	}
}

func (p *Provider) RecordAPILatency(tenantID, model, provider string, status int, duration time.Duration) {
	if p == nil || p.apiLatencyHist == nil {
		return
	}
	statusLabel := strconv.Itoa(status)
	p.apiLatencyHist.WithLabelValues(tenantID, model, provider, statusLabel).Observe(duration.Seconds())
}

func (p *Provider) RecordTokens(tenantID, model, provider string, promptTokens, completionTokens int64) {
	if p == nil || p.apiTokensCounter == nil {
		return
	}
	if promptTokens > 0 {
		p.apiTokensCounter.WithLabelValues(tenantID, model, provider, "prompt").Add(float64(promptTokens))
	}
	if completionTokens > 0 {
		p.apiTokensCounter.WithLabelValues(tenantID, model, provider, "completion").Add(float64(completionTokens))
	}
}
