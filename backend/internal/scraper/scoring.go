package scraper

import (
	"math"
	"strings"
	"time"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
)

// listingScore is kept separate from persistence so the ranking policy remains
// deterministic, explainable and testable.
type listingScore struct {
	BranchID       *int64
	Freshness      float64
	Relevance      float64
	BudgetQuality  float64
	Priority       float64
	MatchedTags    []string
}

func scoreListing(order model.Order, branches []model.Branch, now time.Time) listingScore {
	title := normalizeScoreText(order.Title)
	description := normalizeScoreText(order.Description)
	result := listingScore{Freshness: scoredFreshness(order.SourcePublishedAt, order.FirstSeenAt, now)}

	bestBranchScore := 0.0
	for _, branch := range branches {
		if order.BudgetCents != nil && *order.BudgetCents > branch.MaxBudgetCents {
			continue
		}
		score, matches := branchRelevance(title, description, branch.Keywords)
		if score > bestBranchScore {
			bestBranchScore, result.MatchedTags = score, matches
			id := branch.ID
			result.BranchID = &id
			result.BudgetQuality = scoredBudgetQuality(order.BudgetCents, branch.MaxBudgetCents)
		}
	}

	genericScore, genericMatches := branchRelevance(title, description, []string{
		"react", "next.js", "nextjs", "typescript", "javascript", "node.js", "nodejs",
		"golang", "go", "python", "frontend", "backend", "fullstack", "лендінг", "верстка",
	})
	if bestBranchScore == 0 {
		bestBranchScore, result.MatchedTags = genericScore, genericMatches
		result.BudgetQuality = scoredBudgetQuality(order.BudgetCents, 0)
	} else {
		bestBranchScore = math.Min(100, bestBranchScore+genericScore*0.18)
	}

	result.Relevance = roundScore(bestBranchScore)
	result.Priority = roundScore(0.42*result.Freshness + 0.48*result.Relevance + 0.10*result.BudgetQuality)
	return result
}

func normalizeScoreText(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

// A title hit has twice the weight of a description hit. Exact multi-word tags
// are treated as a single signal, so repeated wording cannot inflate a listing.
func branchRelevance(title, description string, keywords []string) (float64, []string) {
	if len(keywords) == 0 { return 0, nil }
	score, hits := 0.0, 0
	matches := make([]string, 0, len(keywords))
	seen := map[string]bool{}
	for _, raw := range keywords {
		keyword := normalizeScoreText(raw)
		if keyword == "" || seen[keyword] { continue }
		seen[keyword] = true
		if strings.Contains(title, keyword) {
			score += 28
			hits++
			matches = append(matches, raw)
			continue
		}
		if strings.Contains(description, keyword) {
			score += 13
			hits++
			matches = append(matches, raw)
		}
	}
	coverage := float64(hits) / float64(len(seen))
	score += coverage * 24
	return math.Min(100, score), matches
}

func scoredFreshness(publishedAt *time.Time, firstSeenAt, now time.Time) float64 {
	at := firstSeenAt
	if publishedAt != nil { at = *publishedAt }
	hours := math.Max(0, now.Sub(at).Hours())
	return roundScore(100 * math.Exp(-hours/24))
}

// Unknown budget is neutral rather than destructive. Within a profile/branch
// budget cap, a larger viable budget receives a larger opportunity score.
func scoredBudgetQuality(budget *int, maximum int) float64 {
	if budget == nil { return 50 }
	if maximum <= 0 { return math.Min(85, 45+float64(*budget)/200) }
	if *budget > maximum { return 0 }
	ratio := float64(*budget) / float64(maximum)
	return roundScore(55 + ratio*45)
}

func roundScore(value float64) float64 {
	return math.Round(math.Max(0, math.Min(100, value))*100) / 100
}
