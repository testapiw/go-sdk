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

// AlertObserver — необязательный наблюдатель доставки алертов. Позволяет
// приложению фиксировать метрики (латентность, ошибки) в точке захвата —
// непосредственно вокруг сетевого вызова к API Telegram — без изменения
// сигнатур существующих функций и без передачи клиента метрик внутрь SDK.
//
// Реализация должна быть быстрой и неблокирующей: она вызывается
// синхронно после завершения сетевого вызова.
type AlertObserver interface {
	// ObserveAlertDelivery вызывается после попытки доставки алерта.
	// channel — канал доставки (например "telegram").
	// d — время от старта отправки до завершения сетевого вызова.
	// err — ошибка доставки (nil при успехе).
	ObserveAlertDelivery(channel string, d time.Duration, err error)
}

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

	// observer — необязательный наблюдатель доставки (метрики Фазы 3).
	observer AlertObserver
}

// SetAlertObserver устанавливает наблюдателя доставки алертов. Вызывается
// приложением после инициализации логгера; сигнатуры существующих функций
// не меняются. Передача nil отключает наблюдение.
func (t *telegramNotifier) SetAlertObserver(o AlertObserver) {
	if t == nil {
		return
	}
	t.observer = o
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
	// Фаза 3: точка захвата метрик доставки алерта. Отсчёт начинается
	// перед сетевым вызовом и завершается после получения ответа.
	start := time.Now()

	payload := map[string]string{
		"chat_id":    chatID,
		"text":       message,
		"parse_mode": "HTML",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.observe("telegram", time.Since(start), err)
		return false
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)
	resp, err := t.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.observe("telegram", time.Since(start), err)
		return false
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.observe("telegram", time.Since(start),
			fmt.Errorf("telegram API status %d", resp.StatusCode))
		return false
	}

	t.observe("telegram", time.Since(start), nil)
	return true
}

// observe уведомляет наблюдателя о результате доставки (если он задан).
func (t *telegramNotifier) observe(channel string, d time.Duration, err error) {
	if t == nil || t.observer == nil {
		return
	}
	t.observer.ObserveAlertDelivery(channel, d, err)
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
