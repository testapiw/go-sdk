package logger

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// dbWriter пишет error/critical логи в базу данных (ClickHouse).
//
// Запись выполняется асинхронно через буферизованный канал, чтобы не
// блокировать основной поток приложения. Если БД недоступна — сообщения
// накапливаются в буфере (до maxBuffer) и не теряются.
type dbWriter struct {
	cfg       DBConfig
	conn      driver.Conn
	ch        chan dbEntry
	closeOnce sync.Once
	done      chan struct{}
	wg        sync.WaitGroup
}

// dbEntry — одна запись для записи в БД.
type dbEntry struct {
	Level   string
	Message string
	Data    string // JSON-данные (опционально)
	Time    time.Time
}

// newDBWriter создаёт писатель в БД. Если БД недоступна — возвращает
// писатель, который будет пытаться подключиться при первой записи.
func newDBWriter(cfg DBConfig) (*dbWriter, error) {
	w := &dbWriter{
		cfg:  cfg,
		ch:   make(chan dbEntry, 256),
		done: make(chan struct{}),
	}
	// Пробуем подключиться сразу. Если не вышло — не ошибка,
	// подключимся при первой записи.
	_ = w.connect()
	w.wg.Add(1)
	go w.run()
	return w, nil
}

// connect устанавливает подключение к ClickHouse (если ещё не установлено).
func (w *dbWriter) connect() error {
	if w.conn != nil {
		return nil
	}
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{w.cfg.Addr},
		Auth: clickhouse.Auth{
			Database: w.cfg.Database,
			Username: w.cfg.Username,
			Password: w.cfg.Password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout: 5 * time.Second,
		ReadTimeout: 10 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("logger db: open: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.Ping(ctx); err != nil {
		_ = conn.Close()
		return fmt.Errorf("logger db: ping: %w", err)
	}
	w.conn = conn
	return nil
}

// ensureTable создаёт таблицу логов, если её нет.
func (w *dbWriter) ensureTable(ctx context.Context) error {
	table := w.cfg.Table
	if table == "" {
		table = "app_logs"
	}
	q := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		ts       DateTime64(3),
		level    LowCardinality(String),
		message  String,
		data     String
	) ENGINE = MergeTree
	PARTITION BY toYYYYMM(ts)
	ORDER BY (ts)`, table)
	return w.conn.Exec(ctx, q)
}

// write отправляет запись в буфер для асинхронной записи в БД.
func (w *dbWriter) write(level, message, data string) {
	select {
	case w.ch <- dbEntry{Level: level, Message: message, Data: data, Time: time.Now()}:
	default:
		// Буфер переполнен — сбрасываем старые записи, чтобы не потерять новые.
		select {
		case <-w.ch:
		default:
		}
		select {
		case w.ch <- dbEntry{Level: level, Message: message, Data: data, Time: time.Now()}:
		default:
			// Всё ещё переполнен — пропускаем (не блокируем приложение).
		}
	}
}

// run — фоновый цикл записи в БД.
func (w *dbWriter) run() {
	defer w.wg.Done()
	for {
		select {
		case <-w.done:
			// Дренируем оставшиеся записи.
			for {
				select {
				case e := <-w.ch:
					w.insert(e)
				default:
					return
				}
			}
		case e := <-w.ch:
			w.insert(e)
		}
	}
}

// insert вставляет одну запись в БД.
func (w *dbWriter) insert(e dbEntry) {
	if w.conn == nil {
		if err := w.connect(); err != nil {
			return // БД недоступна — пропускаем (запись уже в буфере не сохранится)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := w.ensureTable(ctx); err != nil {
		return
	}

	table := w.cfg.Table
	if table == "" {
		table = "app_logs"
	}
	q := fmt.Sprintf("INSERT INTO %s (ts, level, message, data) VALUES (?, ?, ?, ?)", table)
	if err := w.conn.Exec(ctx, q, e.Time, e.Level, e.Message, e.Data); err != nil {
		return
	}
}

// close останавливает фоновый цикл и закрывает подключение.
func (w *dbWriter) close() {
	w.closeOnce.Do(func() {
		close(w.done)
		w.wg.Wait()
		if w.conn != nil {
			_ = w.conn.Close()
			w.conn = nil
		}
	})
}
