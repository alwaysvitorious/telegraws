package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type TelegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

func SendToTelegram(ctx context.Context, message string, botToken string, chatID string) error {
	telegramAPI := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	telegramMsg := TelegramMessage{
		ChatID:    chatID,
		Text:      message,
		ParseMode: "Markdown",
	}

	jsonData, err := json.Marshal(telegramMsg)
	if err != nil {
		return fmt.Errorf("error marshaling Telegram message: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", telegramAPI, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 40 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending telegram message: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned non-200 status: %d", resp.StatusCode)
	}

	return nil
}
