# Go SDK

Reusable SDK modules for building HTTP clients and market-data provider adapters.

## Modules

- **http-sdk** — `github.com/testapiw/go-sdk/http-sdk`
  HTTP client with retry, rate limiting, and circuit breaker.

- **provider-sdk** — `github.com/testapiw/go-sdk/provider-sdk`
  Provider adapters and contracts (CoinGecko, factory).

- **logger-sdk** — `github.com/testapiw/go-sdk/logger-sdk`
  Логгер с уровнями сообщений: info/debug в файл с ротацией по дням,
  error/critical в базу данных (ClickHouse), уведомления в Telegram
  для critical с дедупликацией.

## Installation

```bash
go get github.com/testapiw/go-sdk/http-sdk@latest
go get github.com/testapiw/go-sdk/provider-sdk@latest
go get github.com/testapiw/go-sdk/logger-sdk@latest
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

### Use the logger

```go
import "github.com/testapiw/go-sdk/logger-sdk"

l, err := logger.New(logger.ConfigFromEnv())
if err != nil {
    log.Fatal(err)
}
defer l.Close()

l.Info("service started")
l.Debug("request", map[string]any{"path": "/api"})
l.Error("failed to connect", err)
l.Critical("database down", map[string]any{"host": "..."}, true) // true — уведомить в Telegram
```

Конфигурация через переменные окружения:

| Переменная | Описание | По умолчанию |
|---|---|---|
| `LOG_LEVEL` | минимальный уровень для файла: `debug`/`info` | `info` |
| `LOG_DIR` | директория логов | `./logs` |
| `LOG_FILE_PREFIX` | префикс имени файла | `app` |
| `LOG_COINS_THRESHOLD` | минимальное количество реальных монет | `95` |
| `LOG_DB_ENABLED` | запись error/critical в БД | `false` |
| `CLICKHOUSE_ADDR` | адрес ClickHouse | `127.0.0.1:9000` |
| `CLICKHOUSE_DB` | база данных | `analytics` |
| `CLICKHOUSE_USER` | пользователь | `analytics` |
| `CLICKHOUSE_PASSWORD` | пароль | `analytics_ch` |
| `LOG_DB_TABLE` | таблица логов | `app_logs` |
| `LOG_TG_ENABLED` | уведомления в Telegram | `false` |
| `LOG_TG_BOT_TOKEN` | токен бота | — |
| `LOG_TG_CHAT_IDS` | получатели (ID чатов/каналов), через запятую | — |
| `LOG_TG_DEDUP_MIN` | интервал дедупликации (мин) | `30` |

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