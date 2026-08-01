# Go SDK

Reusable SDK modules for building HTTP clients and market-data provider adapters.

## Modules

- **http-sdk** — `github.com/testapiw/go-sdk/http-sdk`
  HTTP client with retry, rate limiting, and circuit breaker.

- **provider-sdk** — `github.com/testapiw/go-sdk/provider-sdk`
  Provider adapters and contracts (CoinGecko, factory).

## Installation

```bash
go get github.com/testapiw/go-sdk/http-sdk@latest
go get github.com/testapiw/go-sdk/provider-sdk@latest
```

For local development, use the workspace and replace directives:

```go
replace (
    github.com/testapiw/go-sdk/http-sdk => ../SDK/http-sdk
    github.com/testapiw/go-sdk/provider-sdk => ../SDK/provider-sdk
)
```

## Usage

### Build a provider through the factory

The factory selects a provider by name and reads credentials from the
environment:

```go
import "github.com/testapiw/go-sdk/provider-sdk/provider/factory"

adapter, err := factory.New().Create("coingecko")
if err != nil {
    log.Fatal(err)
}

prices, err := adapter.Prices(ctx, []string{"bitcoin", "ethereum"})
```

### Build a provider directly

```go
import "github.com/testapiw/go-sdk/provider-sdk/provider/coingecko"

cfg := coingecko.DefaultConfig()
cfg.APIKey = "your-api-key"

adapter, err := coingecko.New("coingecko", cfg, nil)
if err != nil {
    log.Fatal(err)
}

prices, err := adapter.Prices(ctx, []string{"bitcoin", "ethereum"})
```

### Use the HTTP client directly

```go
import "github.com/testapiw/go-sdk/http-sdk/client"

httpClient := client.NewClient(nil, 10*time.Second)
```

## Development

```bash
cd SDK
go work sync
go test ./...
```

## Publishing

Tag each module and push:

```bash
git tag http-sdk/v0.1.0
git tag provider-sdk/v0.1.0
git push origin --tags
```