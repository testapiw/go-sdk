package coingecko

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	stdhttp "net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/testapiw/go-sdk/http-sdk/breaker"
	httpx "github.com/testapiw/go-sdk/http-sdk/client"
	"github.com/testapiw/go-sdk/http-sdk/ratelimit"
	"github.com/testapiw/go-sdk/http-sdk/retry"
	"github.com/testapiw/go-sdk/provider-sdk/provider/base"
	"github.com/testapiw/go-sdk/provider-sdk/provider/contract"
)

type CoinGeckoAdapter struct {
	base      *base.BaseAdapter
	sanitizer Sanitizer
	config    Config
}

var _ contract.Provider = (*CoinGeckoAdapter)(nil)

func New(cfg Config, doer httpx.Doer, logger base.Logger, metrics base.Metrics) (*CoinGeckoAdapter, error) {
	if err := cfg.Validate(); err != nil {
		return nil, contract.NewError(contract.ErrConfiguration, "coingecko", "new", "", err)
	}
	limiter, err := ratelimit.New(cfg.RateLimiter)
	if err != nil {
		return nil, contract.NewError(contract.ErrConfiguration, "coingecko", "new", "", err)
	}
	client := httpx.NewClient(doer, cfg.Timeout)
	b := base.New(client, retry.New(cfg.Retry), breaker.New(cfg.CircuitBreaker), limiter, logger, metrics)
	return &CoinGeckoAdapter{base: b, sanitizer: NewSanitizer(), config: cfg}, nil
}
func (a *CoinGeckoAdapter) Ping(ctx context.Context) error {
	var dto pingDTO
	if err := a.get(ctx, PingEndpoint, nil, &dto); err != nil {
		return err
	}
	if dto.GeckoSays == "" {
		return contract.NewError(contract.ErrInvalidResponse, "coingecko", "ping", "", errors.New("missing gecko_says"))
	}
	return nil
}
func (a *CoinGeckoAdapter) Prices(ctx context.Context, ids []string) (map[string]contract.Price, error) {
	if len(ids) == 0 {
		return map[string]contract.Price{}, nil
	}
	for _, id := range ids {
		if !validID(id) {
			return nil, contract.NewError(contract.ErrInvalidResponse, "coingecko", "prices", "", fmt.Errorf("invalid id %q", id))
		}
	}
	q := url.Values{"ids": {strings.Join(ids, ",")}, "vs_currencies": {"usd"}, "include_market_cap": {"true"}, "include_24hr_vol": {"true"}, "include_24hr_change": {"true"}, "include_last_updated_at": {"true"}}
	var dto priceDTO
	if err := a.get(ctx, PriceEndpoint, q, &dto); err != nil {
		return nil, err
	}
	out, err := a.sanitizer.SanitizePrice(dto)
	if err != nil {
		return nil, contract.NewError(contract.ErrInvalidResponse, "coingecko", "prices", "", err)
	}
	return out, nil
}
func (a *CoinGeckoAdapter) Coins(ctx context.Context, r contract.CoinsRequest) ([]contract.Coin, error) {
	currency := r.Currency
	if currency == "" {
		currency = "usd"
	}
	per := r.PerPage
	if per == 0 {
		per = 100
	}
	page := r.Page
	if page == 0 {
		page = 1
	}
	q := url.Values{"vs_currency": {currency}, "per_page": {strconv.Itoa(per)}, "page": {strconv.Itoa(page)}, "sparkline": {strconv.FormatBool(r.Sparkline)}}
	if len(r.IDs) > 0 {
		q.Set("ids", strings.Join(r.IDs, ","))
	}
	if r.Order != "" {
		q.Set("order", r.Order)
	}
	var dto []coinDTO
	if err := a.get(ctx, MarketsEndpoint, q, &dto); err != nil {
		return nil, err
	}
	out, err := a.sanitizer.SanitizeCoins(dto)
	if err != nil {
		return nil, contract.NewError(contract.ErrInvalidResponse, "coingecko", "coins", "", err)
	}
	return out, nil
}
func (a *CoinGeckoAdapter) History(ctx context.Context, id string, from, to time.Time) ([]contract.HistoryPoint, error) {
	if !validID(id) || from.IsZero() || to.IsZero() || !from.Before(to) {
		return nil, contract.NewError(contract.ErrInvalidResponse, "coingecko", "history", "", errors.New("invalid id or range"))
	}
	endpoint := "/coins/" + url.PathEscape(id) + "/market_chart/range"
	q := url.Values{"vs_currency": {"usd"}, "from": {strconv.FormatInt(from.UTC().Unix(), 10)}, "to": {strconv.FormatInt(to.UTC().Unix(), 10)}}
	return a.history(ctx, endpoint, q)
}
func (a *CoinGeckoAdapter) HistoryRange(ctx context.Context, id, currency, days string) ([]contract.HistoryPoint, error) {
	if !validID(id) {
		return nil, contract.ErrInvalidResponse
	}
	if currency == "" {
		currency = "usd"
	}
	if days == "" {
		days = "1"
	}
	return a.history(ctx, "/coins/"+url.PathEscape(id)+"/market_chart", url.Values{"vs_currency": {currency}, "days": {days}})
}
func (a *CoinGeckoAdapter) history(ctx context.Context, endpoint string, q url.Values) ([]contract.HistoryPoint, error) {
	var dto historyDTO
	if err := a.get(ctx, endpoint, q, &dto); err != nil {
		return nil, err
	}
	out, err := a.sanitizer.SanitizeHistory(dto)
	if err != nil {
		return nil, contract.NewError(contract.ErrInvalidResponse, "coingecko", "history", "", err)
	}
	return out, nil
}
func (a *CoinGeckoAdapter) get(ctx context.Context, endpoint string, q url.Values, target any) error {
	requestID := requestID(ctx)
	headers := stdhttp.Header{"Accept": {"application/json"}, "User-Agent": {a.config.UserAgent}}
	if requestID != "" {
		headers.Set("X-Request-ID", requestID)
	}
	if a.config.APIKey != "" {
		name := "x-cg-demo-api-key"
		if a.config.APIType == APIPro {
			name = "x-cg-pro-api-key"
		}
		headers.Set(name, a.config.APIKey)
	}
	resp, err := a.base.Do(ctx, "coingecko", endpoint, requestID, httpx.Request{Method: stdhttp.MethodGet, URL: strings.TrimRight(a.config.BaseURL, "/") + endpoint, Query: q, Headers: headers})
	if err != nil {
		return err
	}
	if err = json.Unmarshal(resp.Body, target); err != nil {
		return contract.NewError(contract.ErrInvalidResponse, "coingecko", endpoint, requestID, err)
	}
	return nil
}

type requestIDKey struct{}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}
func requestID(ctx context.Context) string { v, _ := ctx.Value(requestIDKey{}).(string); return v }
