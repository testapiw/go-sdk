package coingecko

import "encoding/json"

type pingDTO struct {
	GeckoSays string `json:"gecko_says"`
}
type priceDTO map[string]struct {
	USD         *json.Number `json:"usd"`
	MarketCap   *json.Number `json:"usd_market_cap"`
	Volume      *json.Number `json:"usd_24h_vol"`
	Change      *json.Number `json:"usd_24h_change"`
	LastUpdated *int64       `json:"last_updated_at"`
}
type coinDTO struct {
	ID            string   `json:"id"`
	Symbol        string   `json:"symbol"`
	Name          string   `json:"name"`
	Image         string   `json:"image"`
	CurrentPrice  *float64 `json:"current_price"`
	MarketCap     *float64 `json:"market_cap"`
	MarketCapRank *int     `json:"market_cap_rank"`
	TotalVolume   *float64 `json:"total_volume"`
	LastUpdated   string   `json:"last_updated"`
}
type historyDTO struct {
	Prices       [][]float64 `json:"prices"`
	MarketCaps   [][]float64 `json:"market_caps"`
	TotalVolumes [][]float64 `json:"total_volumes"`
}
