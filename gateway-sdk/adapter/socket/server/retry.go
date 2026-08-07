package server

import (
	"context"
	"time"
)

// retry executes fn until it succeeds or the retry limit is reached.
//
// A RetryAttempts value of 1 means "execute once without retries".
func retry(
	ctx context.Context,
	attempts int,
	delay time.Duration,
	fn func() error,
) error {
	if attempts <= 0 {
		attempts = 1
	}

	var err error

	for i := 0; i < attempts; i++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err = fn(); err == nil {
			return nil
		}

		// No delay after the last attempt.
		if i == attempts-1 {
			break
		}

		timer := time.NewTimer(delay)

		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()

		case <-timer.C:
		}
	}

	return err
}
