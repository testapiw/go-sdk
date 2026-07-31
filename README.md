# Go SDK

This directory contains reusable SDK modules.

## Available SDKs

### http-sdk
HTTP client infrastructure with retry, rate limiting, and circuit breaker support.

**Location:** `github.com/testapiw/go-sdk/http-sdk`

**Features:**
- Circuit breaker pattern (using gobreaker)
- Rate limiting with configurable requests per second
- Retry policies with exponential backoff and jitter
- HTTP client abstraction with middleware support

**Packages:**
- `breaker` - Circuit breaker implementation
- `client` - HTTP client with request/response handling
- `ratelimit` - Rate limiting functionality
- `retry` - Retry logic with configurable policies
- `transport` - HTTP middleware

### provider-sdk
Market data provider adapters and contracts.

**Location:** `github.com/testapiw/go-sdk/provider-sdk`

**Features:**
- Provider interface abstraction
- CoinGecko adapter implementation
- Base adapter with common functionality
- Data sanitization and validation
- Built-in error handling and classification

**Packages:**
- `provider/contract` - Provider interfaces and models
- `provider/base` - Base adapter implementation
- `provider/coingecko` - CoinGecko specific implementation
- `provider/factory` - Provider factory

## Installation

For production use:

```bash
go get github.com/testapiw/go-sdk/http-sdk@latest
go get github.com/testapiw/go-sdk/provider-sdk@latest
```

For local development, use the workspace and replace directives:

```go
// In your project's go.mod
replace (
    github.com/testapiw/go-sdk/http-sdk => ../SDK/http-sdk
    github.com/testapiw/go-sdk/provider-sdk => ../SDK/provider-sdk
)
```

## Development

### Workspace Setup
The SDK uses Go workspaces for local development:

```bash
cd SDK
go work sync
```

### Running Tests

Test all modules:
```bash
cd SDK
go test ./...
```

Test specific module:
```bash
cd SDK/http-sdk
go test ./...
```

### Publishing to GitHub

1. Commit and push your changes
2. Tag the release for each module:
   ```bash
   git tag http-sdk/v0.1.0
   git tag provider-sdk/v0.1.0
   git push origin --tags
   ```

3. Users can then install specific versions:
   ```bash
   go get github.com/testapiw/go-sdk/http-sdk@v0.1.0
   ```

## Usage Examples

### Using http-sdk

```go
import (
    "github.com/testapiw/go-sdk/http-sdk/client"
    "github.com/testapiw/go-sdk/http-sdk/retry"
)

httpClient := client.NewClient(nil, 10*time.Second)
retryExecutor := retry.New(retry.DefaultPolicy())
```

### Using provider-sdk

```go
import (
    "github.com/testapiw/go-sdk/provider-sdk/provider/coingecko"
)

cfg := coingecko.DefaultConfig()
cfg.APIKey = "your-api-key"

adapter, err := coingecko.New(cfg, nil, nil, nil)
if err != nil {
    log.Fatal(err)
}

prices, err := adapter.Prices(ctx, []string{"bitcoin", "ethereum"})
```

## Dependencies

### http-sdk
- `github.com/sony/gobreaker` - Circuit breaker implementation
- `golang.org/x/time` - Rate limiting

### provider-sdk
- `github.com/testapiw/go-sdk/http-sdk` - HTTP infrastructure
- `github.com/sony/gobreaker` - Circuit breaker states