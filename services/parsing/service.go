package parsing

import (
	"context"
	"sync"
	"time"

	"github.com/lamnetu/ufaru-go/domain"
	"github.com/lamnetu/ufaru-go/event"
	"github.com/lamnetu/ufaru-go/eventbus"
	"github.com/lamnetu/ufaru-go/services/parsers"
)

// Service — application-сервис: один проход по всем источникам.
type Service struct {
	parsers []parsers.Parser
	bus     *eventbus.Bus
}

func NewService(bus *eventbus.Bus, ps ...parsers.Parser) *Service {
	return &Service{parsers: ps, bus: bus}
}

// Run обходит все источники параллельно (горутина на парсер).
// SENIOR NOTE: WaitGroup ждёт завершения всех; шина thread-safe,
// поэтому Publish из разных горутин безопасен без дополнительного мьютекса.
func (s *Service) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for _, p := range s.parsers {
		wg.Add(1)
		go func(p parsers.Parser) {
			defer wg.Done()
			s.parseOne(ctx, p)
		}(p)
	}
	wg.Wait()
}

func (s *Service) parseOne(ctx context.Context, p parsers.Parser) {
	news, err := p.Parse(ctx)
	if err != nil {
		s.bus.Publish(ctx, &event.NewsParseFailedEvent{
			Source:     domain.Source(p.Name()),
			ErrorType:  event.ErrorTypeParse,
			Message:    err.Error(),
			OccurredAt: time.Now().UTC(),
		})
		return
	}

	for _, n := range news {
		if !n.Valid() {
			continue // доменное правило: без заголовка/URL не публикуем
		}
		s.bus.Publish(ctx, &event.NewsParsedEvent{
			News:       n,
			OccurredAt: time.Now().UTC(),
		})
	}
}