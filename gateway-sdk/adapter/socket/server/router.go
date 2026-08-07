package server

import (
	"fmt"

	"github.com/testapiw/go-sdk/gateway-sdk"
)

// Subscribe registers a handler for the specified action.
//
// Each action may have only one handler.
//
// Subscribe must be called before Run. Registering handlers after the server
// has started is not allowed.
func (s *Server) Subscribe(action string, handler gateway.Handler) error {
	if s.started.Load() {
		return fmt.Errorf("server: cannot register handler after server start")
	}

	if action == "" {
		return fmt.Errorf("server: action is required")
	}

	if handler == nil {
		return fmt.Errorf("server: handler is nil")
	}

	if _, exists := s.subs[action]; exists {
		return fmt.Errorf("server: handler already registered for action %q", action)
	}

	s.subs[action] = handler

	return nil
}
