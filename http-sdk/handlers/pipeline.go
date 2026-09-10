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

	// Telemetry — an optional HTTP request metrics observer. If set,
	// a telemetry handler is added to the pipeline (last), and each request
	// sends metrics considering retries and waits. nil means telemetry is off.
	Telemetry TelemetryObserver
}

// Pipeline builds the resilience handlers in a fixed, unambiguous order:
// rate limit → breaker → retry. The breaker must run before the retry
// handler so it records a failure before the retry handler decides to
// retry. Nil config fields fall back to defaults.
//
// If cfg.Telemetry is set, the telemetry handler is added LAST:
// only then does it see the terminal state of the request (after all
// retries) and can measure the full duration including attempts.
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

	handlers := []contract.Handler{
		NewRateLimit(ratelimit),
		NewBreaker(breaker),
		NewRetry(retry),
	}

	// Telemetry — strictly last, to see the final result after retries.
	if cfg.Telemetry != nil {
		handlers = append(handlers, NewTelemetry(cfg.Telemetry))
	}

	return handlers
}
