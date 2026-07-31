package transport

import "time"

// State is the current state of the request state machine.
type State uint8

const (
	// StateReady means the request is ready to be sent (or re-sent).
	StateReady State = iota
	// StateWait means the machine is waiting before the next attempt.
	StateWait
	// StateDone means the machine has finished (success or terminal error).
	StateDone
)

// Action is the transition requested by a handler.
type Action uint8

const (
	// ActionReturn continues to the next handler or completes the request.
	ActionReturn Action = iota
	// ActionRetry requests another attempt of the request.
	ActionRetry
	// ActionWait requests a delay before the next attempt.
	ActionWait
)

// Decision is the transition requested by a handler after observing an event.
type Decision struct {
	Action Action

	// Delay is used with ActionWait.
	Delay time.Duration

	// Error short-circuits the machine to StateDone when set with ActionReturn.
	Error error
}
