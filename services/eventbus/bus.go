package eventbus

import (
	"context"
	"reflect"
	"sync"

	"github.com/lamnetu/ufaru-go/event"
)

// Handler — сигнатура обработчика события.
type Handler func(ctx context.Context, evt event.Event)

// Bus — in-memory шина событий.
type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

// New создаёт шину.
func New() *Bus {
	return &Bus{
		handlers: make(map[string][]Handler),
	}
}

// eventName возвращает короткое имя типа события ("NewsParsedEvent").
// Elem() используется, потому что в Publish мы передаём указатель на событие.
func eventName(evt event.Event) string {
	return reflect.TypeOf(evt).Elem().Name()
}

// Subscribe регистрирует обработчик на тип события.
// evt передаётся как образец типа; значение игнорируется, важен только тип.
func (b *Bus) Subscribe(evt event.Event, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	name := eventName(evt)
	b.handlers[name] = append(b.handlers[name], handler)
}

// Publish вызывает все обработчики данного типа события.
// Обработчики выполняются синхронно в потоке парсера.
//
// SENIOR NOTE: для production стоит заменить на асинхронную модель
// с горутинами и каналами (chan event.Event). Здесь синхронно для простоты MVP:
// нет race conditions, легко тестировать, легко дебажить.
func (b *Bus) Publish(ctx context.Context, evt event.Event) {
	b.mu.RLock()
	handlers := b.handlers[eventName(evt)]
	b.mu.RUnlock()

	for _, h := range handlers {
		h(ctx, evt)
	}
}