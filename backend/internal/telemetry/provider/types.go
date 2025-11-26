package provider

import "time"

// Result captures the outcome of a provider call for telemetry accounting.
type Result string

const (
	ResultSuccess  Result = "success"
	ResultError    Result = "error"
	ResultTimeout  Result = "timeout"
	ResultCanceled Result = "canceled"
)

// Sample represents a single provider call and the telemetry attributes needed for SLI windows.
type Sample struct {
	Provider     string
	ModelAlias   string
	Route        string
	Result       Result
	ErrorClass   string
	Latency      time.Duration
	InputTokens  int64
	OutputTokens int64
	Timestamp    time.Time
}

// defaultLatencyBucketsMs mirrors the Prometheus histogram buckets used for provider latency.
func defaultLatencyBucketsMs() []int64 {
	return []int64{50, 100, 200, 500, 1_000, 2_000, 5_000, 10_000, 20_000}
}
