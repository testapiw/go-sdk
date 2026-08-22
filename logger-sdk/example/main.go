// Пример использования sdk-logger.
//
// Запуск:
//
//	cd SDK/logger-sdk
//	go run ./example
//
// Конфигурация через переменные окружения (см. README.md):
//
//	LOG_LEVEL=debug LOG_DIR=/tmp/logs go run ./example
//
// Можно также скопировать .env.example в .env рядом с примером —
// он будет загружен автоматически.
package main

import (
	"bufio"
	"log"
	"os"
	"strings"

	"github.com/testapiw/go-sdk/logger-sdk"
)

func main() {
	loadDotEnv(".env")

	l, err := logger.New(logger.ConfigFromEnv())
	if err != nil {
		log.Fatalf("logger init: %v", err)
	}
	defer l.Close()

	l.Info("service started")
	l.Debug("request", map[string]any{"path": "/api/v1/prices", "method": "GET"})
	l.Info("prices updated", map[string]any{"count": 42})

	// error/critical пишутся всегда (в файл и в БД, если включена).
	l.Error("failed to fetch prices", map[string]any{"provider": "coingecko"})
	l.Critical("database connection lost", map[string]any{"host": "127.0.0.1:9000"})

	// critical с уведомлением в Telegram (если включено).
	l.CriticalNotify("service is down", map[string]any{"service": "pricewriter"})

	log.Println("done. check logs in", logger.ConfigFromEnv().LogDir)
}

// loadDotEnv загружает переменные из файла .env (KEY=VALUE), не перезаписывая
// уже установленные переменные окружения.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // нет файла — не ошибка
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if key == "" {
			continue
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}
