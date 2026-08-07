package server

import (
	"fmt"
	"os"
	"time"
)

// Default configuration values.
const (
	// DefaultSocketMode defines the permissions applied to the Unix socket.
	DefaultSocketMode = 0o660

	// DefaultReadTimeout limits a single read operation.
	DefaultReadTimeout = 5 * time.Second

	// DefaultWriteTimeout limits a single write operation.
	DefaultWriteTimeout = 5 * time.Second

	// DefaultMaxRequestSize is the maximum accepted JSON request size.
	DefaultMaxRequestSize int64 = 5 << 20 // 5 MB

	// DefaultRetryAttempts defines how many times a handler is retried
	// before an error response is returned.
	DefaultRetryAttempts = 3

	// DefaultRetryDelay is the pause between retry attempts.
	DefaultRetryDelay = 100 * time.Millisecond
)

// Config holds the Unix socket server parameters.
type Config struct {
	// SocketPath is the filesystem path of the Unix Domain Socket.
	// Required.
	SocketPath string

	// SocketMode defines filesystem permissions for the socket file.
	SocketMode os.FileMode

	// User defines the target username or UID string (e.g. "www-data" or "1000")
	// for changing socket ownership via chown. Optional.
	User string

	// Group defines the target group name or GID string (e.g. "www-data" or "1000")
	// for changing socket group ownership via chown. Optional.
	Group string

	// ReadTimeout limits a single request read.
	ReadTimeout time.Duration

	// WriteTimeout limits a single response write.
	WriteTimeout time.Duration

	// MaxRequestSize limits the maximum JSON document size.
	MaxRequestSize int64

	// RetryAttempts specifies how many times a handler is retried
	// when it returns an error.
	RetryAttempts int

	// RetryDelay specifies the delay between retry attempts.
	RetryDelay time.Duration
}

// validate applies default values and validates the configuration.
func (c *Config) validate() error {
	if c.SocketPath == "" {
		return fmt.Errorf("server: socket path is required")
	}

	if c.SocketMode == 0 {
		c.SocketMode = DefaultSocketMode
	}

	if c.ReadTimeout <= 0 {
		c.ReadTimeout = DefaultReadTimeout
	}

	if c.WriteTimeout <= 0 {
		c.WriteTimeout = DefaultWriteTimeout
	}

	if c.MaxRequestSize <= 0 {
		c.MaxRequestSize = DefaultMaxRequestSize
	}

	if c.RetryAttempts <= 0 {
		c.RetryAttempts = DefaultRetryAttempts
	}

	if c.RetryDelay <= 0 {
		c.RetryDelay = DefaultRetryDelay
	}

	return nil
}
