package gateway

import (
	"context"
	"encoding/json"
)

// Handler processes a single request payload received from the WP service.
//
// Returning nil means the payload has been processed successfully.
// Returning an error causes the server to retry the handler according to the
// configured retry policy and, if all attempts fail, to return an error
// response to the client.
type Handler func(ctx context.Context, payload json.RawMessage) error

// Server is the transport contract for receiving requests from the WP service.
//
// It is the server-side counterpart of Adapter. Concrete implementations
// (Unix socket, HTTP/nginx, ...) hide the transport details, allowing the
// ingress to switch communication methods without changing business logic.
type Server interface {
	// Run starts accepting requests and blocks until ctx is cancelled or the
	// server is closed.
	Run(ctx context.Context) error

	// Subscribe registers a handler for the given action.
	//
	// Subscribe must be called before Run. Registering handlers after the
	// server has started is not allowed.
	Subscribe(action string, handler Handler) error
}
