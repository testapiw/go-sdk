package handlers

import (
	"context"
	"errors"
	"time"

	"github.com/sony/gobreaker"
	"github.com/testapiw/go-sdk/http-sdk/contract"
)

// BreakerConfig configures the circuit breaker behaviour.
type BreakerConfig struct {
	// Name identifies the breaker (e.g. provider name).
	Name string

	// MaxRequests is the number of requests allowed in the half-open state.
	MaxRequests uint32

	// Interval is the cyclic period of the closed state.
	Interval time.Duration

	// Timeout is the duration the breaker stays open before half-open.
	Timeout time.Duration

	// ReadyToTrip is called when a request fails in the closed state.
	ReadyToTrip func(counts gobreaker.Counts) bool
}

func DefaultBreakerConfig(name string) BreakerConfig {
	return BreakerConfig{
		Name:        name,
		MaxRequests: 1,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
	}
}

type BreakerHandler struct {
	breaker *gobreaker.CircuitBreaker
}

func NewBreaker(cfg BreakerConfig) *BreakerHandler {
	if cfg.Name == "" {
		cfg.Name = "default"
	}
	if cfg.MaxRequests == 0 {
		cfg.MaxRequests = 1
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.ReadyToTrip == nil {
		cfg.ReadyToTrip = DefaultBreakerConfig("").ReadyToTrip
	}

	st := gobreaker.Settings{
		Name:        cfg.Name,
		MaxRequests: cfg.MaxRequests,
		Interval:    cfg.Interval,
		Timeout:     cfg.Timeout,
		ReadyToTrip: cfg.ReadyToTrip,
	}

	return &BreakerHandler{
		breaker: gobreaker.NewCircuitBreaker(st),
	}
}

func (h *BreakerHandler) Handle(
	ctx context.Context,
	event *contract.Event,
) contract.Decision {

	// Before the request: reject if the breaker is open.
	if event.Response == nil && event.Error == nil {
		if h.breaker.State() == gobreaker.StateOpen {
			return contract.Decision{
				Action: contract.ActionReturn,
				Error:  gobreaker.ErrOpenState,
			}
		}
		return contract.Decision{Action: contract.ActionReturn}
	}

	// After the request: record the outcome through the breaker.
	// The wrapped function performs no HTTP call — it only reports the
	// already-completed outcome so the breaker can update its state.
	_, _ = h.breaker.Execute(func() (interface{}, error) {
		if event.Error != nil {
			return nil, event.Error
		}
		if event.Response != nil && event.Response.StatusCode >= 500 {
			return nil, errServerError
		}
		return event.Response, nil
	})

	return contract.Decision{Action: contract.ActionReturn}
}

// errServerError marks a 5xx response as a breaker failure.
var errServerError = errors.New("server error")
