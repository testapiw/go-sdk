package http

import (
	"context"
	stdhttp "net/http"
	"net/url"
)

type Request struct {
	Method  string
	URL     string
	Query   url.Values
	Headers stdhttp.Header
}

func (r Request) Build(ctx context.Context) (*stdhttp.Request, error) {
	u, err := url.Parse(r.URL)
	if err != nil {
		return nil, err
	}

	if len(r.Query) > 0 {
		q := u.Query()

		for key, values := range r.Query {
			for _, value := range values {
				q.Add(key, value)
			}
		}

		u.RawQuery = q.Encode()
	}

	req, err := stdhttp.NewRequestWithContext(
		ctx,
		r.Method,
		u.String(),
		nil,
	)
	if err != nil {
		return nil, err
	}

	if r.Headers != nil {
		req.Header = r.Headers.Clone()
	}

	return req, nil
}