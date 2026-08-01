package coingecko

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/testapiw/go-sdk/provider-sdk/provider/base"
)

type APIType string

const (
	APIDemo APIType = "demo"
	APIPro  APIType = "pro"
)

type Config struct {
	BaseURL string
	APIKey  string
	APIType APIType

	UserAgent string

	// Transport-level settings (timeout, resilience handlers).
	base.Config
}

// DefaultConfig returns a config with sensible defaults. It reads the
// COINGECKO_API_KEY environment variable: when set, the provider switches to
// the pro API; otherwise it uses the public demo API.
func DefaultConfig() Config {
	cfg := Config{
		BaseURL:   "https://api.coingecko.com/api/v3",
		APIType:   APIDemo,
		UserAgent: "xentaura-market-data/1.0",
		Config: base.Config{
			Timeout: 10 * time.Second,
		},
	}

	if key := os.Getenv("COINGECKO_API_KEY"); key != "" {
		cfg.APIKey = key
		cfg.APIType = APIPro
	}

	return cfg
}

func (c Config) Validate() error {
	u, err := url.Parse(c.BaseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid BaseURL")
	}

	if c.APIType != APIDemo && c.APIType != APIPro {
		return fmt.Errorf("invalid APIType")
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("invalid timeout")
	}

	return nil
}