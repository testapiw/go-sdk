package handlers

import (
	"context"
	"math"
	"math/rand"
	stdhttp "net/http"
	"strconv"
	"time"

	http "github.com/testapiw/go-sdk/http-sdk/client"
	"github.com/testapiw/go-sdk/http-sdk/transport"
)

// RetryPolicy configures the retry behaviour of the transport.
type RetryPolicy struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	// Retryable returns true when the event should be retried.
	Retryable func(*transport.Event) bool
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries: 3,
		BaseDelay:  200 * time.Millisecond,
		MaxDelay:   2 * time.Second,
		Retryable: func(e *transport.Event) bool {
			if e.Error != nil {
				return true
			}
			if e.Response != nil {
				// Retry on 5xx and 429.
				return e.Response.StatusCode >= 500 ||
					e.Response.StatusCode == 429
			}
			return false
		},
	}
}

type RetryHandler struct {
	policy RetryPolicy
}

func NewRetry(policy RetryPolicy) *RetryHandler {
	if policy.MaxRetries <= 0 {
		policy.MaxRetries = 1
	}
	if policy.BaseDelay <= 0 {
		policy.BaseDelay = 200 * time.Millisecond
	}
	if policy.MaxDelay <= 0 {
		policy.MaxDelay = 2 * time.Second
	}
	if policy.Retryable == nil {
		policy.Retryable = DefaultRetryPolicy().Retryable
	}
	return &RetryHandler{policy: policy}
}

func (h *RetryHandler) Handle(
	ctx context.Context,
	event *transport.Event,
) transport.Decision {

	// Retries are only evaluated after the request completes.
	if event.Response == nil && event.Error == nil {
		return transport.Decision{Action: transport.ActionReturn}
	}

	if !h.policy.Retryable(event) {
		return transport.Decision{Action: transport.ActionReturn}
	}

	count, _ := event.Get(transport.ContextRetryCount)
	retries, _ := count.(int)

	if retries >= h.policy.MaxRetries {
		return transport.Decision{Action: transport.ActionReturn}
	}

	retries++
	event.Set(transport.ContextRetryCount, retries)

	// For 429, honour the server's Retry-After header when present.
	// Otherwise fall back to exponential backoff.
	delay := h.backoff(retries)
	if event.Response != nil && event.Response.StatusCode == 429 {
		if ra := retryAfter(event.Response); ra > 0 {
			delay = ra
		}
	}

	return transport.Decision{
		Action: transport.ActionRetry,
		Delay:  delay,
	}
}

// backoff computes an exponential backoff with full jitter.
func (h *RetryHandler) backoff(attempt int) time.Duration {
	exp := math.Pow(2, float64(attempt-1))
	base := float64(h.policy.BaseDelay) * exp
	if base > float64(h.policy.MaxDelay) {
		base = float64(h.policy.MaxDelay)
	}
	return time.Duration(rand.Int63n(int64(base)))
}

// retryAfter parses the Retry-After header. It supports both a delay in
// seconds and an absolute HTTP-date. Returns 0 when absent or invalid.
func retryAfter(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}

	v := resp.Header.Get("Retry-After")
	if v == "" {
		return 0
	}

	// Delay in seconds.
	if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}

	// Absolute HTTP-date.
	if t, err := stdhttp.ParseTime(v); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}

	return 0
}
