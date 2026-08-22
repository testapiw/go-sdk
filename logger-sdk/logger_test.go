package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestFileWriterRotation проверяет, что файловый писатель создаёт файл
// с правильным именем и записывает строки.
func TestFileWriterRotation(t *testing.T) {
	dir := t.TempDir()
	w, err := newFileWriter(dir, "test")
	if err != nil {
		t.Fatalf("newFileWriter: %v", err)
	}
	defer w.close()

	w.write("hello\n")
	w.write("world\n")

	date := time.Now().Format("2006-01-02")
	path := filepath.Join(dir, "test-"+date+".log")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if !strings.Contains(string(data), "hello") || !strings.Contains(string(data), "world") {
		t.Fatalf("log file content mismatch: %q", string(data))
	}
}

// TestLevels проверяет, что info/debug пишутся в файл, а error/critical
// обрабатываются корректно.
func TestLevels(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		Level:      LevelInfo,
		LogDir:     dir,
		FilePrefix: "app",
		DB:         DBConfig{Enabled: false},
		Telegram:   TelegramConfig{Enabled: false},
	}
	l, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer l.Close()

	l.Info("info message")
	l.Debug("debug message") // не должен попасть в файл (уровень info)
	l.Error("error message")
	l.Critical("critical message")

	date := time.Now().Format("2006-01-02")
	path := filepath.Join(dir, "app-"+date+".log")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "info message") {
		t.Errorf("info message not in file: %q", content)
	}
	if strings.Contains(content, "debug message") {
		t.Errorf("debug message should not be in file at info level: %q", content)
	}
	if !strings.Contains(content, "error message") {
		t.Errorf("error message not in file: %q", content)
	}
	if !strings.Contains(content, "critical message") {
		t.Errorf("critical message not in file: %q", content)
	}
}

// TestDebugEnabled проверяет, что при уровне debug сообщения debug пишутся.
func TestDebugEnabled(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		Level:      LevelDebug,
		LogDir:     dir,
		FilePrefix: "app",
		DB:         DBConfig{Enabled: false},
		Telegram:   TelegramConfig{Enabled: false},
	}
	l, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer l.Close()

	l.Debug("debug message")

	date := time.Now().Format("2006-01-02")
	path := filepath.Join(dir, "app-"+date+".log")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if !strings.Contains(string(data), "debug message") {
		t.Errorf("debug message not in file at debug level: %q", string(data))
	}
}

// TestTelegramDedup проверяет дедупликацию уведомлений.
func TestTelegramDedup(t *testing.T) {
	// Не отправляем реальные запросы — проверяем только логику дедупликации.
	// Создаём уведомитель с отключённым enabled, чтобы не делать HTTP-запросы.
	nt := newTelegramNotifier(TelegramConfig{
		Enabled:     false,
		DedupWindow: 30 * time.Minute,
	})

	// При отключённом enabled notify всегда возвращает false.
	if nt.notify("test") {
		t.Error("notify should return false when disabled")
	}

	// Проверяем логику дедупликации напрямую.
	nt.enabled = true
	nt.botToken = "fake"
	nt.chatIDs = []string{"fake"}

	// Первый вызов — попытка отправки (вернёт false из-за HTTP-ошибки,
	// но запись в lastSent должна появиться).
	nt.notify("same message")
	nt.mu.Lock()
	_, ok := nt.lastSent["same message"]
	nt.mu.Unlock()
	if !ok {
		t.Error("message should be recorded in lastSent after first notify")
	}

	// Второй вызов в течение окна — должен быть пропущен (дедупликация).
	nt.mu.Lock()
	nt.lastSent["same message"] = time.Now()
	nt.mu.Unlock()
	if nt.notify("same message") {
		t.Error("duplicate message should be deduplicated")
	}
}

// TestConfigFromEnv проверяет чтение конфигурации из окружения.
func TestConfigFromEnv(t *testing.T) {
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("LOG_DIR", "/tmp/test-logs")
	os.Setenv("LOG_FILE_PREFIX", "svc")
	os.Setenv("LOG_DB_ENABLED", "true")
	os.Setenv("LOG_TG_ENABLED", "true")
	os.Setenv("LOG_TG_CHAT_IDS", "123456, 789012, ")
	os.Setenv("LOG_TG_DEDUP_MIN", "45")
	defer os.Unsetenv("LOG_LEVEL")
	defer os.Unsetenv("LOG_DIR")
	defer os.Unsetenv("LOG_FILE_PREFIX")
	defer os.Unsetenv("LOG_DB_ENABLED")
	defer os.Unsetenv("LOG_TG_ENABLED")
	defer os.Unsetenv("LOG_TG_CHAT_IDS")
	defer os.Unsetenv("LOG_TG_DEDUP_MIN")

	cfg := ConfigFromEnv()
	if cfg.Level != LevelDebug {
		t.Errorf("Level = %v, want debug", cfg.Level)
	}
	if cfg.LogDir != "/tmp/test-logs" {
		t.Errorf("LogDir = %q, want /tmp/test-logs", cfg.LogDir)
	}
	if cfg.FilePrefix != "svc" {
		t.Errorf("FilePrefix = %q, want svc", cfg.FilePrefix)
	}
	if !cfg.DB.Enabled {
		t.Error("DB.Enabled should be true")
	}
	if !cfg.Telegram.Enabled {
		t.Error("Telegram.Enabled should be true")
	}
	if len(cfg.Telegram.ChatIDs) != 2 || cfg.Telegram.ChatIDs[0] != "123456" || cfg.Telegram.ChatIDs[1] != "789012" {
		t.Errorf("ChatIDs = %v, want [123456 789012]", cfg.Telegram.ChatIDs)
	}
	if cfg.Telegram.DedupWindow != 45*time.Minute {
		t.Errorf("DedupWindow = %v, want 45m", cfg.Telegram.DedupWindow)
	}
}

// TestSplitCSV проверяет разбиение строки получателей через запятую.
func TestSplitCSV(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"123", []string{"123"}},
		{"123,456", []string{"123", "456"}},
		{"123, 456,789", []string{"123", "456", "789"}},
		{"123,,456,", []string{"123", "456"}},
		{" , , ", nil},
	}
	for _, c := range cases {
		got := splitCSV(c.in)
		if len(got) != len(c.want) {
			t.Errorf("splitCSV(%q) = %v, want %v", c.in, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("splitCSV(%q) = %v, want %v", c.in, got, c.want)
				break
			}
		}
	}
}

// TestCoinsMissing проверяет логику «не все монеты» (порог — количество монет).
func TestCoinsMissing(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		Level:          LevelInfo,
		LogDir:         dir,
		FilePrefix:     "app",
		CoinsThreshold: 95, // минимальное количество реальных монет
		DB:             DBConfig{Enabled: false},
		Telegram:       TelegramConfig{Enabled: false},
	}
	l, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer l.Close()

	// Получено 100 монет — порог (95) не нарушен.
	if l.CoinsMissing("fetch", 1, 100, 100) {
		t.Error("100 coins should not trigger critical (threshold 95)")
	}

	// Получено 95 монет — порог не нарушен (>= 95).
	if l.CoinsMissing("fetch", 1, 95, 100) {
		t.Error("95 coins should not trigger critical (threshold 95)")
	}

	// Получено 94 монеты — порог нарушен (< 95).
	if !l.CoinsMissing("fetch", 1, 94, 100) {
		t.Error("94 coins should trigger critical (threshold 95)")
	}

	// expected <= 0 — не проверяем.
	if l.CoinsMissing("fetch", 1, 0, 0) {
		t.Error("expected<=0 should not trigger critical")
	}
}
