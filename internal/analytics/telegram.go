package analytics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Telegram struct {
	token  string
	chatID string
}

func NewTelegram(token, chatID string) *Telegram {
	return &Telegram{token: token, chatID: chatID}
}

func (t *Telegram) Send(msg string) {
	if t.token == "" || t.chatID == "" {
		return
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.token)

	payload := map[string]string{
		"chat_id":    t.chatID,
		"text":       msg,
		"parse_mode": "Markdown",
	}

	data, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("telegram send error: %v", err)
		return
	}
	defer resp.Body.Close()
}

func (t *Telegram) Notify(event Event) {
	var msg string

	switch event.Type {
	case "install_copy":
		msg = "📋 *Install command copied*"
	case "install_download":
		msg = "⬇️ *Install script downloaded*"
	case "github_click":
		msg = "⭐ *GitHub link clicked*"
	default:
		return
	}

	if event.Referrer != "" {
		msg += fmt.Sprintf("\nreferrer: `%s`", event.Referrer)
	}

	if event.Country != "" {
		msg += fmt.Sprintf("\ncountry: `%s`", event.Country)
	}

	t.Send(msg)
}

func (t *Telegram) SendStats(stats *Stats) {
	msg := "📊 *steered analytics*\n\n"

	msg += "*events*\n"
	msg += fmt.Sprintf("page views: `%d`\n", stats.EventCounts["page_view"])
	msg += fmt.Sprintf("install copied: `%d`\n", stats.EventCounts["install_copy"])
	msg += fmt.Sprintf("install downloaded: `%d`\n", stats.EventCounts["install_download"])
	msg += fmt.Sprintf("github clicks: `%d`\n", stats.EventCounts["github_click"])
	msg += fmt.Sprintf("video plays: `%d`\n", stats.EventCounts["video_play"])

	if len(stats.TopReferrers) > 0 {
		msg += "\n*top referrers*\n"
		for k, v := range stats.TopReferrers {
			msg += fmt.Sprintf("`%s`: %d\n", k, v)
		}
	}

	if len(stats.TopCountries) > 0 {
		msg += "\n*top countries*\n"
		for k, v := range stats.TopCountries {
			msg += fmt.Sprintf("`%s`: %d\n", k, v)
		}
	}

	t.Send(msg)
}

func (t *Telegram) Verify() {
	if t.token == "" || t.chatID == "" {
		log.Println("telegram: disabled — token or chat ID missing")
		return
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/getMe", t.token)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("telegram: connection failed — %v", err)
		return
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			Username string `json:"username"`
		} `json:"result"`
	}

	json.NewDecoder(resp.Body).Decode(&result)

	if result.OK {
		log.Printf("telegram: connected — @%s", result.Result.Username)
	} else {
		log.Println("telegram: connection failed — invalid token")
	}
}
