package handlers

import (
	"context"
	"errors"
	stdhttp "net/http"
	"testing"
	"time"

	http "github.com/testapiw/go-sdk/http-sdk/client"
	"github.com/testapiw/go-sdk/http-sdk/contract"
)

// retryable — политика из дефолтов с быстрыми задержками для тестов.
func retryable(maxRetries int, base, max time.Duration) RetryPolicy {
	policy := DefaultRetryPolicy()
	policy.MaxRetries = maxRetries
	policy.BaseDelay = base
	policy.MaxDelay = max
	return policy
}

// newResponse собирает ответ SDK с заданным статусом и заголовками.
func newResponse(status int, header stdhttp.Header) *http.Response {
	return &http.Response{StatusCode: status, Header: header}
}

func TestRetryHandlerDoesNotRecordBeforeCall(t *testing.T) {
	handler := NewRetry(retryable(3, time.Millisecond, 2*time.Millisecond))

	// Запрос ещё не отправлен: ответа и ошибки нет.
	event := &contract.Event{Operation: contract.Operation{Provider: "coingecko"}}

	decision := handler.Handle(context.Background(), event)

	if decision.Action != contract.ActionReturn {
		t.Fatalf("action = %v, want ActionReturn", decision.Action)
	}
	if got := Attempts(event); len(got) != 0 {
		t.Fatalf("journal = %d records, want 0", len(got))
	}
}

func TestRetryHandlerStopsOnSuccess(t *testing.T) {
	handler := NewRetry(retryable(3, time.Millisecond, 2*time.Millisecond))

	event := &contract.Event{Response: newResponse(200, nil)}
	decision := handler.Handle(context.Background(), event)

	if decision.Action != contract.ActionReturn {
		t.Fatalf("action = %v, want ActionReturn", decision.Action)
	}
	if decision.Delay != 0 {
		t.Fatalf("delay = %v, want 0", decision.Delay)
	}

	journal := Attempts(event)
	if len(journal) != 1 {
		t.Fatalf("journal = %d records, want 1", len(journal))
	}
	if journal[0].Retry || journal[0].StatusCode != 200 || journal[0].Number != 1 {
		t.Fatalf("record = %+v, want number=1 status=200 retry=false", journal[0])
	}
}

func TestRetryHandlerRetriesServerError(t *testing.T) {
	base := 10 * time.Millisecond
	handler := NewRetry(retryable(3, base, 40*time.Millisecond))

	event := &contract.Event{Response: newResponse(503, nil)}
	decision := handler.Handle(context.Background(), event)

	if decision.Action != contract.ActionRetry {
		t.Fatalf("action = %v, want ActionRetry", decision.Action)
	}
	if decision.Delay <= 0 || decision.Delay > base {
		t.Fatalf("delay = %v, want (0, %v]", decision.Delay, base)
	}

	journal := Attempts(event)
	if len(journal) != 1 {
		t.Fatalf("journal = %d records, want 1", len(journal))
	}
	if !journal[0].Retry || journal[0].RetryDelay != decision.Delay {
		t.Fatalf("record = %+v, want retry=true delay=%v", journal[0], decision.Delay)
	}
	if Retries(event) != 1 {
		t.Fatalf("retries = %d, want 1", Retries(event))
	}
}

func TestRetryHandlerRetriesTransportError(t *testing.T) {
	handler := NewRetry(retryable(3, time.Millisecond, 2*time.Millisecond))

	errBoom := errors.New("connection reset")
	event := &contract.Event{Error: errBoom}
	decision := handler.Handle(context.Background(), event)

	if decision.Action != contract.ActionRetry {
		t.Fatalf("action = %v, want ActionRetry", decision.Action)
	}
	if journal := Attempts(event); len(journal) != 1 || !errors.Is(journal[0].Error, errBoom) {
		t.Fatalf("journal = %+v, want one record with the transport error", journal)
	}
}

func TestRetryHandlerHonoursRetryAfter(t *testing.T) {
	handler := NewRetry(retryable(3, time.Millisecond, 2*time.Millisecond))

	header := stdhttp.Header{}
	header.Set("Retry-After", "2")

	event := &contract.Event{Response: newResponse(429, header)}
	decision := handler.Handle(context.Background(), event)

	if decision.Action != contract.ActionRetry {
		t.Fatalf("action = %v, want ActionRetry", decision.Action)
	}
	if decision.Delay != 2*time.Second {
		t.Fatalf("delay = %v, want 2s (Retry-After)", decision.Delay)
	}
}
func TestRetryHandlerDoesNotRetryClientError(t *testing.T) {
	handler := NewRetry(retryable(3, time.Millisecond, 2*time.Millisecond))

	event := &contract.Event{Response: newResponse(404, nil)}
	decision := handler.Handle(context.Background(), event)

	if decision.Action != contract.ActionReturn {
		t.Fatalf("action = %v, want ActionReturn", decision.Action)
	}
	if journal := Attempts(event); len(journal) != 1 || journal[0].Retry {
		t.Fatalf("journal = %+v, want one record without retry", journal)
	}
}

func TestRetryHandlerExhaustsMaxRetries(t *testing.T) {
	handler := NewRetry(retryable(2, time.Millisecond, 2*time.Millisecond))

	event := &contract.Event{}
	actions := make([]contract.Action, 0, 3)

	// Три завершённые попытки: две с повтором, третья — окончательная.
	for i := 0; i < 3; i++ {
		event.Response = newResponse(500, nil)
		decision := handler.Handle(context.Background(), event)
		actions = append(actions, decision.Action)
		event.Response = nil
	}

	want := []contract.Action{
		contract.ActionRetry,
		contract.ActionRetry,
		contract.ActionReturn,
	}
	for i, action := range actions {
		if action != want[i] {
			t.Fatalf("attempt %d: action = %v, want %v", i+1, action, want[i])
		}
	}

	journal := Attempts(event)
	if len(journal) != 3 {
		t.Fatalf("journal = %d records, want 3", len(journal))
	}
	for i, record := range journal {
		if record.Number != i+1 {
			t.Fatalf("record %d: number = %d, want %d", i, record.Number, i+1)
		}
	}
	if Retries(event) != 2 {
		t.Fatalf("retries = %d, want 2", Retries(event))
	}
}

func TestRetryHandlerNormalisesPolicy(t *testing.T) {
	handler := NewRetry(RetryPolicy{})

	if handler.policy.MaxRetries != 1 {
		t.Fatalf("MaxRetries = %d, want 1", handler.policy.MaxRetries)
	}
	if handler.policy.BaseDelay != 200*time.Millisecond {
		t.Fatalf("BaseDelay = %v, want 200ms", handler.policy.BaseDelay)
	}
	if handler.policy.MaxDelay != 2*time.Second {
		t.Fatalf("MaxDelay = %v, want 2s", handler.policy.MaxDelay)
	}
	if handler.policy.Retryable == nil {
		t.Fatal("Retryable = nil, want default policy")
	}
}

func TestRetryHandlerBackoffBounds(t *testing.T) {
	base := 100 * time.Millisecond
	maxDelay := 500 * time.Millisecond
	handler := NewRetry(retryable(10, base, maxDelay))

	for attempt := 1; attempt <= 10; attempt++ {
		delay := handler.backoff(attempt)
		if delay < 0 {
			t.Fatalf("attempt %d: delay = %v, want >= 0", attempt, delay)
		}
		if delay > maxDelay {
			t.Fatalf("attempt %d: delay = %v, want <= %v", attempt, delay, maxDelay)
		}
	}
}

func TestRetryAfterParsing(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   time.Duration
	}{
		{name: "без заголовка", header: "", want: 0},
		{name: "секунды", header: "3", want: 3 * time.Second},
		{name: "ноль секунд", header: "0", want: 0},
		{name: "мусор", header: "soon", want: 0},
		{name: "отрицательное", header: "-5", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := stdhttp.Header{}
			if tt.header != "" {
				header.Set("Retry-After", tt.header)
			}

			if got := retryAfter(newResponse(429, header)); got != tt.want {
				t.Fatalf("retryAfter = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRetryAfterHTTPDate(t *testing.T) {
	header := stdhttp.Header{}
	header.Set("Retry-After", time.Now().Add(2*time.Second).UTC().Format(stdhttp.TimeFormat))

	delay := retryAfter(newResponse(429, header))
	if delay <= 0 || delay > 2*time.Second {
		t.Fatalf("retryAfter = %v, want (0, 2s]", delay)
	}

	// Дата в прошлом — пауза не нужна.
	header.Set("Retry-After", time.Now().Add(-time.Minute).UTC().Format(stdhttp.TimeFormat))
	if got := retryAfter(newResponse(429, header)); got != 0 {
		t.Fatalf("retryAfter = %v, want 0 for a past date", got)
	}
}
