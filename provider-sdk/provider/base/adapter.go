package base

import (
	"context"

	http "github.com/testapiw/go-sdk/http-sdk/client"
	"github.com/testapiw/go-sdk/http-sdk/transport"
)

// BaseAdapter wraps a transport and exposes the request lifecycle to
// providers. It is intentionally free of logging and metrics — the transport
// collects metrics into the returned Result, and the application decides how
// to log or persist them.
type BaseAdapter struct {
	transport *transport.Transport
}

// New creates a base adapter around the given transport.
func New(transport *transport.Transport) *BaseAdapter {
	return &BaseAdapter{transport: transport}
}

// Do runs the request through the transport state machine and returns the
// Result, which carries both the response/error and the collected metrics.
func (a *BaseAdapter) Do(
	ctx context.Context,
	op transport.Operation,
	req http.Request,
) *transport.Result {

	return a.transport.Do(ctx, op, req)
}