package handlers

import (
	"github.com/testapiw/go-sdk/http-sdk/contract"
)

// Config aggregates the optional resilience handler settings. Nil fields
// fall back to the package defaults, so a zero Config yields a fully
// protected pipeline.
type Config struct {
	// Name identifies the breaker (e.g. provider name).
	Name string

	Retry     *RetryPolicy
	RateLimit *RateLimitConfig
	Breaker   *BreakerConfig
}

// Pipeline builds the resilience handlers in a fixed, unambiguous order:
// rate limit → breaker → retry. The breaker must run before the retry
// handler so it records a failure before the retry handler decides to
// retry. Nil config fields fall back to defaults.
func Pipeline(cfg Config) []contract.Handler {
	retry := DefaultRetryPolicy()
	if cfg.Retry != nil {
		retry = *cfg.Retry
	}

	ratelimit := DefaultRateLimitConfig()
	if cfg.RateLimit != nil {
		ratelimit = *cfg.RateLimit
	}

	breaker := DefaultBreakerConfig(cfg.Name)
	if cfg.Breaker != nil {
		breaker = *cfg.Breaker
	}

	return []contract.Handler{
		NewRateLimit(ratelimit),
		NewBreaker(breaker),
		NewRetry(retry),
	}
}
