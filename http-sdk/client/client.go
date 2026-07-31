package http

import (
	"context"
	"fmt"
	"io"
	stdhttp "net/http"
	"time"
)

const defaultMaxBody = 10 << 20 // 10 MB

type Doer interface {
	Do(*stdhttp.Request) (*stdhttp.Response, error)
}

type Client interface {
	Do(context.Context, Request) (*Response, error)
}

type client struct {
	doer    Doer
	timeout time.Duration
	maxBody int64
}

func NewClient(doer Doer, timeout time.Duration) Client {
	if doer == nil {
		doer = stdhttp.DefaultClient
	}

	return &client{
		doer:    doer,
		timeout: timeout,
		maxBody: defaultMaxBody,
	}
}

func (c *client) Do(ctx context.Context, req Request) (*Response, error) {
	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	httpRequest, err := req.Build(ctx)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	httpResponse, err := c.doer.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer httpResponse.Body.Close()

	body, err := io.ReadAll(
		io.LimitReader(httpResponse.Body, c.maxBody+1),
	)
	if err != nil {
		return nil, err
	}

	if int64(len(body)) > c.maxBody {
		return nil, fmt.Errorf("response body exceeds %d bytes", c.maxBody)
	}

	return &Response{
		StatusCode: httpResponse.StatusCode,
		Header:     httpResponse.Header.Clone(),
		Body:       body,
	}, nil
}