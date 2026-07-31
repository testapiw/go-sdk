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

	httpx "github.com/testapiw/go-sdk/http-sdk/client"
	"github.com/testapiw/go-sdk/http-sdk/handlers"
	"github.com/testapiw/go-sdk/http-sdk/transport"

	"github.com/testapiw/go-sdk/provider-sdk/provider/base"
	"github.com/testapiw/go-sdk/provider-sdk/provider/contract"
)

type CoinGeckoAdapter struct {
	base      *base.BaseAdapter
	sanitizer Sanitizer
	config    Config

	// onResult is invoked after every request with the transport Result.
	// It is optional and never blocks the request lifecycle.
	onResult func(*transport.Result)
}

var _ contract.Provider = (*CoinGeckoAdapter)(nil)

// OnResult registers a callback invoked after every request with the
// transport Result (response, error, timing, attempts). The callback runs
// synchronously after the request completes; keep it fast or hand off to a
// queue. Passing nil disables it.
func (a *CoinGeckoAdapter) OnResult(fn func(*transport.Result)) {
	a.onResult = fn
}

func New(
	cfg Config,
	doer httpx.Doer,
) (*CoinGeckoAdapter, error) {

	if err := cfg.Validate(); err != nil {
		return nil, contract.NewError(
			contract.ErrConfiguration,
			"coingecko",
			"new",
			"",
			err,
		)
	}

	client := httpx.NewClient(
		doer,
		cfg.Timeout,
	)

	t, err := transport.New(client)
	if err != nil {
		return nil, contract.NewError(
			contract.ErrConfiguration,
			"coingecko",
			"new",
			"",
			err,
		)
	}

	// Register resilience handlers. When config fields are nil, defaults
	// are used so the provider is protected out of the box.
	retry := handlers.DefaultRetryPolicy()
	if cfg.Retry != nil {
		retry = *cfg.Retry
	}
	t.Use(handlers.NewRetry(retry))

	ratelimit := handlers.DefaultRateLimitConfig()
	if cfg.RateLimit != nil {
		ratelimit = *cfg.RateLimit
	}
	t.Use(handlers.NewRateLimit(ratelimit))

	breaker := handlers.DefaultBreakerConfig("coingecko")
	if cfg.Breaker != nil {
		breaker = *cfg.Breaker
	}
	t.Use(handlers.NewBreaker(breaker))

	return &CoinGeckoAdapter{
		base:      base.New(t),
		sanitizer: NewSanitizer(),
		config:    cfg,
	}, nil
}

func (a *CoinGeckoAdapter) Ping(ctx context.Context) error {

	var dto pingDTO

	if err := a.get(ctx, "Ping", PingEndpoint, nil, &dto); err != nil {
		return err
	}

	if dto.GeckoSays == "" {
		return contract.NewError(
			contract.ErrInvalidResponse,
			"coingecko",
			"ping",
			"",
			errors.New("missing gecko_says"),
		)
	}

	return nil
}

func (a *CoinGeckoAdapter) Prices(
	ctx context.Context,
	ids []string,
) (map[string]contract.Price, error) {

	if len(ids) == 0 {
		return map[string]contract.Price{}, nil
	}

	for _, id := range ids {
		if !validID(id) {
			return nil, contract.NewError(
				contract.ErrInvalidResponse,
				"coingecko",
				"prices",
				"",
				fmt.Errorf("invalid id %q", id),
			)
		}
	}

	q := url.Values{
		"ids":                     {strings.Join(ids, ",")},
		"vs_currencies":           {"usd"},
		"include_market_cap":      {"true"},
		"include_24hr_vol":        {"true"},
		"include_24hr_change":     {"true"},
		"include_last_updated_at": {"true"},
	}

	var dto priceDTO

	if err := a.get(ctx, "Prices", PriceEndpoint, q, &dto); err != nil {
		return nil, err
	}

	out, err := a.sanitizer.SanitizePrice(dto)
	if err != nil {
		return nil, contract.NewError(
			contract.ErrInvalidResponse,
			"coingecko",
			"prices",
			"",
			err,
		)
	}

	return out, nil
}

func (a *CoinGeckoAdapter) Coins(
	ctx context.Context,
	r contract.CoinsRequest,
) ([]contract.Coin, error) {

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

	q := url.Values{
		"vs_currency": {currency},
		"per_page":    {strconv.Itoa(per)},
		"page":        {strconv.Itoa(page)},
		"sparkline":   {strconv.FormatBool(r.Sparkline)},
	}

	if len(r.IDs) > 0 {
		q.Set("ids", strings.Join(r.IDs, ","))
	}

	if r.Order != "" {
		q.Set("order", r.Order)
	}

	var dto []coinDTO

	if err := a.get(ctx, "Coins", MarketsEndpoint, q, &dto); err != nil {
		return nil, err
	}

	out, err := a.sanitizer.SanitizeCoins(dto)
	if err != nil {
		return nil, contract.NewError(
			contract.ErrInvalidResponse,
			"coingecko",
			"coins",
			"",
			err,
		)
	}

	return out, nil
}

func (a *CoinGeckoAdapter) History(
	ctx context.Context,
	id string,
	from,
	to time.Time,
) ([]contract.HistoryPoint, error) {

	if !validID(id) ||
		from.IsZero() ||
		to.IsZero() ||
		!from.Before(to) {

		return nil, contract.NewError(
			contract.ErrInvalidResponse,
			"coingecko",
			"history",
			"",
			errors.New("invalid id or range"),
		)
	}

	endpoint := "/coins/" + url.PathEscape(id) + "/market_chart/range"

	q := url.Values{
		"vs_currency": {"usd"},
		"from":        {strconv.FormatInt(from.UTC().Unix(), 10)},
		"to":          {strconv.FormatInt(to.UTC().Unix(), 10)},
	}

	return a.history(ctx, endpoint, q)
}

func (a *CoinGeckoAdapter) HistoryRange(
	ctx context.Context,
	id string,
	currency string,
	days string,
) ([]contract.HistoryPoint, error) {

	if !validID(id) {
		return nil, contract.ErrInvalidResponse
	}

	if currency == "" {
		currency = "usd"
	}

	if days == "" {
		days = "1"
	}

	return a.history(
		ctx,
		"/coins/"+url.PathEscape(id)+"/market_chart",
		url.Values{
			"vs_currency": {currency},
			"days":        {days},
		},
	)
}

func (a *CoinGeckoAdapter) history(
	ctx context.Context,
	endpoint string,
	q url.Values,
) ([]contract.HistoryPoint, error) {

	var dto historyDTO

	if err := a.get(ctx, "History", endpoint, q, &dto); err != nil {
		return nil, err
	}

	out, err := a.sanitizer.SanitizeHistory(dto)
	if err != nil {
		return nil, contract.NewError(
			contract.ErrInvalidResponse,
			"coingecko",
			"history",
			"",
			err,
		)
	}

	return out, nil
}

func (a *CoinGeckoAdapter) get(
	ctx context.Context,
	name string,
	endpoint string,
	q url.Values,
	target any,
) error {

	requestID := requestID(ctx)

	headers := stdhttp.Header{
		"Accept":     {"application/json"},
		"User-Agent": {a.config.UserAgent},
	}

	if requestID != "" {
		headers.Set("X-Request-ID", requestID)
	}

	if a.config.APIKey != "" {

		header := "x-cg-demo-api-key"

		if a.config.APIType == APIPro {
			header = "x-cg-pro-api-key"
		}

		headers.Set(header, a.config.APIKey)
	}

	op := transport.Operation{
		Provider: "coingecko",
		Name:     name,
		Endpoint: endpoint,
		Method:   stdhttp.MethodGet,
	}

	result := a.base.Do(
		ctx,
		op,
		httpx.Request{
			Method: stdhttp.MethodGet,
			URL: strings.TrimRight(
				a.config.BaseURL,
				"/",
			) + endpoint,
			Query:   q,
			Headers: headers,
		},
	)

	// Notify the application about the result (logging/metrics). This runs
	// after the request completes and does not affect the returned data.
	if a.onResult != nil {
		a.onResult(result)
	}

	if result.Error != nil {
		return result.Error
	}

	if err := json.Unmarshal(result.Response.Body, target); err != nil {
		return contract.NewError(
			contract.ErrInvalidResponse,
			"coingecko",
			endpoint,
			requestID,
			err,
		)
	}

	return nil
}

type requestIDKey struct{}

func WithRequestID(
	ctx context.Context,
	id string,
) context.Context {

	return context.WithValue(
		ctx,
		requestIDKey{},
		id,
	)
}

func requestID(ctx context.Context) string {

	v, _ := ctx.Value(requestIDKey{}).(string)

	return v
}