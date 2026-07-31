package base

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/sony/gobreaker"
	"github.com/testapiw/go-sdk/http-sdk/breaker"
	httpx "github.com/testapiw/go-sdk/http-sdk/client"
	"github.com/testapiw/go-sdk/http-sdk/ratelimit"
	"github.com/testapiw/go-sdk/http-sdk/retry"
	"github.com/testapiw/go-sdk/provider-sdk/provider/contract"
)

type BaseAdapter struct {
	client  httpx.Client
	retry   retry.Executor
	breaker breaker.Breaker
	limiter ratelimit.Limiter
	logger  Logger
	metrics Metrics
}

func New(client httpx.Client, retryer retry.Executor, cb breaker.Breaker, limiter ratelimit.Limiter, logger Logger, metrics Metrics) *BaseAdapter {
	if cb == nil {
		cb = breaker.PassThrough{}
	}
	if limiter == nil {
		limiter = ratelimit.Unlimited{}
	}
	if logger == nil {
		logger = NopLogger{}
	}
	if metrics == nil {
		metrics = NopMetrics{}
	}
	return &BaseAdapter{client: client, retry: retryer, breaker: cb, limiter: limiter, logger: logger, metrics: metrics}
}
func (a *BaseAdapter) Do(ctx context.Context, provider, endpoint, requestID string, req httpx.Request) (*httpx.Response, error) {
	start := time.Now()
	retries := 0
	status := 0
	var finalErr error
	a.metrics.Request(endpoint)
	defer func() {
		d := time.Since(start)
		a.metrics.Latency(endpoint, d)
		a.logger.Log(LogEntry{Endpoint: endpoint, RequestID: requestID, Duration: d, StatusCode: status, RetryCount: retries, Err: finalErr})
	}()
	if err := a.limiter.Wait(ctx); err != nil {
		finalErr = a.classify(err, provider, endpoint, requestID, 0)
		return nil, finalErr
	}
	var resp *httpx.Response
	operation := func() (any, error) {
		result, err := a.retry.Execute(ctx, func(int) (retry.Result, error) {
			var e error
			resp, e = a.client.Do(ctx, req)
			if resp != nil {
				status = resp.StatusCode
			}
			return retry.Result{StatusCode: status}, e
		}, func(int, time.Duration) { retries++; a.metrics.Retry(endpoint) })
		status = result.StatusCode
		if err != nil {
			return nil, err
		}
		if isFailure(status) {
			return nil, statusError(status)
		}
		return resp, nil
	}
	value, err := a.breaker.Execute(ctx, operation)
	if err != nil {
		a.metrics.Error(endpoint)
		mapped := a.classify(err, provider, endpoint, requestID, status)
		finalErr = mapped
		if errors.Is(mapped, contract.ErrTimeout) {
			a.metrics.Timeout(endpoint)
		}
		if errors.Is(mapped, contract.ErrRateLimit) {
			a.metrics.RateLimit(endpoint)
		}
		return nil, mapped
	}
	return value.(*httpx.Response), nil
}

type httpStatusError int

func (e httpStatusError) Error() string { return fmt.Sprintf("provider returned HTTP status %d", e) }
func statusError(s int) error           { return httpStatusError(s) }
func isFailure(s int) bool              { return s < 200 || s >= 300 }
func (a *BaseAdapter) classify(err error, provider, op, id string, status int) error {
	kind := contract.ErrInternal
	var hs httpStatusError
	if errors.As(err, &hs) {
		status = int(hs)
	}
	switch {
	case status == 401 || status == 403:
		kind = contract.ErrUnauthorized
	case status == 429:
		kind = contract.ErrRateLimit
	case status == 500 || status == 502 || status == 503 || status == 504:
		kind = contract.ErrUnavailable
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled):
		kind = contract.ErrTimeout
	case errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests):
		kind = contract.ErrUnavailable
	default:
		var ne net.Error
		if errors.As(err, &ne) {
			kind = contract.ErrNetwork
		}
	}
	return contract.NewError(kind, provider, op, id, err)
}
