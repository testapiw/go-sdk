package coingecko

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testapiw/go-sdk/provider-sdk/provider/contract"
)

func testConfig(rawURL string) Config {
	cfg := DefaultConfig()
	cfg.BaseURL = rawURL
	cfg.APIKey = "secret"
	cfg.RateLimiter.RequestsPerSecond = 1000
	cfg.Retry.InitialDelay = time.Millisecond
	cfg.Retry.MaxDelay = time.Millisecond
	cfg.Retry.Jitter = false
	return cfg
}

func TestAdapterEndpoints(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "secret", r.Header.Get("x-cg-demo-api-key"))
		require.Equal(t, "req-1", r.Header.Get("X-Request-ID"))
		_, _ = w.Write([]byte(`{"gecko_says":"(V3) To the Moon!"}`))
	})
	mux.HandleFunc("/simple/price", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "bitcoin", r.URL.Query().Get("ids"))
		_, _ = w.Write([]byte(`{"bitcoin":{"usd":62000,"usd_market_cap":10,"usd_24h_vol":2,"usd_24h_change":1.5,"last_updated_at":1700000000}}`))
	})
	mux.HandleFunc("/coins/markets", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"id":"bitcoin","symbol":"btc","name":"Bitcoin","image":"https://img.example/btc.png","current_price":62000,"market_cap":10,"market_cap_rank":1,"total_volume":2,"last_updated":"2024-01-01T00:00:00Z"}]`))
	})
	mux.HandleFunc("/coins/bitcoin/market_chart", historyHandler)
	mux.HandleFunc("/coins/bitcoin/market_chart/range", historyHandler)
	server := httptest.NewServer(mux)
	defer server.Close()

	a, err := New(testConfig(server.URL), nil, nil, nil)
	require.NoError(t, err)
	ctx := WithRequestID(context.Background(), "req-1")
	require.NoError(t, a.Ping(ctx))
	prices, err := a.Prices(ctx, []string{"bitcoin"})
	require.NoError(t, err)
	require.Equal(t, 62000.0, prices["bitcoin"].Value)
	coins, err := a.Coins(ctx, contract.CoinsRequest{})
	require.NoError(t, err)
	require.Equal(t, "btc", coins[0].Symbol)
	points, err := a.HistoryRange(ctx, "bitcoin", "usd", "1")
	require.NoError(t, err)
	require.Equal(t, 100.0, points[0].MarketCap)
	points, err = a.History(ctx, "bitcoin", time.Unix(1, 0), time.Unix(2, 0))
	require.NoError(t, err)
	require.Equal(t, 5.0, points[0].Volume)
}

func historyHandler(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte(`{"prices":[[1700000000000,10]],"market_caps":[[1700000000000,100]],"total_volumes":[[1700000000000,5]]}`))
}

func TestRetryAndErrors(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) < 3 {
			http.Error(w, "busy", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"gecko_says":"ok"}`))
	}))
	defer server.Close()
	a, err := New(testConfig(server.URL), nil, nil, nil)
	require.NoError(t, err)
	require.NoError(t, a.Ping(context.Background()))
	require.Equal(t, int32(3), calls.Load())

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "no", http.StatusUnauthorized) }))
	defer bad.Close()
	a, err = New(testConfig(bad.URL), nil, nil, nil)
	require.NoError(t, err)
	require.ErrorIs(t, a.Ping(context.Background()), contract.ErrUnauthorized)
}

func TestInvalidJSONAndSanitizer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{`)) }))
	defer server.Close()
	a, err := New(testConfig(server.URL), nil, nil, nil)
	require.NoError(t, err)
	require.ErrorIs(t, a.Ping(context.Background()), contract.ErrInvalidResponse)

	_, err = NewSanitizer().SanitizeCoins([]coinDTO{{ID: "BAD ID"}})
	require.ErrorIs(t, err, contract.ErrInvalidResponse)
	price := 1.0
	_, err = NewSanitizer().SanitizeCoins([]coinDTO{{ID: "bitcoin", Symbol: "btc", Name: "Bitcoin", Image: "javascript:bad", CurrentPrice: &price}})
	require.ErrorIs(t, err, contract.ErrInvalidResponse)
	negative := int64(-1)
	_, err = NewSanitizer().SanitizePrice(priceDTO{"bitcoin": {USD: &price, LastUpdated: &negative}})
	require.ErrorIs(t, err, contract.ErrInvalidResponse)
	_, err = NewSanitizer().SanitizeHistory(historyDTO{Prices: [][]float64{{-1, 2}}})
	require.True(t, errors.Is(err, contract.ErrInvalidResponse))
}
