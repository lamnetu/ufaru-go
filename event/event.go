package event

import (
	"time"

	"github.com/lamnetu/ufaru-go/domain"
)

// Event — маркерный интерфейс для всех событий системы.
type Event interface {
	IsEvent()
}

// NewsParsedEvent — успешный парсинг одной новости (one message, one entity).
type NewsParsedEvent struct {
	// SENIOR NOTE: указатель *domain.News — не копируем структуру
	// при передаче через каналы в горутины.
	News       *domain.News
	OccurredAt time.Time
}

func (NewsParsedEvent) IsEvent() {}

// ErrorType — семантический тип сбоя для маршрутизации и алертинга.
type ErrorType string

const (
	ErrorTypeNetwork ErrorType = "NETWORK"
	ErrorTypeParse   ErrorType = "PARSE"
	ErrorTypeTimeout ErrorType = "TIMEOUT"
)

// NewsParseFailedEvent — сбой парсинга источника.
type NewsParseFailedEvent struct {
	Source     domain.Source
	ErrorType  ErrorType
	URL        string // может быть пустым, если упал сайт целиком
	Message    string
	OccurredAt time.Time
}

func (NewsParseFailedEvent) IsEvent() {}