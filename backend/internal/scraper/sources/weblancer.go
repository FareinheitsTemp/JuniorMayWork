// Джерело: Weblancer (weblancer.net/jobs) — HTML через goquery.
// Верстка сайту мінялась, тому пробуємо кілька кандидатів-селекторів;
// якщо жоден не спрацював — повертаємо порожній список (менеджер просто залогує).
package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
)

const weblancerJobsURL = "https://www.weblancer.net/jobs/"

var weblancerCards = []string{
	".job_row",
	".job-item",
	".job_item",
	"[class*='job_row']",
}

func FetchWeblancer(ctx context.Context) ([]model.Listing, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	body, status, err := fetchURL(ctx, client, weblancerJobsURL)
	if err != nil {
		return nil, fmt.Errorf("weblancer: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("weblancer: HTTP %d", status)
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("weblancer: html: %w", err)
	}
	var out []model.Listing
	for _, sel := range weblancerCards {
		doc.Find(sel).Each(func(_ int, s *goquery.Selection) {
			titleLink := s.Find("a[href*='/jobs/']").First()
			href, ok := titleLink.Attr("href")
			if !ok {
				return
			}
			title := strings.TrimSpace(titleLink.Text())
			if title == "" {
				return
			}
			desc := strings.TrimSpace(s.Find(".text, .job_text, .description").First().Text())
			raw, _ := json.Marshal(map[string]string{"title": title, "link": href})
			out = append(out, model.Listing{
				ExternalID:  externalIDFromLink(href),
				URL:         weblancerAbsURL(href),
				Title:       title,
				Description: desc,
				BudgetCents: parseBudgetCents(s.Text()),
				Currency:    "USD",
				Skills:      extractWeblancerSkills(s),
				Raw:         raw,
			})
		})
		if len(out) > 0 {
			break
		}
	}
	return out, nil
}

func weblancerAbsURL(href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	return "https://www.weblancer.net" + href
}

func extractWeblancerSkills(s *goquery.Selection) []string {
	skills := []string{}
	s.Find(".skill, .tag, .skills span").Each(func(_ int, el *goquery.Selection) {
		if t := strings.TrimSpace(el.Text()); t != "" {
			skills = append(skills, t)
		}
	})
	return skills
}
