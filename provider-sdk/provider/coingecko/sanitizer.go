package coingecko

import (
	"fmt"
	"math"
	"net/url"
	"regexp"
	"sort"
	"time"
	"github.com/testapiw/go-sdk/provider-sdk/provider/contract"
)

var idPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Sanitizer interface {
	SanitizeCoins([]coinDTO) ([]contract.Coin, error)
	SanitizePrice(priceDTO) (map[string]contract.Price, error)
	SanitizeHistory(historyDTO) ([]contract.HistoryPoint, error)
}
type sanitizer struct{}

func NewSanitizer() Sanitizer { return sanitizer{} }
func validID(v string) bool   { return idPattern.MatchString(v) }
func finite(v float64) bool   { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func (s sanitizer) SanitizeCoins(in []coinDTO) ([]contract.Coin, error) {
	out := make([]contract.Coin, 0, len(in))
	for _, v := range in {
		if !validID(v.ID) || v.Symbol == "" || v.Name == "" || v.CurrentPrice == nil || !finite(*v.CurrentPrice) {
			return nil, fmt.Errorf("%w: invalid coin", contract.ErrInvalidResponse)
		}
		if v.Image != "" {
			u, e := url.ParseRequestURI(v.Image)
			if e != nil || u.Scheme != "https" || u.Host == "" {
				return nil, fmt.Errorf("%w: invalid image URL", contract.ErrInvalidResponse)
			}
		}
		var updated time.Time
		if v.LastUpdated != "" {
			var e error
			updated, e = time.Parse(time.RFC3339, v.LastUpdated)
			if e != nil {
				return nil, fmt.Errorf("%w: invalid date", contract.ErrInvalidResponse)
			}
			updated = updated.UTC()
		}
		c := contract.Coin{ID: v.ID, Symbol: v.Symbol, Name: v.Name, ImageURL: v.Image, CurrentPrice: *v.CurrentPrice, LastUpdated: updated}
		if v.MarketCap != nil {
			if !finite(*v.MarketCap) {
				return nil, contract.ErrInvalidResponse
			}
			c.MarketCap = *v.MarketCap
		}
		if v.TotalVolume != nil {
			if !finite(*v.TotalVolume) {
				return nil, contract.ErrInvalidResponse
			}
			c.TotalVolume = *v.TotalVolume
		}
		if v.MarketCapRank != nil {
			c.MarketCapRank = *v.MarketCapRank
		}
		out = append(out, c)
	}
	return out, nil
}
func (s sanitizer) SanitizePrice(in priceDTO) (map[string]contract.Price, error) {
	out := make(map[string]contract.Price, len(in))
	for id, v := range in {
		if !validID(id) || v.USD == nil {
			return nil, fmt.Errorf("%w: invalid price", contract.ErrInvalidResponse)
		}

		// Точное строковое представление из JSON (json.Number сохраняет
		// исходный текст числа без потери точности).
		valueStr := v.USD.String()
		value, err := v.USD.Float64()
		if err != nil || !finite(value) {
			return nil, fmt.Errorf("%w: invalid price", contract.ErrInvalidResponse)
		}

		p := contract.Price{
			Value:    value,
			ValueStr: valueStr,
			Currency: "usd",
		}

		if v.MarketCap != nil {
			p.MarketCapStr = v.MarketCap.String()
			if mc, err := v.MarketCap.Float64(); err == nil {
				p.MarketCap = mc
			}
		}
		if v.Volume != nil {
			p.Volume24hStr = v.Volume.String()
			if vol, err := v.Volume.Float64(); err == nil {
				p.Volume24h = vol
			}
		}
		if v.Change != nil {
			p.Change24hStr = v.Change.String()
			if ch, err := v.Change.Float64(); err == nil {
				p.Change24h = ch
			}
		}
		if v.LastUpdated != nil {
			if *v.LastUpdated < 0 {
				return nil, fmt.Errorf("%w: invalid timestamp", contract.ErrInvalidResponse)
			}
			p.LastUpdated = time.Unix(*v.LastUpdated, 0).UTC()
		}
		if !finite(p.MarketCap) || !finite(p.Volume24h) || !finite(p.Change24h) {
			return nil, contract.ErrInvalidResponse
		}
		out[id] = p
	}
	return out, nil
}
func (s sanitizer) SanitizeHistory(in historyDTO) ([]contract.HistoryPoint, error) {
	if len(in.Prices) == 0 {
		return []contract.HistoryPoint{}, nil
	}
	caps := pairs(in.MarketCaps)
	vols := pairs(in.TotalVolumes)
	out := make([]contract.HistoryPoint, 0, len(in.Prices))
	for _, p := range in.Prices {
		if len(p) != 2 || p[0] < 0 || !finite(p[0]) || !finite(p[1]) {
			return nil, fmt.Errorf("%w: invalid history point", contract.ErrInvalidResponse)
		}
		ts := int64(p[0])
		hp := contract.HistoryPoint{Time: time.UnixMilli(ts).UTC(), Price: p[1], MarketCap: caps[ts], Volume: vols[ts]}
		out = append(out, hp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out, nil
}
func pairs(in [][]float64) map[int64]float64 {
	m := map[int64]float64{}
	for _, p := range in {
		if len(p) == 2 && finite(p[0]) && finite(p[1]) {
			m[int64(p[0])] = p[1]
		}
	}
	return m
}
