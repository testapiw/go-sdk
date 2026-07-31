package handlers

import (
	"context"
	"time"

	"github.com/testapiw/go-sdk/http-sdk/transport"
)

// Logger receives structured log entries for each completed operation.
type Logger interface {
	Log(LogEntry)
}

type LogEntry struct {
	Operation transport.Operation

	Duration time.Duration

	StatusCode int
	Err        error
}

type LoggerHandler struct {
	logger Logger
}

func NewLogger(logger Logger) *LoggerHandler {
	return &LoggerHandler{
		logger: logger,
	}
}

func (h *LoggerHandler) Handle(
	ctx context.Context,
	event *transport.Event,
) transport.Decision {

	// Only log after the request completes.
	if h.logger == nil || (event.Response == nil && event.Error == nil) {
		return transport.Decision{Action: transport.ActionReturn}
	}

	entry := LogEntry{
		Operation: event.Operation,
		Err:       event.Error,
	}

	if event.Response != nil {
		entry.StatusCode = event.Response.StatusCode
	}

	if startRaw, ok := event.Get(transport.ContextStartedAt); ok {
		if start, ok := startRaw.(time.Time); ok {
			entry.Duration = time.Since(start)
		}
	}

	h.logger.Log(entry)

	return transport.Decision{Action: transport.ActionReturn}
}
