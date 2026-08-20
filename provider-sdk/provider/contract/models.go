package contract

import "time"

type Coin struct {
	ID, Symbol, Name                     string
	ImageURL                             string
	CurrentPrice, MarketCap, TotalVolume float64
	MarketCapRank                        int
	LastUpdated                          time.Time
}

type Price struct {
	// Числовые значения (для расчётов).
	Value, MarketCap, Volume24h, Change24h float64
	// Строковые значения (точное представление из JSON, для хранения).
	ValueStr, MarketCapStr, Volume24hStr, Change24hStr string
	Currency                                           string
	LastUpdated                                        time.Time
}

type HistoryPoint struct {
	Time                     time.Time
	Price, MarketCap, Volume float64
}

type CoinsRequest struct {
	Currency      string
	IDs           []string
	Order         string
	PerPage, Page int
	Sparkline     bool
}

type ProviderResponse[T any] struct {
	Data       T
	RequestID  string
	ReceivedAt time.Time
}
