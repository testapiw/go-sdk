package retry

import "time"

type Policy struct {
	MaxAttempts      int
	InitialDelay     time.Duration
	Multiplier       float64
	MaxDelay         time.Duration
	Jitter           bool

	// HTTP status codes eligible for retry.
	RetryStatusCodes []int

	// Optional custom predicate for retry decisions.
	// If specified, it has higher priority than RetryStatusCodes.
	ShouldRetry func(Result, error) bool
}

func DefaultPolicy() Policy {
	return Policy{
		MaxAttempts:  3,
		InitialDelay: 200 * time.Millisecond,
		Multiplier:   2,
		MaxDelay:     2 * time.Second,
		Jitter:       true,

		RetryStatusCodes: []int{
			500,
			502,
			503,
			504,
		},
	}
}

func (p Policy) Validate() bool {
	return p.MaxAttempts > 0 &&
		p.InitialDelay >= 0 &&
		p.Multiplier >= 1 &&
		p.MaxDelay >= p.InitialDelay
}

func (p Policy) RetryStatus(code int) bool {
	for _, v := range p.RetryStatusCodes {
		if v == code {
			return true
		}
	}
	return false
}

func (p Policy) ShouldRetryResult(r Result, err error) bool {
	if p.ShouldRetry != nil {
		return p.ShouldRetry(r, err)
	}
	return p.RetryStatus(r.StatusCode) || retryError(err)
}