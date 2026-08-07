// Package gateway provides an abstract transport layer for communication
// between the WP service and the collector.
//
// The concrete transport (Unix socket, HTTP/nginx, ...) is hidden behind the
// Adapter interface. This allows switching the communication method without
// changing the business logic:
//
//	WP ──► Gateway ──► Adapter ──► (socket | nginx | ...) ──► Collector
//
// The Gateway is used on the Go side. It owns no business logic; it only
// forwards a Request to the configured Adapter and returns the Response.
package gateway

// NOTE: the client-side abstraction (Adapter, Gateway, New, Call) is currently
// unused — the collector only receives signals from WP (server side). It is
// kept commented out for future use when a Go service needs to send data to
// the collector. Uncomment when needed.
//
// import "context"
//
// // Adapter is the transport contract for sending a single request to the
// // collector and receiving a single response.
// //
// // Implementations must be safe for concurrent use.
// type Adapter interface {
// 	// Call sends req over the underlying transport and returns the response.
// 	//
// 	// Call must honour ctx cancellation and return ctx.Err() when the context
// 	// is cancelled before the round-trip completes.
// 	Call(ctx context.Context, req Request) (Response, error)
// }
//
// // Gateway is a thin wrapper around an Adapter.
// //
// // It is the entry point used by the WP-facing Go code. The concrete adapter
// // is selected at construction time, which is what makes switching between
// // socket and nginx transports a configuration concern rather than a code
// // change.
// type Gateway struct {
// 	adapter Adapter
// }
//
// // New creates a Gateway backed by the given adapter.
// func New(adapter Adapter) (*Gateway, error) {
// 	if adapter == nil {
// 		return nil, ErrNilAdapter
// 	}
//
// 	return &Gateway{adapter: adapter}, nil
// }
//
// // Call forwards req to the underlying adapter.
// func (g *Gateway) Call(ctx context.Context, req Request) (Response, error) {
// 	return g.adapter.Call(ctx, req)
// }