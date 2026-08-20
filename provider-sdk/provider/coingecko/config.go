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
// COINGECKO_API_MODE environment variable (demo | pro) to select the API
// endpoint and type. When mode is "pro", the COINGECKO_API_KEY environment
// variable is required and applied.

// .env COINGECKO_API_MODE  demo | pro
func DefaultConfig() (Config, error) {
	cfg := Config{
		BaseURL:   "https://api.coingecko.com/api/v3", // demo
		APIType:   APIDemo,
		UserAgent: "xentaura-market-data/1.0",
		Config: base.Config{
			Timeout: 10 * time.Second,
		},
	}

	switch os.Getenv("COINGECKO_API_MODE") {
	case string(APIPro):
		key := os.Getenv("COINGECKO_API_KEY")
		if key == "" {
			return Config{}, fmt.Errorf("COINGECKO_API_MODE=pro requires COINGECKO_API_KEY to be set")
		}
		cfg.BaseURL = "https://pro-api.coingecko.com/api/v3"
		cfg.APIType = APIPro
		cfg.APIKey = key
	default: // "demo" or unset
		cfg.BaseURL = "https://api.coingecko.com/api/v3"
		cfg.APIType = APIDemo
	}

	return cfg, nil
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