package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
)

const djinniJobsURL = "https://djinni.co/jobs/keyword-junior/"

// FetchDjinni парсить публічні junior-позиції з djinni.co
func FetchDjinni(ctx context.Context) ([]model.Listing, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	body, status, err := fetchURL(ctx, client, djinniJobsURL)
	if err != nil {
		return nil, fmt.Errorf("djinni: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("djinni: HTTP %d", status)
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("djinni parse: %w", err)
	}

	var out []model.Listing
	doc.Find(".list-jobs__item, [class*='job-item'], .job-list-item").Each(func(_ int, s *goquery.Selection) {
		titleLink := s.Find("a[href*='/jobs/']").First()
		title := strings.TrimSpace(titleLink.Text())
		href, _ := titleLink.Attr("href")
		if title == "" || href == "" {
			return
		}

		if !strings.HasPrefix(href, "http") {
			href = "https://djinni.co" + href
		}

		desc := strings.TrimSpace(s.Find(".job-list-item__description, .text-card, .readmore-text").First().Text())
		salaryText := strings.TrimSpace(s.Find(".public-salary-item, [class*='salary']").First().Text())

		var skills []string
		s.Find(".job-list-item__job-info .badge, .job-additional-info .badge").Each(func(_ int, b *goquery.Selection) {
			txt := strings.TrimSpace(b.Text())
			if txt != "" {
				skills = append(skills, txt)
			}
		})

		raw, _ := json.Marshal(map[string]any{
			"title":  title,
			"link":   href,
			"salary": salaryText,
			"skills": skills,
		})

		extID := href
		if parts := strings.Split(strings.Trim(href, "/"), "/"); len(parts) > 0 {
			extID = "djinni:" + parts[len(parts)-1]
		}

		out = append(out, model.Listing{
			Source:      "djinni",
			ExternalID:  extID,
			URL:         href,
			Title:       title,
			Description: desc,
			BudgetCents: parseBudgetCents(salaryText),
			Currency:    "USD",
			Skills:      skills,
			Raw:         raw,
		})
	})

	return out, nil
}
