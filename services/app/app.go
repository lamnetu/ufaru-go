package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/lamnetu/ufaru-go/config"
	"github.com/lamnetu/ufaru-go/event"
	"github.com/lamnetu/ufaru-go/services/eventbus"
	"github.com/lamnetu/ufaru-go/services/parsers"
	"github.com/lamnetu/ufaru-go/services/parsing"
	"github.com/lamnetu/ufaru-go/services/publishers"
	"github.com/lamnetu/ufaru-go/services/ratelimiter"
)

func Run(ctx context.Context, logger *slog.Logger) error {
	cfg := config.Load()

	bus := eventbus.New()
	closers, err := publishers.Build(cfg, bus)
	if err != nil {
		return err
	}
	defer func() {
		for _, c := range closers {
			if err := c(); err != nil {
				logger.Error("close publisher", "err", err)
			}
		}
	}()

	svc := parsing.NewService(
		parsers.NewUfa1(40), // ← лимит на прогон (40 карточек на ленте)
		parsers.NewPrufy(40),
	)
	hourly := ratelimiter.New(cfg.MaxPerHour, time.Hour)
	daily := ratelimiter.New(cfg.MaxPerDay, 24*time.Hour)

	ticker := time.NewTicker(cfg.ParseInterval)
	defer ticker.Stop()

	logger.Info("parser started",
		"interval", cfg.ParseInterval,
		"max_per_hour", cfg.MaxPerHour,
		"max_per_day", cfg.MaxPerDay,
		"publishers", cfg.EnabledPublishers,
	)

	runOnce := func() {
		if !hourly.Allow() || !daily.Allow() {
			logger.Warn("rate limit reached, skipping run")
			return
		}

		res := svc.Run(ctx)
		now := time.Now().UTC()

		for _, n := range res.News {
			bus.Publish(ctx, &event.NewsParsedEvent{News: n, OccurredAt: now})
		}
		for _, f := range res.Failures {
			bus.Publish(ctx, &event.NewsParseFailedEvent{
				Source:     f.Source,
				ErrorType:  event.ErrorTypeParse,
				Message:    f.Err.Error(),
				OccurredAt: now,
			})
		}

		logger.Info("run finished", "news", len(res.News), "failures", len(res.Failures))
	}

	runOnce()

	for {
		select {
		case <-ctx.Done():
			logger.Info("shutdown")
			return nil
		case <-ticker.C:
			runOnce()
		}
	}
}
