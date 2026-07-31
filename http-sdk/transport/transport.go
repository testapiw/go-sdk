package transport

import (
	"context"
	"fmt"
	"time"

	http "github.com/testapiw/go-sdk/http-sdk/client"
)

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

type Transport struct {
	client   http.Client
	handlers []Handler
}

func New(client http.Client) (*Transport, error) {
	if client == nil {
		return nil, fmt.Errorf("http client is required")
	}

	return &Transport{
		client: client,
	}, nil
}

// Use appends a handler to the chain. Handlers run in registration order.
func (t *Transport) Use(h Handler) *Transport {
	if h != nil {
		t.handlers = append(t.handlers, h)
	}
	return t
}

// Do runs the request through a state machine. The request is static; the
// outcome of each attempt drives the next transition (retry, wait, done).
//
// The transport collects metrics (timing, attempts, status) into the returned
// Result but does not log or persist them — that is the application's job.
func (t *Transport) Do(
	ctx context.Context,
	op Operation,
	req http.Request,
) *Result {

	event := &Event{
		Operation: op,
		Request:   req,
	}

	started := time.Now()
	event.Set(ContextStartedAt, started)

	result := &Result{
		Operation: op,
		StartedAt: started,
	}

	var waited time.Duration

	state := StateReady

	for state != StateDone {
		switch state {
		case StateReady:
			// Run handlers that gate the request (rate limit, breaker).
			proceed, _ := t.advance(ctx, event, &state)
			if !proceed {
				continue
			}

			// Perform the HTTP call.
			resp, err := t.client.Do(ctx, event.Request)

			event.Response = resp
			event.Error = err
			result.Attempts++

			// Run handlers that observe the outcome (retry, breaker).
			proceed, action := t.advance(ctx, event, &state)
			if !proceed {
				if action == ActionRetry {
					result.Retries++
				}
				continue
			}

			state = StateDone

		case StateWait:
			delay, _ := event.Get(ContextWaitDelay)
			if d, ok := delay.(time.Duration); ok && d > 0 {
				select {
				case <-time.After(d):
					waited += d
				case <-ctx.Done():
					result.Error = ctx.Err()
					result.FinishedAt = time.Now()
					return result
				}
			}
			state = StateReady

		case StateDone:
			// unreachable
		}
	}

	result.FinishedAt = time.Now()
	result.Waited = waited
	result.Response = event.Response
	result.Error = event.Error

	return result
}

// advance runs the handlers and applies the resulting transition. It returns
// false when the caller must continue to the next state-machine iteration
// (retry, wait, or terminal error), and true when the caller may proceed
// (all handlers returned ActionReturn without error). When it returns false,
// the second value reports which action triggered the transition.
func (t *Transport) advance(
	ctx context.Context,
	event *Event,
	state *State,
) (bool, Action) {

	for _, h := range t.handlers {
		decision := h.Handle(ctx, event)

		switch decision.Action {
		case ActionReturn:
			if decision.Error != nil {
				event.Error = decision.Error
				*state = StateDone
				return false, ActionReturn
			}
		case ActionRetry:
			// Reset per-attempt state before the next attempt.
			event.Response = nil
			event.Error = nil
			*state = StateReady
			return false, ActionRetry
		case ActionWait:
			event.Set(ContextWaitDelay, decision.Delay)
			*state = StateWait
			return false, ActionWait
		}
	}

	return true, ActionReturn
}