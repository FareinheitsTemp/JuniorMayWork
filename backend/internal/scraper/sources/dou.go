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

const douVacanciesURL = "https://jobs.dou.ua/vacancies/?exp=0-1"

// FetchDOU парсить список вакансій для початківців із jobs.dou.ua
func FetchDOU(ctx context.Context) ([]model.Listing, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	body, status, err := fetchURL(ctx, client, douVacanciesURL)
	if err != nil {
		return nil, fmt.Errorf("dou: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("dou: HTTP %d", status)
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("dou parse: %w", err)
	}

	var out []model.Listing
	doc.Find("li.l-vacancy").Each(func(_ int, s *goquery.Selection) {
		titleLink := s.Find("a.vt").First()
		title := strings.TrimSpace(titleLink.Text())
		href, _ := titleLink.Attr("href")
		if title == "" || href == "" {
			return
		}

		company := strings.TrimSpace(s.Find("a.company").First().Text())
		cities := strings.TrimSpace(s.Find("span.cities").First().Text())
		desc := strings.TrimSpace(s.Find(".sh-info").First().Text())
		salaryText := strings.TrimSpace(s.Find("span.salary").First().Text())

		fullDesc := desc
		if company != "" || cities != "" {
			fullDesc = fmt.Sprintf("Компанія: %s | Локація: %s\n\n%s", company, cities, desc)
		}

		raw, _ := json.Marshal(map[string]string{
			"title":   title,
			"company": company,
			"cities":  cities,
			"link":    href,
			"salary":  salaryText,
		})

		extID := href
		if parts := strings.Split(strings.Trim(href, "/"), "/"); len(parts) > 0 {
			extID = "dou:" + parts[len(parts)-1]
		}

		out = append(out, model.Listing{
			Source:      "dou",
			ExternalID:  extID,
			URL:         href,
			Title:       title,
			Description: fullDesc,
			BudgetCents: parseBudgetCents(salaryText),
			Currency:    "USD",
			Raw:         raw,
		})
	})

	return out, nil
}
