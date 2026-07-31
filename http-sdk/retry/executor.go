package retry

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"time"
)

type Result struct {
	StatusCode int
}

type Executor interface {
	Execute(
		context.Context,
		func(int) (Result, error),
		func(int, time.Duration),
	) (Result, error)
}

type executor struct {
	policy Policy
}

func New(policy Policy) Executor {
	return &executor{policy: policy}
}

func (e *executor) Execute(
	ctx context.Context,
	op func(int) (Result, error),
	onRetry func(int, time.Duration),
) (Result, error) {

	var (
		result Result
		err    error
	)

	for attempt := 1; attempt <= e.policy.MaxAttempts; attempt++ {

		if err := ctx.Err(); err != nil {
			return result, err
		}

		result, err = op(attempt)

		if attempt == e.policy.MaxAttempts || !e.policy.ShouldRetryResult(result, err) {
			return result, err
		}

		delay := e.delay(attempt)

		if onRetry != nil {
			onRetry(attempt, delay)
		}

		timer := time.NewTimer(delay)

		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return result, ctx.Err()

		case <-timer.C:
		}
	}

	return result, err
}

func (e *executor) delay(attempt int) time.Duration {
	delay := float64(e.policy.InitialDelay)

	for i := 1; i < attempt; i++ {
		delay *= e.policy.Multiplier
	}

	d := time.Duration(delay)

	if d > e.policy.MaxDelay {
		d = e.policy.MaxDelay
	}

	if e.policy.Jitter && d > 0 {
		d = time.Duration(rand.Int63n(int64(d) + 1))
	}

	return d
}

func retryError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var ne net.Error
	return errors.As(err, &ne) && (ne.Timeout() || ne.Temporary())
}