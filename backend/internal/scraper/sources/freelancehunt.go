// Джерело: Freelancehunt — офіційний API v2 (Bearer-токен опційний).
// GET https://api.freelancehunt.com/v2/projects — віддає відкритий JSON:API.
package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
)

const freelancehuntAPIURL = "https://api.freelancehunt.com/v2/projects"

type freelancehuntResponse struct {
	Data []struct {
		ID         int64 `json:"id"`
		Attributes struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Link        string `json:"link"`
			Budget      *struct {
				Amount   float64 `json:"amount"`
				Currency string  `json:"currency"`
			} `json:"budget"`
			Skills []struct {
				Name string `json:"name"`
			} `json:"skills"`
		} `json:"attributes"`
	} `json:"data"`
}

func FetchFreelancehunt(ctx context.Context) ([]model.Listing, error) {
	client := &http.Client{Timeout: 20 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, freelancehuntAPIURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", browserUA)
	if token := os.Getenv("JMW_FREELANCEHUNT_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("freelancehunt: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("freelancehunt: HTTP %d", resp.StatusCode)
	}

	var feed freelancehuntResponse
	if err := json.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("freelancehunt: json: %w", err)
	}

	out := make([]model.Listing, 0, len(feed.Data))
	for _, p := range feed.Data {
		a := p.Attributes
		if a.Name == "" {
			continue
		}
		link := a.Link
		if link == "" {
			link = "https://freelancehunt.com/projects"
		}
		skills := make([]string, 0, len(a.Skills))
		for _, sk := range a.Skills {
			if sk.Name != "" {
				skills = append(skills, sk.Name)
			}
		}
		var budget *int
		if a.Budget != nil {
			switch a.Budget.Currency {
			case "USD":
				c := int(a.Budget.Amount * 100)
				budget = &c
			case "UAH":
				c := int(a.Budget.Amount) * 100 / uahPerUsd
				budget = &c
			case "RUB", "RUR":
				// курс ~100 ₽/$: кількість рублів ≈ кількість USD-центів
				c := int(a.Budget.Amount)
				budget = &c
			}
		}
		raw, _ := json.Marshal(p)
		out = append(out, model.Listing{
			ExternalID:  fmt.Sprintf("fh:%d", p.ID),
			URL:         link,
			Title:       a.Name,
			Description: a.Description,
			BudgetCents: budget,
			Currency:    "USD",
			Skills:      skills,
			Raw:         raw,
		})
	}
	return out, nil
}
