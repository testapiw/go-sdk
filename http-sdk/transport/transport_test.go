package transport

import (
	"context"
	"errors"
	stdhttp "net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	http "github.com/testapiw/go-sdk/http-sdk/client"
	"github.com/testapiw/go-sdk/http-sdk/contract"
	"github.com/testapiw/go-sdk/http-sdk/handlers"
)

// sequenceServer поднимает сервер, который отдаёт заданную последовательность
// статусов (по одному на запрос, последний статус повторяется) и считает
// фактическое число вызовов. Заголовки добавляются к каждому ответу.
func sequenceServer(
	t *testing.T,
	statuses []int,
	header map[string]string,
) (*httptest.Server, *int32) {

	t.Helper()

	var calls int32

	srv := httptest.NewServer(stdhttp.HandlerFunc(
		func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			n := int(atomic.AddInt32(&calls, 1))
			idx := n - 1
			if idx >= len(statuses) {
				idx = len(statuses) - 1
			}

			for k, v := range header {
				w.Header().Set(k, v)
			}

			w.WriteHeader(statuses[idx])
		},
	))
	t.Cleanup(srv.Close)

	return srv, &calls
}

// fastTransport собирает транспорт с быстрыми задержками и снятым
// лимитом/предохранителем, чтобы тесты не зависели от таймингов.
func fastTransport(t *testing.T, maxRetries int, base, max time.Duration) *Transport {
	t.Helper()

	client := http.NewClient(nil, 0)

	cfg := handlers.Config{
		Retry: &handlers.RetryPolicy{
			MaxRetries: maxRetries,
			BaseDelay:  base,
			MaxDelay:   max,
		},
		RateLimit: &handlers.RateLimitConfig{RequestsPerSecond: 1000, Burst: 100},
	}

	transport, err := New("test", client, cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return transport
}

func doRequest(t *testing.T, transport *Transport, url string) *Result {
	t.Helper()

	return transport.Do(context.Background(), contract.Operation{
		Provider: "coingecko",
		Name:     "prices",
		Endpoint: "/api/v3/coins/markets",
		Method:   "GET",
	}, http.Request{Method: "GET", URL: url})
}

func TestDoWithoutRetries(t *testing.T) {
	srv, calls := sequenceServer(t, []int{200}, nil)
	transport := fastTransport(t, 3, time.Millisecond, 2*time.Millisecond)

	result := doRequest(t, transport, srv.URL)

	if result.Error != nil {
		t.Fatalf("error = %v, want nil", result.Error)
	}
	if result.StatusCode() != 200 {
		t.Fatalf("status = %d, want 200", result.StatusCode())
	}
	if result.Attempts != 1 || result.Retries != 0 {
		t.Fatalf("attempts/retries = %d/%d, want 1/0", result.Attempts, result.Retries)
	}
	if result.Waited != 0 {
		t.Fatalf("waited = %v, want 0 (успех с первой попытки)", result.Waited)
	}
	if got := int(atomic.LoadInt32(calls)); got != 1 {
		t.Fatalf("http calls = %d, want 1", got)
	}

	journal := result.AttemptLog
	if len(journal) != 1 {
		t.Fatalf("journal = %d records, want 1", len(journal))
	}
	if journal[0].Number != 1 || journal[0].StatusCode != 200 || journal[0].Retry {
		t.Fatalf("record = %+v, want number=1 status=200 retry=false", journal[0])
	}
	if journal[0].Duration() <= 0 {
		t.Fatalf("duration = %v, want > 0", journal[0].Duration())
	}
}

func TestDoRetriesAndKeepsAttemptJournal(t *testing.T) {
	srv, calls := sequenceServer(t, []int{503, 503, 200}, nil)
	transport := fastTransport(t, 3, time.Millisecond, 4*time.Millisecond)

	result := doRequest(t, transport, srv.URL)

	if result.Error != nil {
		t.Fatalf("error = %v, want nil", result.Error)
	}
	if result.StatusCode() != 200 {
		t.Fatalf("status = %d, want 200", result.StatusCode())
	}
	if result.Attempts != 3 || result.Retries != 2 {
		t.Fatalf("attempts/retries = %d/%d, want 3/2", result.Attempts, result.Retries)
	}
	if got := int(atomic.LoadInt32(calls)); got != 3 {
		t.Fatalf("http calls = %d, want 3", got)
	}

	journal := result.AttemptLog
	if len(journal) != 3 {
		t.Fatalf("journal = %d records, want 3", len(journal))
	}

	wantStatus := []int{503, 503, 200}
	wantRetry := []bool{true, true, false}
	for i, record := range journal {
		if record.Number != i+1 {
			t.Fatalf("record %d: number = %d, want %d", i, record.Number, i+1)
		}
		if record.StatusCode != wantStatus[i] {
			t.Fatalf("record %d: status = %d, want %d", i, record.StatusCode, wantStatus[i])
		}
		if record.Retry != wantRetry[i] {
			t.Fatalf("record %d: retry = %v, want %v", i, record.Retry, wantRetry[i])
		}
	}
}

func TestDoAppliesRetryPause(t *testing.T) {
	// Retry-After детерминирован (в отличие от backoff с полным jitter),
	// поэтому именно он доказывает, что пауза реально применяется.
	srv, calls := sequenceServer(t, []int{429, 200}, map[string]string{"Retry-After": "1"})
	transport := fastTransport(t, 3, time.Millisecond, 2*time.Millisecond)

	started := time.Now()
	result := doRequest(t, transport, srv.URL)
	elapsed := time.Since(started)

	if result.Error != nil {
		t.Fatalf("error = %v, want nil", result.Error)
	}
	if result.StatusCode() != 200 {
		t.Fatalf("status = %d, want 200", result.StatusCode())
	}
	if result.Attempts != 2 || result.Retries != 1 {
		t.Fatalf("attempts/retries = %d/%d, want 2/1", result.Attempts, result.Retries)
	}
	if result.Waited < time.Second {
		t.Fatalf("waited = %v, want >= 1s (Retry-After не применён)", result.Waited)
	}
	if elapsed < time.Second {
		t.Fatalf("elapsed = %v, want >= 1s (пауза не выдержана)", elapsed)
	}
	if got := int(atomic.LoadInt32(calls)); got != 2 {
		t.Fatalf("http calls = %d, want 2", got)
	}
	if result.AttemptLog[0].RetryDelay != time.Second {
		t.Fatalf("retry delay = %v, want 1s", result.AttemptLog[0].RetryDelay)
	}
}

func TestDoStopsRetriesWhenContextCancelled(t *testing.T) {
	srv, _ := sequenceServer(t, []int{429}, map[string]string{"Retry-After": "30"})
	transport := fastTransport(t, 3, time.Millisecond, 2*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	started := time.Now()
	result := transport.Do(ctx, contract.Operation{
		Provider: "coingecko",
		Name:     "prices",
		Endpoint: "/api/v3/coins/markets",
		Method:   "GET",
	}, http.Request{Method: "GET", URL: srv.URL})
	elapsed := time.Since(started)

	if !errors.Is(result.Error, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context.DeadlineExceeded", result.Error)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("elapsed = %v, want < 5s (ожидание прервано контекстом)", elapsed)
	}
	// Журнал попыток доступен и на пути отмены.
	if len(result.AttemptLog) != 1 || result.Attempts != 1 {
		t.Fatalf("journal = %d records (attempts=%d), want 1/1", len(result.AttemptLog), result.Attempts)
	}
}

func TestDoRetriesExhausted(t *testing.T) {
	srv, calls := sequenceServer(t, []int{503}, nil)
	transport := fastTransport(t, 2, time.Millisecond, 2*time.Millisecond)

	result := doRequest(t, transport, srv.URL)

	if result.StatusCode() != 503 {
		t.Fatalf("status = %d, want 503", result.StatusCode())
	}
	if result.Attempts != 3 || result.Retries != 2 {
		t.Fatalf("attempts/retries = %d/%d, want 3/2", result.Attempts, result.Retries)
	}
	if got := int(atomic.LoadInt32(calls)); got != 3 {
		t.Fatalf("http calls = %d, want 3", got)
	}
	if result.AttemptLog[2].Retry {
		t.Fatal("последняя попытка не должна быть помечена как повтор")
	}
}
