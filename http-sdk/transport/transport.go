package transport

import (
	"context"
	"fmt"
	"time"

	http "github.com/testapiw/go-sdk/http-sdk/client"
	"github.com/testapiw/go-sdk/http-sdk/contract"
	"github.com/testapiw/go-sdk/http-sdk/handlers"
)

type Transport struct {
	client   http.Client
	handlers []contract.Handler
}

// New creates a transport around the given client and builds the resilience
// pipeline (rate limit → breaker → retry) from cfg. The name identifies the
// breaker (e.g. the provider name). Nil cfg fields fall back to defaults, so
// the transport is protected out of the box.
func New(name string, client http.Client, cfg handlers.Config) (*Transport, error) {
	if client == nil {
		return nil, fmt.Errorf("http client is required")
	}

	cfg.Name = name

	return &Transport{
		client:   client,
		handlers: handlers.Pipeline(cfg),
	}, nil
}

// Do runs the request through a state machine. The request is static; the
// outcome of each attempt drives the next transition (retry, wait, done).
//
// The transport collects metrics (timing, attempts, status) into the returned
// Result but does not log or persist them — that is the application's job.
func (t *Transport) Do(
	ctx context.Context,
	op contract.Operation,
	req http.Request,
) *Result {

	event := &contract.Event{
		Operation: op,
		Request:   req,
	}

	started := time.Now()
	event.Set(contract.ContextStartedAt, started)

	result := &Result{
		Operation: op,
		StartedAt: started,
	}

	var waited time.Duration

	state := contract.StateReady

	for state != contract.StateDone {
		switch state {
		case contract.StateReady:
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
				if action == contract.ActionRetry {
					result.Retries++
				}
				continue
			}

			state = contract.StateDone

		case contract.StateWait:
			delay, _ := event.Get(contract.ContextWaitDelay)
			if d, ok := delay.(time.Duration); ok && d > 0 {
				timer := time.NewTimer(d)
				select {
				case <-timer.C:
					waited += d
				case <-ctx.Done():
					timer.Stop()
					result.Error = ctx.Err()
					result.FinishedAt = time.Now()
					return result
				}
			}
			state = contract.StateReady

		case contract.StateDone:
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
	event *contract.Event,
	state *contract.State,
) (bool, contract.Action) {

	for _, h := range t.handlers {
		decision := h.Handle(ctx, event)

		switch decision.Action {
		case contract.ActionReturn:
			if decision.Error != nil {
				event.Error = decision.Error
				*state = contract.StateDone
				return false, contract.ActionReturn
			}
		case contract.ActionRetry:
			// Reset per-attempt state before the next attempt.
			event.Response = nil
			event.Error = nil
			*state = contract.StateReady
			return false, contract.ActionRetry
		case contract.ActionWait:
			event.Set(contract.ContextWaitDelay, decision.Delay)
			*state = contract.StateWait
			return false, contract.ActionWait
		}
	}

	return true, contract.ActionReturn
}