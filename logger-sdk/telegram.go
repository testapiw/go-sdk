package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// telegramNotifier отправляет уведомления о critical-сообщениях в Telegram.
//
// Дедупликация: одно и то же сообщение (по тексту) не отправляется повторно
// в течение DedupWindow (по умолчанию 30 минут). Это защищает от спама
// при повторяющихся ошибках.
type telegramNotifier struct {
	botToken string
	chatIDs  []string
	enabled  bool

	mu       sync.Mutex
	lastSent map[string]time.Time // message -> время последней отправки
	dedup    time.Duration
	client   *http.Client
}

// newTelegramNotifier создаёт уведомитель. Если не включён — no-op.
func newTelegramNotifier(cfg TelegramConfig) *telegramNotifier {
	dedup := cfg.DedupWindow
	if dedup <= 0 {
		dedup = 30 * time.Minute
	}
	return &telegramNotifier{
		botToken: cfg.BotToken,
		chatIDs:  cfg.ChatIDs,
		enabled:  cfg.Enabled && cfg.BotToken != "" && len(cfg.ChatIDs) > 0,
		lastSent: make(map[string]time.Time),
		dedup:    dedup,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

// notify отправляет уведомление всем получателям, если оно не было
// отправлено недавно. Возвращает true, если уведомление отправлено
// (или пропущено из-за дедупликации).
func (t *telegramNotifier) notify(message string) bool {
	if t == nil || !t.enabled {
		return false
	}

	// Дедупликация.
	t.mu.Lock()
	last, ok := t.lastSent[message]
	now := time.Now()
	if ok && now.Sub(last) < t.dedup {
		t.mu.Unlock()
		return false // недавно отправляли — пропускаем
	}
	t.lastSent[message] = now
	t.mu.Unlock()

	// Очистка старых записей (чтобы map не рос бесконечно).
	t.cleanup(now)

	// Отправка всем получателям.
	sent := false
	for _, chatID := range t.chatIDs {
		if t.send(chatID, message) {
			sent = true
		}
	}
	return sent
}

// send отправляет одно сообщение одному получателю.
func (t *telegramNotifier) send(chatID, message string) bool {
	payload := map[string]string{
		"chat_id":    chatID,
		"text":       message,
		"parse_mode": "HTML",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return false
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)
	resp, err := t.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// cleanup удаляет записи старше dedup*2, чтобы map не рос бесконечно.
func (t *telegramNotifier) cleanup(now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for msg, ts := range t.lastSent {
		if now.Sub(ts) > t.dedup*2 {
			delete(t.lastSent, msg)
		}
	}
}
