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
	providermetrics "github.com/ncecere/open_model_gateway/backend/internal/telemetry/provider"
)

type Provider struct {
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *metric.MeterProvider
	promExporter   *prometheus.Exporter
	promHandler    http.Handler
	shutdownFuncs  []func(context.Context) error

	httpRequestCounter  *promreg.CounterVec
	httpRequestLatency  *promreg.HistogramVec
	apiLatencyHist      *promreg.HistogramVec
	apiTokensCounter    *promreg.CounterVec
	apiRetryCounter     *promreg.CounterVec
	rateLimiterGauge    *promreg.GaugeVec
	budgetEvalHistogram *promreg.HistogramVec
	providerLatency     *promreg.HistogramVec
	providerRequests    *promreg.CounterVec
	providerTokens      *promreg.CounterVec
	providerErrors      *promreg.CounterVec
	providerStreams     *promreg.GaugeVec
	responsesCounter    *promreg.CounterVec
	authFailuresCounter *promreg.CounterVec
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
		retryCounter := promreg.NewCounterVec(
			promreg.CounterOpts{
				Namespace: "open_model_gateway",
				Name:      "api_provider_retries_total",
				Help:      "Number of provider retries triggered by the executor.",
			},
			[]string{"tenant", "model", "provider", "reason"},
		)
		rateLimiterGauge := promreg.NewGaugeVec(
			promreg.GaugeOpts{
				Namespace: "open_model_gateway",
				Name:      "rate_limiter_parallel_inflight",
				Help:      "Current number of in-flight parallel requests per key.",
			},
			[]string{"key", "scope"},
		)
		budgetEval := promreg.NewHistogramVec(
			promreg.HistogramOpts{
				Namespace: "open_model_gateway",
				Name:      "budget_evaluation_duration_seconds",
				Help:      "Duration of budget evaluation database queries.",
				Buckets:   latencyBuckets,
			},
			[]string{"schedule"},
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
		if err := registry.Register(retryCounter); err != nil {
			return nil, err
		}
		if err := registry.Register(rateLimiterGauge); err != nil {
			return nil, err
		}
		if err := registry.Register(budgetEval); err != nil {
			return nil, err
		}
		providerLatency := promreg.NewHistogramVec(
			promreg.HistogramOpts{
				Namespace: "open_model_gateway",
				Name:      "provider_request_duration_seconds",
				Help:      "Upstream provider latency by alias/provider/route.",
				Buckets:   []float64{0.05, 0.1, 0.2, 0.5, 1, 2, 5, 10, 20},
			},
			[]string{"provider", "model_alias", "route", "result"},
		)
		providerRequests := promreg.NewCounterVec(
			promreg.CounterOpts{
				Namespace: "open_model_gateway",
				Name:      "provider_requests_total",
				Help:      "Total provider requests by outcome.",
			},
			[]string{"provider", "model_alias", "route", "result"},
		)
		providerTokens := promreg.NewCounterVec(
			promreg.CounterOpts{
				Namespace: "open_model_gateway",
				Name:      "provider_tokens_total",
				Help:      "Tokens per provider/model alias.",
			},
			[]string{"provider", "model_alias", "route", "direction"},
		)
		providerErrors := promreg.NewCounterVec(
			promreg.CounterOpts{
				Namespace: "open_model_gateway",
				Name:      "provider_upstream_errors_total",
				Help:      "Upstream provider errors by class.",
			},
			[]string{"provider", "model_alias", "route", "class"},
		)
		providerStreams := promreg.NewGaugeVec(
			promreg.GaugeOpts{
				Namespace: "open_model_gateway",
				Name:      "provider_active_streams",
				Help:      "In-flight provider streams by route.",
			},
			[]string{"provider", "model_alias", "route"},
		)
		if err := registry.Register(providerLatency); err != nil {
			return nil, err
		}
		if err := registry.Register(providerRequests); err != nil {
			return nil, err
		}
		if err := registry.Register(providerTokens); err != nil {
			return nil, err
		}
		if err := registry.Register(providerErrors); err != nil {
			return nil, err
		}
		if err := registry.Register(providerStreams); err != nil {
			return nil, err
		}
		provider.httpRequestCounter = httpRequests
		provider.httpRequestLatency = httpLatency
		provider.apiLatencyHist = apiLatency
		provider.apiTokensCounter = tokenCounter
		provider.apiRetryCounter = retryCounter
		provider.rateLimiterGauge = rateLimiterGauge
		provider.budgetEvalHistogram = budgetEval
		provider.providerLatency = providerLatency
		provider.providerRequests = providerRequests
		provider.providerTokens = providerTokens
		provider.providerErrors = providerErrors
		provider.providerStreams = providerStreams

		responsesCounter := promreg.NewCounterVec(
			promreg.CounterOpts{
				Namespace: "open_model_gateway",
				Name:      "responses_api_total",
				Help:      "Total Open Responses API requests by status and truncation.",
			},
			[]string{"model", "status", "truncation", "has_tools", "has_reasoning"},
		)
		if err := registry.Register(responsesCounter); err != nil {
			return nil, err
		}
		provider.responsesCounter = responsesCounter

		authFailures := promreg.NewCounterVec(
			promreg.CounterOpts{
				Namespace: "open_model_gateway",
				Name:      "auth_failures_total",
				Help:      "Authentication/authorization failures by type and source.",
			},
			[]string{"type", "source"},
		)
		if err := registry.Register(authFailures); err != nil {
			return nil, err
		}
		provider.authFailuresCounter = authFailures
	}

	return provider, nil
}

func (p *Provider) PrometheusHandler() http.Handler {
	if p == nil || p.promHandler == nil {
		return nil
	}
	return p.promHandler
}

// RecordProviderMetrics updates provider-level Prometheus series using a sample.
func (p *Provider) RecordProviderMetrics(sample providermetrics.Sample) {
	if p == nil || p.providerRequests == nil {
		return
	}
	provider := sample.Provider
	alias := sample.ModelAlias
	route := sample.Route
	result := string(sample.Result)
	p.providerRequests.WithLabelValues(provider, alias, route, result).Inc()
	if sample.Latency > 0 {
		p.providerLatency.WithLabelValues(provider, alias, route, result).Observe(sample.Latency.Seconds())
	}
	if sample.InputTokens > 0 {
		p.providerTokens.WithLabelValues(provider, alias, route, "input").Add(float64(sample.InputTokens))
	}
	if sample.OutputTokens > 0 {
		p.providerTokens.WithLabelValues(provider, alias, route, "output").Add(float64(sample.OutputTokens))
	}
	if sample.ErrorClass != "" {
		p.providerErrors.WithLabelValues(provider, alias, route, sample.ErrorClass).Inc()
	}
}

func (p *Provider) IncProviderStream(provider, alias, route string) {
	if p == nil || p.providerStreams == nil {
		return
	}
	p.providerStreams.WithLabelValues(provider, alias, route).Inc()
}

func (p *Provider) DecProviderStream(provider, alias, route string) {
	if p == nil || p.providerStreams == nil {
		return
	}
	p.providerStreams.WithLabelValues(provider, alias, route).Dec()
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

func (p *Provider) RecordRetry(tenantID, model, provider, reason string) {
	if p == nil || p.apiRetryCounter == nil {
		return
	}
	p.apiRetryCounter.WithLabelValues(tenantID, model, provider, reason).Inc()
}

func (p *Provider) RecordRateLimiter(key, scope string, value float64) {
	if p == nil || p.rateLimiterGauge == nil {
		return
	}
	p.rateLimiterGauge.WithLabelValues(key, scope).Set(value)
}

func (p *Provider) RateLimiterGauge() *promreg.GaugeVec {
	if p == nil {
		return nil
	}
	return p.rateLimiterGauge
}

func (p *Provider) RecordBudgetEval(schedule string, duration time.Duration) {
	if p == nil || p.budgetEvalHistogram == nil {
		return
	}
	p.budgetEvalHistogram.WithLabelValues(schedule).Observe(duration.Seconds())
}

// RecordResponsesRequest increments the Open Responses API counter.
func (p *Provider) RecordResponsesRequest(model, status, truncation string, hasTools, hasReasoning bool) {
	if p == nil || p.responsesCounter == nil {
		return
	}
	toolsLabel := "false"
	if hasTools {
		toolsLabel = "true"
	}
	reasoningLabel := "false"
	if hasReasoning {
		reasoningLabel = "true"
	}
	p.responsesCounter.WithLabelValues(model, status, truncation, toolsLabel, reasoningLabel).Inc()
}

// RecordAuthFailure increments the auth failure counter.
// failureType: "invalid_key", "expired_token", "missing_token", "rate_limited", "forbidden"
// source: "api", "admin", "user"
func (p *Provider) RecordAuthFailure(failureType, source string) {
	if p == nil || p.authFailuresCounter == nil {
		return
	}
	p.authFailuresCounter.WithLabelValues(failureType, source).Inc()
}
