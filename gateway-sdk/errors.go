package gateway

import "errors"

// Sentinel errors returned by the gateway and its adapters.
var (
	// ErrActionRequired is returned when a request is built without an action.
	ErrActionRequired = errors.New("gateway: action is required")

	// ErrNilAdapter is returned when a Gateway is created with a nil adapter.
	// NOTE: only used by the commented-out client-side Gateway.New.
	// ErrNilAdapter = errors.New("gateway: adapter is nil")

	// ErrTransport is the base error for transport-level failures.
	// Adapters wrap it to provide context about the underlying connection.
	ErrTransport = errors.New("gateway: transport error")

	// ErrRemote is returned when the remote side reports a processing error.
	ErrRemote = errors.New("gateway: remote error")
)
