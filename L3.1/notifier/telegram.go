package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	appErrors "L3.1/errors"
	"L3.1/types"
)

// Telegram реализует отправку уведомлений через Telegram Bot API.
type Telegram struct {
	token  string
	chatID string
	client *http.Client
}

// NewTelegram создаёт новый Telegram-уведомитель с указанным токеном,
// целевым chatID и HTTP-таймаутом.
func NewTelegram(token, chatID string, timeout time.Duration) *Telegram {
	return &Telegram{
		token:  token,
		chatID: chatID,
		client: &http.Client{Timeout: timeout},
	}
}

// Send отправляет сообщение в Telegram через Bot API.
// Возвращает ErrTemporary при сетевых/временных сбоях,
// ErrFatal — при необратимых ошибках (например: неверный чат или токен).
func (t *Telegram) Send(ctx context.Context, ntf types.Notification) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.token)

	log.Printf("[TELEGRAM] Sending message to chat %s: %q", t.chatID, ntf.Message)

	payload := map[string]interface{}{
		"chat_id":    t.chatID,
		"text":       ntf.Message,
		"parse_mode": "Markdown",
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		log.Printf("[TELEGRAM] ERROR building request: %v", err)
		return fmt.Errorf("%w: build request failed: %v", appErrors.ErrTemporary, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		log.Printf("[TELEGRAM] HTTP request FAILED: %v", err)
		return fmt.Errorf("%w: http request failed: %v", appErrors.ErrTemporary, err)
	}
	defer resp.Body.Close()

	log.Printf("[TELEGRAM] Response status: %d %s", resp.StatusCode, resp.Status)

	var tr struct {
		OK          bool            `json:"ok"`
		Result      json.RawMessage `json:"result"`
		Description interface{}     `json:"description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		log.Printf("[TELEGRAM] ERROR decoding response: %v", err)
		return fmt.Errorf("%w: decode failed: %v", appErrors.ErrTemporary, err)
	}

	log.Printf("[TELEGRAM] Telegram returned OK=%v, description=%v", tr.OK, tr.Description)

	if resp.StatusCode >= 500 {
		log.Printf("[TELEGRAM] Server ERROR %d", resp.StatusCode)
		return fmt.Errorf("%w: telegram server error %d", appErrors.ErrTemporary, resp.StatusCode)
	}

	if resp.StatusCode >= 400 {
		log.Printf("[TELEGRAM] FATAL API ERROR %d: %v", resp.StatusCode, tr.Description)
		return fmt.Errorf("%w: telegram fatal %d: %v", appErrors.ErrFatal, resp.StatusCode, tr.Description)
	}

	if !tr.OK {
		log.Printf("[TELEGRAM] Unexpected response from Telegram: %v", tr.Description)
		return fmt.Errorf("%w: unexpected response: %v", appErrors.ErrTemporary, tr.Description)
	}

	log.Printf("[TELEGRAM] Message delivered successfully.")
	return nil
}
