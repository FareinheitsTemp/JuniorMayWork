// Спільні хелпери джерел: HTTP-запити, парсинг бюджетів, ідентифікатори.
package sources

import (
	"context"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const browserUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"

// Орієнтовний курс конвертації гривні в долари (грубо, 2026).
const uahPerUsd = 41

var budgetRe = regexp.MustCompile(`\$\s*([\d][\d,]*)(?:\s*(?:-|–|to)\s*\$?\s*([\d][\d,]*))?`)

// parseBudgetCents дістає бюджет у $ з тексту ("$50", "$30-$100").
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

// uahToUsdCents: суму в гривнях (з пробілами/комами) → USD-центи.
func uahToUsdCents(raw string) *int {
	clean := strings.NewReplacer(" ", "", "\u00a0", "", ",", "").Replace(raw)
	v, err := strconv.Atoi(clean)
	if err != nil {
		return nil
	}
	cents := v * 100 / uahPerUsd
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
