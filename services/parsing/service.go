package parsing

import (
	"context"

	"github.com/lamnetu/ufaru-go/domain"
	"github.com/lamnetu/ufaru-go/services/parsers"
)

// Failure — сбой конкретного источника за проход.
type Failure struct {
	Source domain.Source
	Err    error
}

// Result — итог одного прохода. Сервис ничего не публикует и не логирует.
type Result struct {
	News     []*domain.News
	Failures []Failure
}

type Service struct {
	parsers []parsers.Parser
}

func NewService(ps ...parsers.Parser) *Service {
	return &Service{parsers: ps}
}

// Run — опрашивает источники и возвращает результат.
// Сбой одного источника не валит остальные.
func (s *Service) Run(ctx context.Context) Result {
	var res Result
	for _, p := range s.parsers {
		if ctx.Err() != nil {
			return res
		}
		news, err := p.ParseNewsList(ctx)
		if err != nil {
			res.Failures = append(res.Failures, Failure{
				Source: domain.Source(p.Name()),
				Err:    err,
			})
			continue
		}
		res.News = append(res.News, news...)
	}
	return res
}
