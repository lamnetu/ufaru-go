package parsers

import (
	"context"

	"github.com/lamnetu/ufaru-go/domain"
)

// Parser — контракт источника новостей.
type Parser interface {
	ParseNewsList(ctx context.Context) ([]*domain.News, error)
	Name() string
}
