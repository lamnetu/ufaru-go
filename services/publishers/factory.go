package publishers

import (
	"fmt"

	"github.com/lamnetu/ufaru-go/config"
	"github.com/lamnetu/ufaru-go/event"
	"github.com/lamnetu/ufaru-go/eventbus"
)

// Build — включает паблишеры по конфигу и подписывает их на шину.
// Возвращает клоузеры (kafka и т.п.), которые вызывающий обязан закрыть.
func Build(cfg config.Config, bus *eventbus.Bus) ([]func() error, error) {
	var closers []func() error

	for _, name := range cfg.EnabledPublishers {
		switch name {
		case "console":
			c := NewConsole()
			bus.Subscribe(&event.NewsParsedEvent{}, c.Handle)
			bus.Subscribe(&event.NewsParseFailedEvent{}, c.Handle)

		case "kafka":
			k, err := NewKafka(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaTopicFailed)
			if err != nil {
				return nil, fmt.Errorf("kafka init: %w", err)
			}
			bus.Subscribe(&event.NewsParsedEvent{}, k.Handle)
			bus.Subscribe(&event.NewsParseFailedEvent{}, k.Handle)
			closers = append(closers, k.Close)

		case "rest":
			r := NewRest(cfg.APIEndpoint, cfg.APIKey)
			bus.Subscribe(&event.NewsParsedEvent{}, r.Handle)
			bus.Subscribe(&event.NewsParseFailedEvent{}, r.Handle)

		default:
			return nil, fmt.Errorf("unknown publisher %q", name)
		}
	}
	return closers, nil
}
