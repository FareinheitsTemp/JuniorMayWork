// Джерело: Upwork RSS-пошук (публічний, без ключів).
package sources

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
)

const upworkFeedURL = "https://www.upwork.com/ab/feed/jobs/rss"

const browserUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"

// Пошукові фрази: дрібні React/JS-гігі.
var upworkQueries = []string{"react", "javascript", "next.js"}

var budgetRe = regexp.MustCompile(`\$\s*([\d][\d,]*)(?:\s*(?:-|–|to)\s*\$?\s*([\d][\d,]*))?`)

// parseBudgetCents дістає бюджет з тексту ("$50", "$30-$100", "Budget: $75").
// Для діапазону беремо верхню межу. Немає збігу — nil (бюджет невідомий).
func parseBudgetCents(text string) *int {
	m := budgetRe.FindStringSubmatch(text)
	if m == nil {
		return nil
	}
	raw := m[2]
	if raw == "" {
		raw = m[1]
	}
	v, err := strconv.Atoi(strings.ReplaceAll(raw, ",", ""))
	if err != nil {
		return nil
	}
	cents := v * 100
	return &cents
}

// fetchURL: GET із browser-UA і обмеженням розміру відповіді.
func fetchURL(ctx context.Context, client *http.Client, rawURL string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", browserUA)
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func externalIDFromLink(link string) string {
	id := link
	if i := strings.IndexByte(id, '?'); i >= 0 {
		id = id[:i]
	}
	return id
}

type rssFeed struct {
	Channel struct {
		Items []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			Description string `xml:"description"`
			PubDate     string `xml:"pubDate"`
		} `xml:"item"`
	} `xml:"channel"`
}

func FetchUpwork(ctx context.Context) ([]model.Listing, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	var out []model.Listing
	seen := map[string]bool{}
	for _, q := range upworkQueries {
		feedURL := fmt.Sprintf("%s?q=%s&sort=recency", upworkFeedURL, url.QueryEscape(q))
		body, status, err := fetchURL(ctx, client, feedURL)
		if err != nil {
			return nil, fmt.Errorf("upwork %q: %w", q, err)
		}
		if status != http.StatusOK {
			return nil, fmt.Errorf("upwork %q: HTTP %d", q, status)
		}
		var feed rssFeed
		if err := xml.Unmarshal(body, &feed); err != nil {
			return nil, fmt.Errorf("upwork %q: rss parse: %w", q, err)
		}
		for _, item := range feed.Channel.Items {
			link := strings.TrimSpace(item.Link)
			if link == "" || seen[link] {
				continue
			}
			seen[link] = true
			raw, _ := json.Marshal(item)
			out = append(out, model.Listing{
				ExternalID:  externalIDFromLink(link),
				URL:         link,
				Title:       strings.TrimSpace(item.Title),
				Description: item.Description,
				BudgetCents: parseBudgetCents(item.Title + " " + item.Description),
				Currency:    "USD",
				Raw:         raw,
			})
		}
	}
	return out, nil
}
