package base

import (
	"time"

	"github.com/testapiw/go-sdk/http-sdk/handlers"
)

// Config holds the transport-level settings shared by all providers.
// Provider-specific configs embed this type.
type Config struct {
	// Timeout is the per-request timeout.
	Timeout time.Duration

	// Resilience handler configuration. Nil fields fall back to defaults.
	handlers.Config
}