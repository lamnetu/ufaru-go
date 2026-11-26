package parsers

import (
	"context"

	"github.com/lamnetu/ufaru-go/domain"
)

// Parser — контракт конкретного парсера источника новостей.
//
// SENIOR NOTE: это producer-side интерфейс (лежит там же, где и реализации).
// Альтернатива — consumer-side: объявить интерфейс в пакете parsing/service.go,
// чтобы потребитель (оркестратор) определял, что ему нужно. Это даёт более чистые
// зависимости (parsers не зависит от parsing), но требует дополнительного импорта
// в оркестраторе. Для MVP с 2-3 реализациями producer-side проще и читаемее.
type Parser interface {
	// Parse возвращает список новостей с источника.
	// context используется для таймаутов и graceful shutdown.
	Parse(ctx context.Context) ([]*domain.News, error)

	// Name — уникальный идентификатор источника (напр. "ufa1", "prufy").
	// Используется для логов, метрик, дедупликации по источнику.
	Name() string
}