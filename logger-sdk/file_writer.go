package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// fileWriter пишет логи в файл с ротацией по дням.
//
// Имя файла: <LogDir>/<FilePrefix>-YYYY-MM-DD.log
// При наступлении нового дня автоматически создаётся новый файл.
type fileWriter struct {
	mu       sync.Mutex
	dir      string
	prefix   string
	file     *os.File
	curDate  string // текущая дата файла (YYYY-MM-DD)
	lastErr  error  // последняя ошибка записи (для диагностики)
}

// newFileWriter создаёт писатель в файл с ротацией по дням.
func newFileWriter(dir, prefix string) (*fileWriter, error) {
	if dir == "" {
		dir = "./logs"
	}
	if prefix == "" {
		prefix = "app"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("logger: create log dir: %w", err)
	}
	w := &fileWriter{dir: dir, prefix: prefix}
	if err := w.rotate(); err != nil {
		return nil, err
	}
	return w, nil
}

// rotate открывает файл для текущей даты (если дата изменилась — новый файл).
func (w *fileWriter) rotate() error {
	date := time.Now().Format("2006-01-02")
	if w.file != nil && w.curDate == date {
		return nil
	}
	if w.file != nil {
		_ = w.file.Close()
		w.file = nil
	}
	path := filepath.Join(w.dir, fmt.Sprintf("%s-%s.log", w.prefix, date))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("logger: open log file %s: %w", path, err)
	}
	w.file = f
	w.curDate = date
	return nil
}

// write записывает строку в файл, при необходимости ротируя по дате.
func (w *fileWriter) write(line string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.rotate(); err != nil {
		w.lastErr = err
		return
	}
	if _, err := w.file.WriteString(line); err != nil {
		w.lastErr = err
	}
}

// close закрывает файл.
func (w *fileWriter) close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		err := w.file.Close()
		w.file = nil
		return err
	}
	return nil
}
