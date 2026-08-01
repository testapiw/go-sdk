package contract

import (
	"context"
	"time"

	"github.com/testapiw/go-sdk/http-sdk/transport"
)

type Provider interface {
	Ping(context.Context) error
	Coins(context.Context, CoinsRequest) ([]Coin, error)
	Prices(context.Context, []string) (map[string]Price, error)
	History(context.Context, string, time.Time, time.Time) ([]HistoryPoint, error)
	HistoryRange(context.Context, string, string, string) ([]HistoryPoint, error)

	// OnResult registers a callback invoked after every request with the
	// transport Result (response, error, timing, attempts). Passing nil
	// disables it.
	OnResult(func(*transport.Result))
}
