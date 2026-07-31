package ratelimit

import (
	"context"
	"fmt"

	"golang.org/x/time/rate"
)

type Config struct {
	RequestsPerSecond float64
	Burst             int
}

func (c Config) Validate() bool {
	return c.RequestsPerSecond > 0 &&
		c.Burst > 0
}

type Limiter interface {
	Wait(context.Context) error
}

type limiter struct {
	value *rate.Limiter
}

func New(cfg Config) (Limiter, error) {
	if !cfg.Validate() {
		return nil, fmt.Errorf("invalid rate limiter configuration")
	}

	return &limiter{
		value: rate.NewLimiter(
			rate.Limit(cfg.RequestsPerSecond),
			cfg.Burst,
		),
	}, nil
}

func (l *limiter) Wait(ctx context.Context) error {
	return l.value.Wait(ctx)
}

type Unlimited struct{}

func (Unlimited) Wait(context.Context) error {
	return nil
}