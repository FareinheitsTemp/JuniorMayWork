// Пакет sources: реєстр джерел замовлень. Новий сайт = новий Fetch-метод.
package sources

import (
	"context"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
)

type FetchFunc func(ctx context.Context) ([]model.Listing, error)

// Sources повертає всі доступні джерела за ім'ям (див. JMW_SOURCES).
func Sources() map[string]FetchFunc {
	return map[string]FetchFunc{
		"telegram":     FetchTelegram,
		"freelancehunt": FetchFreelancehunt,
		"weblancer":    FetchWeblancer,
	}
}
