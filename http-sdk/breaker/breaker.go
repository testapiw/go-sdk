package breaker

import (
	"context"
	"time"

	"github.com/sony/gobreaker"
)

type Config struct {
	Name        string
	MaxRequests uint32

	Interval time.Duration
	Timeout  time.Duration

	FailureThreshold uint32

	IsFailure func(error) bool
}

type Breaker interface {
	Execute(context.Context, func() (any, error)) (any, error)
	State() gobreaker.State
}

type breaker struct {
	cb *gobreaker.CircuitBreaker
}

func New(cfg Config) Breaker {
	threshold := cfg.FailureThreshold
	if threshold == 0 {
		threshold = 5
	}

	settings := gobreaker.Settings{
		Name:        cfg.Name,
		MaxRequests: cfg.MaxRequests,
		Interval:    cfg.Interval,
		Timeout:     cfg.Timeout,

		ReadyToTrip: func(c gobreaker.Counts) bool {
			return c.ConsecutiveFailures >= threshold
		},
	}

	if cfg.IsFailure != nil {
		settings.IsSuccessful = func(err error) bool {
			return !cfg.IsFailure(err)
		}
	}

	cb := gobreaker.NewCircuitBreaker(settings)

	return &breaker{
		cb: cb,
	}
}

func (b *breaker) Execute(ctx context.Context, fn func() (any, error)) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return b.cb.Execute(fn)
}

func (b *breaker) State() gobreaker.State {
	return b.cb.State()
}

type PassThrough struct{}

func (PassThrough) Execute(ctx context.Context, fn func() (any, error)) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return fn()
}

func (PassThrough) State() gobreaker.State {
	return gobreaker.StateClosed
}