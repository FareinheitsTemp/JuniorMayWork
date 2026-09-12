// Джерело: Kwork (kwork.ru/projects) — HTML через goquery.
// Публічна лента замовлень «Хочу, щоб…» — якраз дрібні гігі.
package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
)

const kworkProjectsURL = "https://kwork.ru/projects"

// Орієнтовний курс конвертації рублів у долари (грубо, 2026).
const kworkRubPerUsd = 100

// Кандидати-контейнери картки (верстка Kwork мінялась час від часу).
var kworkCards = []string{
	".want-card",
	".wants-card",
	"[class*='want-card']",
}

// NBSP у Go-регекспах — це \x{00a0} (синтаксис \u не підтримується).
var rubRe = regexp.MustCompile(`(\d[\d\s\x{00a0},]*)\s*(?:₽|руб)`)

// parseKworkBudgetCents: «от 3 000 ₽» → 3000 USD-центів після конвертації курсом.
func parseKworkBudgetCents(text string) *int {
	m := rubRe.FindStringSubmatch(text)
	if m == nil {
		return parseBudgetCents(text) // раптом бюджет у $
	}
	raw := strings.NewReplacer(" ", "", "\u00a0", "", ",", "").Replace(m[1])
	rub, err := strconv.Atoi(raw)
	if err != nil {
		return nil
	}
	cents := rub * 100 / kworkRubPerUsd
	return &cents
}

func kworkAbsURL(href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	return "https://kwork.ru" + href
}

func FetchKwork(ctx context.Context) ([]model.Listing, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	body, status, err := fetchURL(ctx, client, kworkProjectsURL)
	if err != nil {
		return nil, fmt.Errorf("kwork: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("kwork: HTTP %d", status)
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("kwork: html: %w", err)
	}
	var out []model.Listing
	for _, sel := range kworkCards {
		doc.Find(sel).Each(func(_ int, s *goquery.Selection) {
			titleLink := s.Find("a[href*='/projects/']").First()
			href, ok := titleLink.Attr("href")
			if !ok {
				return
			}
			title := strings.TrimSpace(titleLink.Text())
			if title == "" {
				return
			}
			desc := strings.TrimSpace(s.Find(".want-card__text, .wants-card__text, .description").First().Text())
			raw, _ := json.Marshal(map[string]string{"title": title, "link": href})
			out = append(out, model.Listing{
				ExternalID:  externalIDFromLink(href),
				URL:         kworkAbsURL(href),
				Title:       title,
				Description: desc,
				BudgetCents: parseKworkBudgetCents(s.Text()),
				Currency:    "USD", // бюджет конвертовано з ₽, див. kworkRubPerUsd
				Raw:         raw,
			})
		})
	if len(out) > 0 {
			break
		}
	}
	return out, nil
}
