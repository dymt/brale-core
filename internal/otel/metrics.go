package otel

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
)

// Meters for brale-core core metrics.
var (
	meter     = otel.Meter("brale-core")
	noopMeter = metricnoop.NewMeterProvider().Meter("brale-core")

	// Pipeline metrics
	PipelineRoundsTotal   metric.Int64Counter
	PipelineLatencyMs     metric.Int64Histogram
	PipelineTokensTotal   metric.Int64Counter
	PipelineErrorsTotal   metric.Int64Counter
	PipelineGateDecisions metric.Int64Counter

	// LLM call metrics
	LLMCallLatencyMs metric.Int64Histogram
	LLMCallTokenIn   metric.Int64Counter
	LLMCallTokenOut  metric.Int64Counter
	LLMCallErrors    metric.Int64Counter

	// Notification metrics
	NotifyEnqueueTotal metric.Int64Counter
	NotifyDeliverTotal metric.Int64Counter
	NotifyFailTotal    metric.Int64Counter

	// Position metrics
	PositionOpenTotal  metric.Int64Counter
	PositionCloseTotal metric.Int64Counter
)

func init() {
	PipelineRoundsTotal = newInt64Counter("brale.pipeline.rounds.total", "Total decision pipeline rounds")
	PipelineLatencyMs = newInt64Histogram("brale.pipeline.latency_ms", "Pipeline round latency in ms")
	PipelineTokensTotal = newInt64Counter("brale.pipeline.tokens.total", "Total tokens consumed by pipeline rounds")
	PipelineErrorsTotal = newInt64Counter("brale.pipeline.errors.total", "Total pipeline round errors")
	PipelineGateDecisions = newInt64Counter("brale.pipeline.gate.decisions", "Gate decisions by action")

	LLMCallLatencyMs = newInt64Histogram("brale.llm.call.latency_ms", "Individual LLM call latency")
	LLMCallTokenIn = newInt64Counter("brale.llm.call.token_in", "LLM input tokens")
	LLMCallTokenOut = newInt64Counter("brale.llm.call.token_out", "LLM output tokens")
	LLMCallErrors = newInt64Counter("brale.llm.call.errors", "LLM call errors")

	NotifyEnqueueTotal = newInt64Counter("brale.notify.enqueue.total", "Notifications enqueued")
	NotifyDeliverTotal = newInt64Counter("brale.notify.deliver.total", "Notifications delivered")
	NotifyFailTotal = newInt64Counter("brale.notify.fail.total", "Notification delivery failures")

	PositionOpenTotal = newInt64Counter("brale.position.open.total", "Positions opened")
	PositionCloseTotal = newInt64Counter("brale.position.close.total", "Positions closed")
}

func newInt64Counter(name, description string) metric.Int64Counter {
	counter, err := meter.Int64Counter(name, metric.WithDescription(description))
	if err == nil {
		return counter
	}
	counter, _ = noopMeter.Int64Counter(name, metric.WithDescription(description))
	return counter
}

func newInt64Histogram(name, description string) metric.Int64Histogram {
	histogram, err := meter.Int64Histogram(name, metric.WithDescription(description))
	if err == nil {
		return histogram
	}
	histogram, _ = noopMeter.Int64Histogram(name, metric.WithDescription(description))
	return histogram
}
