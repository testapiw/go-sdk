package handlers

import (
	"context"
	"math/rand/v2"
	stdhttp "net/http"
	"strconv"
	"time"

	http "github.com/testapiw/go-sdk/http-sdk/client"
	"github.com/testapiw/go-sdk/http-sdk/contract"
)

// RetryPolicy configures the retry behaviour of the transport.
type RetryPolicy struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	// Retryable returns true when the event should be retried.
	Retryable func(*contract.Event) bool
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries: 3,
		BaseDelay:  200 * time.Millisecond,
		MaxDelay:   2 * time.Second,
		Retryable: func(e *contract.Event) bool {
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
	event *contract.Event,
) contract.Decision {

	// Retries are only evaluated after the request completes.
	if event.Response == nil && event.Error == nil {
		return contract.Decision{Action: contract.ActionReturn}
	}

	if !h.policy.Retryable(event) {
		return contract.Decision{Action: contract.ActionReturn}
	}

	count, _ := event.Get(contract.ContextRetryCount)
	retries, _ := count.(int)

	if retries >= h.policy.MaxRetries {
		return contract.Decision{Action: contract.ActionReturn}
	}

	retries++
	event.Set(contract.ContextRetryCount, retries)

	// For 429, honour the server's Retry-After header when present.
	// Otherwise fall back to exponential backoff.
	delay := h.backoff(retries)
	if event.Response != nil && event.Response.StatusCode == 429 {
		if ra := retryAfter(event.Response); ra > 0 {
			delay = ra
		}
	}

	return contract.Decision{
		Action: contract.ActionRetry,
		Delay:  delay,
	}
}

// backoff computes an exponential backoff with full jitter.
func (h *RetryHandler) backoff(attempt int) time.Duration {
	// 2^(attempt-1), capped to avoid overflow on large attempt counts.
	exp := uint64(1) << min(attempt-1, 62)
	base := int64(h.policy.BaseDelay) * int64(exp)
	if base > int64(h.policy.MaxDelay) {
		base = int64(h.policy.MaxDelay)
	}
	if base <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(base))
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
