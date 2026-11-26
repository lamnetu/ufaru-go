package parsing

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/lamnetu/ufaru-go/domain"
	"github.com/lamnetu/ufaru-go/event"
	"github.com/lamnetu/ufaru-go/eventbus"
	"github.com/lamnetu/ufaru-go/services/parsers"
)

// Service — application-сервис: координирует парсеры и публикует события.
// SENIOR NOTE: знает только интерфейс parsers.Parser, не конкретные реализации.
type Service struct {
	bus     *eventbus.Bus
	parsers []parsers.Parser
}

func New(bus *eventbus.Bus, ps ...parsers.Parser) *Service {
	return &Service{bus: bus, parsers: ps}
}

// Run запускает все парсеры параллельно и ждёт завершения.
func (s *Service) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, p := range s.parsers {
		// SENIOR NOTE: с Go 1.22 переменная цикла создаётся заново на каждой
		// итерации, поэтому p := p больше не нужен (раньше был классический баг).
		g.Go(func() error {
			return s.parseOne(ctx, p)
		})
	}
	return g.Wait()
}

// parseOne парсит один источник и публикует события.
// Ошибка источника НЕ прерывает весь Run: публикуем NewsParseFailedEvent
// и возвращаем nil, чтобы остальные источники доработали.
func (s *Service) parseOne(ctx context.Context, p parsers.Parser) error {
	news, err := p.Parse(ctx)
	if err != nil {
		et := event.ErrorTypeNetwork
		if ctx.Err() != nil {
			et = event.ErrorTypeTimeout
		}
		s.bus.Publish(ctx, &event.NewsParseFailedEvent{
			Source:     domain.Source(p.Name()),
			ErrorType:  et,
			Message:    err.Error(),
			OccurredAt: time.Now().UTC(),
		})
		return nil
	}

	for _, n := range news {
		if !n.Valid() {
			continue
		}
		s.bus.Publish(ctx, &event.NewsParsedEvent{
			News:       n,
			OccurredAt: time.Now().UTC(),
		})
	}
	return nil
}