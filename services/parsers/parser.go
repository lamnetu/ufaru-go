package parsers

import (
	"context"

	"github.com/lamnetu/ufaru-go/domain"
)

// Parser — контракт источника новостей.
// SENIOR NOTE: имя ParseNewsList говорящее: вызывающий код сразу понимает,
// что парсится лента (список), а не детальная страница одной статьи.
type Parser interface {
	// ParseNewsList собирает список новостей с ленты источника.
	ParseNewsList(ctx context.Context) ([]*domain.News, error)

	// Name — уникальный идентификатор источника ("ufa1", "prufy").
	Name() string
}
