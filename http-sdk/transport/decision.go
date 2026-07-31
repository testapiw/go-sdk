package transport

import "time"

type Action uint8

const (
	ActionReturn Action = iota
	ActionRetry
	ActionWait
)

type Decision struct {
    Action Action

    Delay time.Duration

    Error error
}