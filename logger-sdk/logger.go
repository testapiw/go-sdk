package logger

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Logger — основной логгер с уровнями сообщений.
//
// Поведение:
//   - debug, info — пишутся в файл с ротацией по дням (включаются по требованию через .env)
//   - error, critical — пишутся всегда, в базу данных (ClickHouse)
//   - critical — дополнительно может отправлять уведомление в Telegram
//
// Использование:
//
//	l := logger.New(logger.ConfigFromEnv())
//	defer l.Close()
//
//	l.Info("service started")
//	l.Debug("request", map[string]any{"path": "/api"})
//	l.Error("failed to connect", err)
//	l.Critical("database down", map[string]any{"host": "..."}, true) // true — уведомить в Telegram
type Logger struct {
	cfg       Config
	file      *fileWriter
	db        *dbWriter
	telegram  *telegramNotifier
	mu        sync.Mutex
	closed    bool
}

// New создаёт логгер из конфигурации.
func New(cfg Config) (*Logger, error) {
	l := &Logger{cfg: cfg}

	// Файловый писатель (info/debug) — всегда создаём, даже если уровень
	// выше info, чтобы можно было включить debug без перезапуска.
	fw, err := newFileWriter(cfg.LogDir, cfg.FilePrefix)
	if err != nil {
		return nil, err
	}
	l.file = fw

	// Писатель в БД (error/critical) — только если включён.
	if cfg.DB.Enabled {
		dw, err := newDBWriter(cfg.DB)
		if err != nil {
			// Не фатально — логируем в файл и продолжаем без БД.
			fw.write(formatLine(LevelError, "logger: db writer init failed: "+err.Error(), "", time.Now()))
		} else {
			l.db = dw
		}
	}

	// Уведомитель в Telegram (critical).
	l.telegram = newTelegramNotifier(cfg.Telegram)

	return l, nil
}

// Debug пишет отладочное сообщение в файл (если уровень debug включён).
func (l *Logger) Debug(msg string, data ...any) {
	l.log(LevelDebug, msg, data...)
}

// Info пишет информационное сообщение в файл (если уровень info включён).
func (l *Logger) Info(msg string, data ...any) {
	l.log(LevelInfo, msg, data...)
}

// Error пишет ошибку всегда: в файл и в базу данных.
func (l *Logger) Error(msg string, data ...any) {
	l.log(LevelError, msg, data...)
}

// Critical пишет критическую ошибку всегда: в файл и в базу данных.
// Если notify == true — дополнительно отправляет уведомление в Telegram
// (с дедупликацией: не чаще одного раза за 30 минут на одно сообщение).
func (l *Logger) Critical(msg string, data ...any) {
	l.log(LevelCritical, msg, data...)
}

// CriticalNotify — то же, что Critical, но с уведомлением в Telegram.
func (l *Logger) CriticalNotify(msg string, data ...any) {
	l.log(LevelCritical, msg, data...)
	l.notify(msg, data...)
}

// CoinsMissing проверяет, получено ли меньше минимального количества монет
// (CoinsThreshold, по умолчанию 95). Если да — пишет critical
// с уведомлением в Telegram. Возвращает true, если порог нарушен.
//
// CoinsThreshold — это количество реальных монет, а не процент.
// Например, если ожидалось 100 монет, а получено меньше 95 — critical.
//
// Использование:
//
//	if l.CoinsMissing("coins fetch", group, received, expected) {
//	    // порог нарушен — уже залогировано и уведомлено
//	}
func (l *Logger) CoinsMissing(what string, group int, received, expected int) bool {
	threshold := l.cfg.CoinsThreshold
	if threshold <= 0 {
		threshold = 95
	}
	if expected <= 0 {
		return false
	}
	// Получено меньше минимального количества монет.
	if received < threshold {
		l.CriticalNotify("not all coins received",
			map[string]any{
				"what":      what,
				"group":     group,
				"received":  received,
				"expected":  expected,
				"threshold": threshold,
			})
		return true
	}
	return false
}

// log — внутренняя реализация записи сообщения.
func (l *Logger) log(level Level, msg string, data ...any) {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return
	}
	l.mu.Unlock()

	// Сериализуем дополнительные данные (если есть).
	dataJSON := ""
	if len(data) > 0 {
		if b, err := json.Marshal(data); err == nil {
			dataJSON = string(b)
		}
	}

	now := time.Now()

	// error/critical — всегда в файл и в БД.
	if level >= LevelError {
		if l.file != nil {
			l.file.write(formatLine(level, msg, dataJSON, now))
		}
		if l.db != nil {
			l.db.write(level.String(), msg, dataJSON)
		}
		return
	}

	// info/debug — в файл, только если уровень включён.
	if level >= l.cfg.Level && l.file != nil {
		l.file.write(formatLine(level, msg, dataJSON, now))
	}
}

// notify отправляет уведомление в Telegram (с дедупликацией).
func (l *Logger) notify(msg string, data ...any) {
	if l.telegram == nil {
		return
	}
	text := msg
	if len(data) > 0 {
		if b, err := json.Marshal(data); err == nil {
			text = fmt.Sprintf("%s\n%s", msg, string(b))
		}
	}
	l.telegram.notify(text)
}

// Close закрывает все писатели.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil
	}
	l.closed = true

	var firstErr error
	if l.file != nil {
		if err := l.file.close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if l.db != nil {
		l.db.close()
	}
	return firstErr
}

// formatLine форматирует строку лога.
func formatLine(level Level, msg, data string, t time.Time) string {
	line := fmt.Sprintf("%s [%s] %s", t.Format("2006-01-02 15:04:05.000"), level.String(), msg)
	if data != "" {
		line += " " + data
	}
	return line + "\n"
}
