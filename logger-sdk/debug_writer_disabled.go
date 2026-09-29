//go:build !debug

package logger

import "time"

type debugWriter interface {
	write(time.Time, string, string)
	close() error
}

func newDebugUDPWriter(string) (debugWriter, error) { return nil, nil }
