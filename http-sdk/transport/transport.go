package transport

import (
	"context"
	"fmt"

	http "github.com/testapiw/go-sdk/http-sdk/client"
)

type Transport struct {
	client http.Client
}

func New(client http.Client) (*Transport, error) {
	if client == nil {
		return nil, fmt.Errorf("http client is required")
	}

	return &Transport{
		client: client,
	}, nil
}

func (t *Transport) Do(
	ctx context.Context,
	req http.Request,
) (*http.Response, error) {
	return t.client.Do(ctx, req)
}