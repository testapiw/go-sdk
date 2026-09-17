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

// Attempt описывает одну попытку HTTP-запроса.
//
// Журнал попыток ведёт обработчик ретраев (handlers.RetryHandler): повторы —
// его предметная область, поэтому номер попытки, признак повтора и пауза
// проставляются там. Транспорт сообщает только измеренные факты вызова
// (границы, статус, ошибка) через состояние события.
type Attempt struct {
	// Number — номер попытки, начиная с 1.
	Number int

	// StartedAt / FinishedAt — границы попытки.
	StartedAt  time.Time
	FinishedAt time.Time

	// StatusCode — HTTP-статус ответа (0, если ответа не было).
	StatusCode int

	// Error — ошибка транспорта (nil, если ответ получен).
	Error error

	// Retry — true, если после этой попытки выполнена ещё одна.
	Retry bool

	// RetryDelay — пауза перед следующей попыткой (backoff/Retry-After).
	RetryDelay time.Duration
}

// Duration возвращает длительность попытки.
func (a Attempt) Duration() time.Duration {
	if a.StartedAt.IsZero() || a.FinishedAt.IsZero() {
		return 0
	}
	return a.FinishedAt.Sub(a.StartedAt)
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

	// Delay is the pause before the next attempt. It is used with
	// ActionWait and with ActionRetry (the retry handler returns the
	// backoff / Retry-After pause this way, and the transport applies it).
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

	// ContextAttempts — журнал попыток ([]Attempt). Его ведёт обработчик
	// ретраев: только он знает, была ли попытка повторена и с какой паузой.
	ContextAttempts = "attempts"

	// ContextAttemptStartedAt / ContextAttemptFinishedAt — границы текущей
	// попытки. Их проставляет транспорт — он единственный, кто видит вызов.
	ContextAttemptStartedAt  = "attempt_started_at"
	ContextAttemptFinishedAt = "attempt_finished_at"
)
