package contract

import (
	"context"
	"time"
)

type Provider interface {
	Ping(context.Context) error
	Coins(context.Context, CoinsRequest) ([]Coin, error)
	Prices(context.Context, []string) (map[string]Price, error)
	History(context.Context, string, time.Time, time.Time) ([]HistoryPoint, error)
	HistoryRange(context.Context, string, string, string) ([]HistoryPoint, error)
}
