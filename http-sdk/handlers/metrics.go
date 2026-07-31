package handlers

import (
	"context"
	"time"

	"github.com/testapiw/go-sdk/http-sdk/transport"
)

type Metrics interface {
	Request(op transport.Operation)
	Error(op transport.Operation)
	Latency(op transport.Operation, duration time.Duration)
}

type MetricsHandler struct {
	metrics Metrics
}

func NewMetrics(metrics Metrics) *MetricsHandler {
	return &MetricsHandler{
		metrics: metrics,
	}
}

func (h *MetricsHandler) Handle(
	ctx context.Context,
	event *transport.Event,
) transport.Decision {

	if h.metrics == nil {
		return transport.Decision{Action: transport.ActionReturn}
	}

	op := event.Operation

	// Before the request: no response yet.
	if event.Response == nil && event.Error == nil {
		h.metrics.Request(op)
		return transport.Decision{Action: transport.ActionReturn}
	}

	// After the request: record error and latency.
	if event.Error != nil {
		h.metrics.Error(op)
	}

	if startRaw, ok := event.Get(transport.ContextStartedAt); ok {
		if start, ok := startRaw.(time.Time); ok {
			h.metrics.Latency(op, time.Since(start))
		}
	}

	return transport.Decision{Action: transport.ActionReturn}
}