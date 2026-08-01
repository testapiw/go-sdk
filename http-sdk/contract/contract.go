// Package contract defines the shared types between the transport and its
// handlers. It lives in its own package so that transport and handlers can
// depend on each other without an import cycle.
package contract

import (
	"context"
	"time"

	http "github.com/testapiw/go-sdk/http-sdk/client"
)

// Operation identifies a request.
type Operation struct {
	Provider string
	Name     string
	Endpoint string
	Method   string
}

// Event carries the request lifecycle state observed by handlers.
type Event struct {
	Operation Operation

	Request  http.Request
	Response *http.Response
	Error    error

	State map[string]any
}

func (e *Event) Set(key string, value any) {
	if e.State == nil {
		e.State = make(map[string]any)
	}

	e.State[key] = value
}

func (e *Event) Get(key string) (any, bool) {
	if e.State == nil {
		return nil, false
	}

	v, ok := e.State[key]
	return v, ok
}

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

// Handler observes an event and returns a Decision that drives the
// request state machine. Handlers are invoked on every state transition.
type Handler interface {
	Handle(context.Context, *Event) Decision
}

// HandlerFunc adapts a function to the Handler interface.
type HandlerFunc func(context.Context, *Event) Decision

func (f HandlerFunc) Handle(ctx context.Context, e *Event) Decision {
	return f(ctx, e)
}

// Event state keys.
const (
	ContextStartedAt  = "started_at"
	ContextRetryCount = "retry_count"
	ContextWaitDelay  = "wait_delay"
)
