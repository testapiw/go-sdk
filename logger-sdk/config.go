// Package logger — простой SDK-логгер с уровнями сообщений.
//
// Уровни:
//   - debug, info — пишутся в файл с ротацией по дням (включаются по требованию через .env)
//   - error, critical — пишутся всегда, в базу данных (ClickHouse)
//   - critical — дополнительно может отправлять уведомление в Telegram
//
// Логика существующего логирования не меняется — модуль создан для тестирования
// и последующей замены.
package logger

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Level — уровень логирования.
type Level int

const (
	// LevelDebug — отладочные сообщения (включаются по требованию).
	LevelDebug Level = iota
	// LevelInfo — информационные сообщения (включаются по требованию).
	LevelInfo
	// LevelError — ошибки, пишутся всегда.
	LevelError
	// LevelCritical — критические ошибки, пишутся всегда.
	LevelCritical
)

// String возвращает строковое представление уровня.
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelError:
		return "ERROR"
	case LevelCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel разбирает строку уровня из конфигурации.
func ParseLevel(s string) Level {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "DEBUG":
		return LevelDebug
	case "INFO":
		return LevelInfo
	case "ERROR":
		return LevelError
	case "CRITICAL":
		return LevelCritical
	default:
		return LevelInfo
	}
}

// Config — параметры логгера, читаются из переменных окружения (.env).
type Config struct {
	// Level — минимальный уровень для записи в файл (debug/info).
	// error/critical пишутся всегда независимо от этого значения.
	Level Level

	// LogDir — директория для файлов логов (info/debug).
	LogDir string

	// FilePrefix — префикс имени файла лога, например "app".
	// Файлы: <LogDir>/<FilePrefix>-2026-08-22.log
	FilePrefix string

	// CoinsThreshold — минимальное количество реальных монет.
	// Если получено меньше этого количества — critical.
	// По умолчанию 95.
	CoinsThreshold int

	// DB — параметры подключения к ClickHouse для записи error/critical.
	DB DBConfig

	// Telegram — параметры уведомлений в Telegram для critical.
	Telegram TelegramConfig
}

// DBConfig — параметры подключения к базе данных (ClickHouse).
type DBConfig struct {
	Addr     string // host:port, например "127.0.0.1:9000"
	Database string
	Username string
	Password string
	Table    string // таблица для логов (по умолчанию "app_logs")
	Enabled  bool   // false — запись в БД отключена
}

// TelegramConfig — параметры уведомлений в Telegram.
type TelegramConfig struct {
	BotToken string // токен бота
	// ChatIDs — получатели (ID чатов/каналов), через запятую.
	ChatIDs []string
	Enabled bool // false — уведомления отключены
	// DedupWindow — минимальный интервал между повторными уведомлениями
	// об одном и том же сообщении (по умолчанию 30 минут).
	DedupWindow time.Duration
}

// ConfigFromEnv читает конфигурацию из переменных окружения.
//
// Переменные:
//
//	LOG_LEVEL          — минимальный уровень для файла: debug|info (default info)
//	LOG_DIR            — директория логов (default ./logs)
//	LOG_FILE_PREFIX    — префикс имени файла (default app)
//	LOG_COINS_THRESHOLD— минимальное количество реальных монет (default 95)
//	LOG_DB_ENABLED     — запись error/critical в БД: true|false (default false)
//	CLICKHOUSE_ADDR    — адрес ClickHouse (default 127.0.0.1:9000)
//	CLICKHOUSE_DB      — база данных (default analytics)
//	CLICKHOUSE_USER    — пользователь (default analytics)
//	CLICKHOUSE_PASSWORD— пароль (default analytics_ch)
//	LOG_DB_TABLE       — таблица логов (default app_logs)
//	LOG_TG_ENABLED     — уведомления в Telegram: true|false (default false)
//	LOG_TG_BOT_TOKEN   — токен бота
//	LOG_TG_CHAT_IDS    — получатели (ID чатов/каналов), через запятую
//	LOG_TG_DEDUP_MIN   — интервал дедупликации в минутах (default 30)
func ConfigFromEnv() Config {
	return Config{
		Level:          ParseLevel(envOr("LOG_LEVEL", "info")),
		LogDir:         envOr("LOG_DIR", "./logs"),
		FilePrefix:     envOr("LOG_FILE_PREFIX", "app"),
		CoinsThreshold: envInt("LOG_COINS_THRESHOLD", 95),
		DB: DBConfig{
			Enabled:  envBool("LOG_DB_ENABLED", false),
			Addr:     envOr("CLICKHOUSE_ADDR", "127.0.0.1:9000"),
			Database: envOr("CLICKHOUSE_DB", "analytics"),
			Username: envOr("CLICKHOUSE_USER", "analytics"),
			Password: envOr("CLICKHOUSE_PASSWORD", "analytics_ch"),
			Table:    envOr("LOG_DB_TABLE", "app_logs"),
		},
		Telegram: TelegramConfig{
			Enabled:     envBool("LOG_TG_ENABLED", false),
			BotToken:    os.Getenv("LOG_TG_BOT_TOKEN"),
			ChatIDs:     splitCSV(os.Getenv("LOG_TG_CHAT_IDS")),
			DedupWindow: time.Duration(envInt("LOG_TG_DEDUP_MIN", 30)) * time.Minute,
		},
	}
}

// envOr возвращает значение переменной окружения name или fallback, если она пуста.
func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

// envBool читает булеву переменную окружения.
func envBool(name string, fallback bool) bool {
	v := os.Getenv(name)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

// envInt читает целочисленную переменную окружения.
func envInt(name string, fallback int) int {
	v := os.Getenv(name)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

// splitCSV разбивает строку вида "a,b, c" на список без пустых элементов.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
