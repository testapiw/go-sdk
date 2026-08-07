package server

import (
	"context"
	"fmt"
	"net"

	"github.com/testapiw/go-sdk/gateway-sdk"
)

// handle processes a single client connection.
//
// The connection lifecycle is:
//
//	Request
//	   ↓
//	decode JSON
//	   ↓
//	find handler
//	   ↓
//	retry handler
//	   ↓
//	write response
//	   ↓
//	close connection
func (s *Server) handle(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	req, err := s.readRequest(conn)
	if err != nil {
		s.writeError(conn, err)
		return
	}

	handler, ok := s.subs[req.Action]
	if !ok {
		s.writeError(conn, fmt.Errorf("server: unknown action %q", req.Action))
		return
	}

	err = retry(
		ctx,
		s.cfg.RetryAttempts,
		s.cfg.RetryDelay,
		func() error {
			return handler(ctx, req.Payload)
		},
	)
	if err != nil {
		s.writeError(conn, err)
		return
	}

	_ = s.writeResponse(conn, gateway.Response{Status: "ok"})
}
