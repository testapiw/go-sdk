package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/testapiw/go-sdk/gateway-sdk"
)

// readRequest reads and decodes a single JSON request from conn.
func (s *Server) readRequest(conn net.Conn) (gateway.Request, error) {
	if err := conn.SetReadDeadline(time.Now().Add(s.cfg.ReadTimeout)); err != nil {
		return gateway.Request{}, fmt.Errorf("server: set read deadline: %w", err)
	}

	// Protect against oversized requests.
	reader := io.LimitReader(conn, s.cfg.MaxRequestSize)

	var req gateway.Request

	if err := json.NewDecoder(reader).Decode(&req); err != nil {
		return gateway.Request{}, fmt.Errorf("server: decode request: %w", err)
	}

	if req.Action == "" {
		return gateway.Request{}, fmt.Errorf("server: request action is required")
	}

	return req, nil
}

// writeResponse encodes and writes a JSON response.
func (s *Server) writeResponse(conn net.Conn, resp gateway.Response) error {
	if err := conn.SetWriteDeadline(time.Now().Add(s.cfg.WriteTimeout)); err != nil {
		return fmt.Errorf("server: set write deadline: %w", err)
	}

	if err := json.NewEncoder(conn).Encode(resp); err != nil {
		return fmt.Errorf("server: encode response: %w", err)
	}

	return nil
}

// writeError sends an error response to the client.
//
// Transport errors while sending the response are intentionally ignored because
// there is nothing meaningful the server can do if the client has already gone
// away.
func (s *Server) writeError(conn net.Conn, err error) {
	_ = s.writeResponse(conn, gateway.Response{
		Status:  "error",
		Message: err.Error(),
	})
}
