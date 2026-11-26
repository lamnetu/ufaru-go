package publishers

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/lamnetu/ufaru-go/event"
)

// Kafka — publisher, отправляющий события в Kafka.
type Kafka struct {
	producer sarama.AsyncProducer
	topic    string
	topicErr string
	mu       sync.RWMutex
}

func NewKafka(brokers []string, topic, topicErr string) (*Kafka, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Retry.Max = 3
	config.Producer.Return.Successes = false

	producer, err := sarama.NewAsyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}

	k := &Kafka{
		producer: producer,
		topic:    topic,
		topicErr: topicErr,
	}

	// Фоновая горутина читает ошибки от producer'а
	go func() {
		for err := range producer.Errors() {
			log.Printf("[kafka] producer error: %v", err.Err)
		}
	}()

	return k, nil
}

type kafkaMessage struct {
	Type       string    `json:"type"`
	Source     string    `json:"source,omitempty"`
	Title      string    `json:"title,omitempty"`
	Content    string    `json:"content,omitempty"`
	URL        string    `json:"url,omitempty"`
	Category   string    `json:"category,omitempty"`
	ErrorType  string    `json:"error_type,omitempty"`
	Message    string    `json:"message,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
}

func (k *Kafka) Handle(_ context.Context, evt event.Event) {
	var msg kafkaMessage
	var topic string

	switch e := evt.(type) {
	case *event.NewsParsedEvent:
		msg = kafkaMessage{
			Type:       "news_parsed",
			Source:     string(e.News.Source),
			Title:      e.News.Title,
			Content:    e.News.Content,
			URL:        e.News.URL,
			Category:   string(e.News.Category),
			OccurredAt: e.OccurredAt,
		}
		topic = k.topic

	case *event.NewsParseFailedEvent:
		msg = kafkaMessage{
			Type:       "news_parse_failed",
			Source:     string(e.Source),
			ErrorType:  string(e.ErrorType),
			Message:    e.Message,
			URL:        e.URL,
			OccurredAt: e.OccurredAt,
		}
		topic = k.topicErr

	default:
		return
	}

	body, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[kafka] marshal error: %v", err)
		return
	}

	k.mu.RLock()
	producer := k.producer
	k.mu.RUnlock()

	// Если circuit breaker открыт — логируем и пропускаем
	// (в production нужно пересоздавать producer)
	select {
	case producer.Input() <- &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(msg.URL),
		Value: sarama.ByteEncoder(body),
	}:
	default:
		log.Printf("[kafka] circuit breaker open, message dropped: %s", msg.URL)
	}
}

func (k *Kafka) Close() error {
	return k.producer.Close()
}
