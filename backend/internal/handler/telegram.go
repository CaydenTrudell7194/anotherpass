package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

var telegramClient = &http.Client{Timeout: 8 * time.Second}

func telegramBotToken() string { return strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")) }

func notifyTelegram(text string) {
	settings := LoadSiteSettings()
	token := telegramBotToken()
	if !settings.TelegramEnabled || token == "" || settings.TelegramChatID == "" {
		return
	}
	go func() {
		body, _ := json.Marshal(map[string]string{"chat_id": settings.TelegramChatID, "text": text})
		req, err := http.NewRequest(http.MethodPost, "https://api.telegram.org/bot"+token+"/sendMessage", bytes.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := telegramClient.Do(req)
		if err == nil {
			resp.Body.Close()
		}
	}()
}
