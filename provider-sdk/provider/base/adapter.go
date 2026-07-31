package base

import (
	"context"

	http "github.com/testapiw/go-sdk/http-sdk/client"
	"github.com/testapiw/go-sdk/http-sdk/transport"
)

type BaseAdapter struct {
	transport *transport.Transport

	logger  Logger
	metrics Metrics
}

func New(
	transport *transport.Transport,
	logger Logger,
	metrics Metrics,
) *BaseAdapter {

	if logger == nil {
		logger = NopLogger{}
	}

	if metrics == nil {
		metrics = NopMetrics{}
	}

	return &BaseAdapter{
		transport: transport,
		logger:    logger,
		metrics:   metrics,
	}
}

func (a *BaseAdapter) Do(
	ctx context.Context,
	req http.Request,
) (*http.Response, error) {

	a.metrics.Request(req.URL)

	resp, err := a.transport.Do(ctx, req)

	if err != nil {
		a.metrics.Error(req.URL)
		return nil, err
	}

	return resp, nil
}