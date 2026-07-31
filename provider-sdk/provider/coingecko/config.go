package coingecko

import (
	"fmt"
	"net/url"
	"time"
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

	Timeout time.Duration

	UserAgent string
}

func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://api.coingecko.com/api/v3",
		APIType:   APIDemo,
		Timeout:   10 * time.Second,
		UserAgent: "xentaura-market-data/1.0",
	}
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