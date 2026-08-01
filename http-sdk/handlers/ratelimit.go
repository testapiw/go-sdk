package handlers

import (
	"context"

	"golang.org/x/time/rate"
	"github.com/testapiw/go-sdk/http-sdk/contract"
)

// RateLimitConfig configures the rate limiter.
type RateLimitConfig struct {
	// RequestsPerSecond is the sustained request rate.
	RequestsPerSecond float64

	// Burst is the maximum burst size.
	Burst int
}

func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerSecond: 10,
		Burst:             20,
	}
}

type RateLimitHandler struct {
	limiter *rate.Limiter
}

func NewRateLimit(cfg RateLimitConfig) *RateLimitHandler {
	if cfg.RequestsPerSecond <= 0 {
		cfg.RequestsPerSecond = 10
	}
	if cfg.Burst <= 0 {
		cfg.Burst = 20
	}

	return &RateLimitHandler{
		limiter: rate.NewLimiter(
			rate.Limit(cfg.RequestsPerSecond),
			cfg.Burst,
		),
	}
}

func (h *RateLimitHandler) Handle(
	ctx context.Context,
	event *contract.Event,
) contract.Decision {

	// Rate limiting only applies before the request is sent.
	if event.Response != nil || event.Error != nil {
		return contract.Decision{Action: contract.ActionReturn}
	}

	// Wait for a token, respecting context cancellation.
	if err := h.limiter.Wait(ctx); err != nil {
		return contract.Decision{
			Action: contract.ActionReturn,
			Error:  err,
		}
	}

	return contract.Decision{Action: contract.ActionReturn}
}
