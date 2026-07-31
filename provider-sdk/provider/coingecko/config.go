package coingecko

import (
	"fmt"
	"net/url"
	"time"
	"github.com/testapiw/go-sdk/http-sdk/breaker"
	"github.com/testapiw/go-sdk/http-sdk/ratelimit"
	"github.com/testapiw/go-sdk/http-sdk/retry"
)

type APIType string

const (
	APIDemo APIType = "demo"
	APIPro  APIType = "pro"
)

type Config struct {
	BaseURL, APIKey string
	APIType         APIType
	Timeout         time.Duration
	Retry           retry.Policy
	RateLimiter     ratelimit.Config
	CircuitBreaker  breaker.Config
	UserAgent       string
}

func DefaultConfig() Config {
	return Config{BaseURL: "https://api.coingecko.com/api/v3", APIType: APIDemo, Timeout: 10 * time.Second, Retry: retry.DefaultPolicy(), RateLimiter: ratelimit.Config{RequestsPerSecond: 5, Burst: 1}, CircuitBreaker: breaker.Config{Name: "coingecko", MaxRequests: 1, Timeout: 30 * time.Second, FailureThreshold: 5}, UserAgent: "xentaura-market-data/1.0"}
}
func (c Config) Validate() error {
	u, e := url.Parse(c.BaseURL)
	if e != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid BaseURL")
	}
	if c.APIType != APIDemo && c.APIType != APIPro {
		return fmt.Errorf("invalid APIType")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("invalid timeout")
	}
	if !c.Retry.Validate() {
		return fmt.Errorf("invalid retry policy")
	}
	return nil
}
