// Джерело: Reddit — r/slavelabour і r/forhire (публічний JSON; гігі до $100).
package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
)

var redditSubs = []string{"slavelabour", "forhire"}

type redditFeed struct {
	Data struct {
		Children []struct {
			Data struct {
				ID        string  `json:"id"`
				Title     string  `json:"title"`
				Selftext  string  `json:"selftext"`
				Permalink string  `json:"permalink"`
				Author    string  `json:"author"`
				Flair     string  `json:"link_flair_text"`
				Created  float64 `json:"created_utc"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

func FetchReddit(ctx context.Context) ([]model.Listing, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	var out []model.Listing
	for _, sub := range redditSubs {
		apiURL := fmt.Sprintf("https://www.reddit.com/r/%s/new.json?limit=50&raw_json=1", sub)
		body, status, err := fetchURL(ctx, client, apiURL)
		if err != nil {
			return nil, fmt.Errorf("reddit r/%s: %w", sub, err)
		}
		if status != http.StatusOK {
			return nil, fmt.Errorf("reddit r/%s: HTTP %d", sub, status)
		}
		var feed redditFeed
		if err := json.Unmarshal(body, &feed); err != nil {
			return nil, fmt.Errorf("reddit r/%s: json: %w", sub, err)
		}
		for _, c := range feed.Data.Children {
			d := c.Data
			if d.ID == "" {
				continue
			}
			raw, _ := json.Marshal(d)
			out = append(out, model.Listing{
				ExternalID:  "r/" + sub + ":" + d.ID,
				URL:         "https://www.reddit.com" + d.Permalink,
				Title:       d.Title,
				Description: d.Selftext,
				BudgetCents: parseBudgetCents(d.Title + " " + d.Selftext),
				Currency:    "USD",
				Raw:         raw,
			})
		}
	}
	return out, nil
}
