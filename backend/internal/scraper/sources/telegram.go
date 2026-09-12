// Джерело: Telegram-канали через публічні веб-дзеркала t.me/s/<канал>.
// Канали задаються через JMW_TELEGRAM_CHANNELS (через кому, без @).
package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
)

var uahRe = regexp.MustCompile(`(\d[\d\s\x{00a0},]*)\s*(?:грн|₴|UAH)`)

// parseUAHBudgetCents шукає суму в гривнях і конвертує в USD-центи.
func parseUAHBudgetCents(text string) *int {
	m := uahRe.FindStringSubmatch(text)
	if m == nil {
		return nil
	}
	return uahToUsdCents(m[1])
}

func telegramChannels() []string {
	raw := os.Getenv("JMW_TELEGRAM_CHANNELS")
	if raw == "" {
		raw = "freelance_for_ukraine"
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimPrefix(strings.TrimSpace(p), "@"); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func FetchTelegram(ctx context.Context) ([]model.Listing, error) {
	channels := telegramChannels()
	client := &http.Client{Timeout: 20 * time.Second}
	var out []model.Listing
	for _, ch := range channels {
		body, status, err := fetchURL(ctx, client, "https://t.me/s/"+ch)
		if err != nil {
			return nil, fmt.Errorf("telegram @%s: %w", ch, err)
		}
		if status != http.StatusOK {
			return nil, fmt.Errorf("telegram @%s: HTTP %d", ch, status)
		}
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
		if err != nil {
			return nil, fmt.Errorf("telegram @%s: html: %w", ch, err)
		}
		// <br> → перенос рядка, щоб перший рядок став заголовком
		doc.Find("br").Each(func(_ int, br *goquery.Selection) {
			br.ReplaceWithHtml("\n")
		})
		doc.Find(".tgme_widget_message").Each(func(_ int, s *goquery.Selection) {
			post, ok := s.Attr("data-post")
			if !ok {
				return
			}
			text := strings.TrimSpace(s.Find(".tgme_widget_message_text").First().Text())
			if text == "" {
				return
			}
			href, _ := s.Find(".tgme_widget_message_date").First().Attr("href")
			if href == "" {
				href = "https://t.me/" + post
			}
			title := text
			if i := strings.IndexByte(title, '\n'); i > 0 {
				title = title[:i]
			}
			if len(title) > 120 {
				title = title[:120] + "…"
			}
			budget := parseUAHBudgetCents(text)
			if budget == nil {
				budget = parseBudgetCents(text)
			}
			raw, _ := json.Marshal(map[string]string{"channel": ch, "post": post})
			out = append(out, model.Listing{
				ExternalID:  "tg:" + post,
				URL:         href,
				Title:       title,
				Description: text,
				BudgetCents: budget,
				Currency:    "USD",
				Raw:         raw,
			})
		})
	}
	return out, nil
}
