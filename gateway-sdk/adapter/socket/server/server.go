// Package server provides a gateway.Server that receives requests from the
// WP service over a Unix Domain Socket.
//
// The wire protocol matches the external/ipc server:
//
//	client ──► {"action":"...","payload":...} ──► server
//	client ◄── {"status":"ok"|"error","message":"..."} ◄── server
//
// Each connection carries exactly one request and one response, then closes.
//
// The package is split into focused files following the Single Responsibility
// Principle:
//
//	config.go   — Config, defaults, validation
//	server.go   — Server struct, New, Run, Close (lifecycle)
//	router.go   — Subscribe (handler registration)
//	handler.go  — handle (per-connection dispatch)
//	codec.go    — readRequest, writeResponse, writeError
//	retry.go    — retry policy
//	resolve.go  — user/group resolution
package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"sync/atomic"

	"github.com/testapiw/go-sdk/gateway-sdk"
)

// Server receives requests over a Unix Domain Socket and dispatches them to
// handlers registered for request actions.
type Server struct {
	cfg Config

	ln net.Listener

	subs map[string]gateway.Handler

	started atomic.Bool
	closed  atomic.Bool

	wg sync.WaitGroup
}

// New creates a Unix socket server.
func New(cfg Config) (*Server, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	// Remove a stale socket left by a previous process.
	if err := os.Remove(cfg.SocketPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("server: remove stale socket: %w", err)
	}

	ln, err := net.Listen("unix", cfg.SocketPath)
	if err != nil {
		return nil, fmt.Errorf("server: listen: %w", err)
	}

	// Change file ownership (User/Group) if specified in Config.
	if cfg.User != "" || cfg.Group != "" {
		uid, gid, err := resolveIDs(cfg.User, cfg.Group)
		if err != nil {
			_ = ln.Close()
			_ = os.Remove(cfg.SocketPath)
			return nil, fmt.Errorf("server: resolve ids: %w", err)
		}

		if err := os.Chown(cfg.SocketPath, uid, gid); err != nil {
			_ = ln.Close()
			_ = os.Remove(cfg.SocketPath)
			return nil, fmt.Errorf("server: chown: %w", err)
		}
	}

	if err := os.Chmod(cfg.SocketPath, cfg.SocketMode); err != nil {
		_ = ln.Close()
		_ = os.Remove(cfg.SocketPath)
		return nil, fmt.Errorf("server: chmod: %w", err)
	}

	return &Server{
		cfg:  cfg,
		ln:   ln,
		subs: make(map[string]gateway.Handler),
	}, nil
}

// Run starts accepting incoming connections.
//
// For every accepted connection a separate goroutine is started.
// Run returns nil when the server is closed and ctx.Err() when the context is
// cancelled.
func (s *Server) Run(ctx context.Context) error {
	if !s.started.CompareAndSwap(false, true) {
		return fmt.Errorf("server: server already started")
	}

	go func() {
		<-ctx.Done()
		_ = s.Close()
	}()

	for {
		conn, err := s.ln.Accept()
		if err != nil {
			if s.closed.Load() {
				s.wg.Wait()
				return nil
			}

			return fmt.Errorf("server: accept: %w", err)
		}

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handle(ctx, conn)
		}()
	}
}

// Close stops the server and removes the Unix socket.
//
// Close is safe to call multiple times.
func (s *Server) Close() error {
	if !s.closed.CompareAndSwap(false, true) {
		return nil
	}

	if err := s.ln.Close(); err != nil {
		return fmt.Errorf("server: close listener: %w", err)
	}

	if err := os.Remove(s.cfg.SocketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("server: remove socket: %w", err)
	}

	return nil
}
