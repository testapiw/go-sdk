package base

import (
	"context"
	"encoding/json"
	"fmt"

	httpx "github.com/testapiw/go-sdk/http-sdk/client"
	httpcontract "github.com/testapiw/go-sdk/http-sdk/contract"
	"github.com/testapiw/go-sdk/http-sdk/transport"

	"github.com/testapiw/go-sdk/provider-sdk/provider/contract"
)

// BaseAdapter wraps a transport and exposes the request lifecycle to
// providers. It is intentionally free of logging and metrics — the transport
// collects metrics into the returned Result, and the application decides how
// to log or persist them.
type BaseAdapter struct {
	transport *transport.Transport

	// onResult is invoked after every request with the transport Result.
	// It is optional and never blocks the request lifecycle.
	onResult func(*transport.Result)
}

// New builds a base adapter for the given provider: it validates the config,
// creates the HTTP client and the resilience pipeline (rate limit → breaker
// → retry) from cfg. The name identifies the breaker (e.g. the provider
// name). The optional validate hook lets a provider check its own config
// before the adapter is built; errors are wrapped as ErrConfiguration.
func New(
	name string,
	cfg Config,
	doer httpx.Doer,
	validate func() error,
) (*BaseAdapter, error) {

	if validate != nil {
		if err := validate(); err != nil {
			return nil, contract.NewError(
				contract.ErrConfiguration,
				name,
				"new",
				"",
				err,
			)
		}
	}

	client := httpx.NewClient(doer, cfg.Timeout)

	t, err := transport.New(name, client, cfg.Config)
	if err != nil {
		return nil, contract.NewError(
			contract.ErrConfiguration,
			name,
			"new",
			"",
			err,
		)
	}

	return &BaseAdapter{transport: t}, nil
}

// OnResult registers a callback invoked after every request with the
// transport Result (response, error, timing, attempts). The callback runs
// synchronously after the request completes; keep it fast or hand off to a
// queue. Passing nil disables it.
func (a *BaseAdapter) OnResult(fn func(*transport.Result)) {
	a.onResult = fn
}

// Do runs the request through the transport state machine and returns the
// Result, which carries both the response/error and the collected metrics.
func (a *BaseAdapter) Do(
	ctx context.Context,
	op httpcontract.Operation,
	req httpx.Request,
) *transport.Result {

	result := a.transport.Do(ctx, op, req)

	// Notify the application about the result (logging/metrics). This runs
	// after the request completes and does not affect the returned data.
	if a.onResult != nil {
		a.onResult(result)
	}

	return result
}

// GetJSON runs a GET request and unmarshals the response body into target.
// It returns the transport error, or a JSON unmarshal error on a malformed
// body.
func (a *BaseAdapter) GetJSON(
	ctx context.Context,
	op httpcontract.Operation,
	req httpx.Request,
	target any,
) error {

	result := a.Do(ctx, op, req)
	if result.Error != nil {
		return result.Error
	}

	// Перевіряємо HTTP статус. Якщо сервер повернув помилку (4xx/5xx),
	// не намагаємось розпарсити тіло як ціни — повертаємо зрозумілу помилку.
	if result.Response != nil && !result.Response.Success() {
		return contract.NewError(
			contract.ErrInvalidResponse,
			op.Provider,
			op.Endpoint,
			op.Name,
			fmt.Errorf("http status %d: %s", result.Response.StatusCode, truncateBody(result.Response.Body)),
		)
	}

	if err := json.Unmarshal(result.Response.Body, target); err != nil {
		return contract.NewError(
			contract.ErrInvalidResponse,
			op.Provider,
			op.Endpoint,
			op.Name,
			err,
		)
	}

	return nil
}

// truncateBody обрізає тіло відповіді до 200 символів для логування.
func truncateBody(b []byte) string {
	if len(b) > 200 {
		return string(b[:200]) + "..."
	}
	return string(b)
}