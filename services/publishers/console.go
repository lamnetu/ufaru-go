package publishers

import (
	"context"
	"fmt"

	"github.com/lamnetu/ufaru-go/event"
)

// Console — publisher для отладки: пишет события в stdout.
type Console struct{}

func NewConsole() *Console { return &Console{} }

// Handle соответствует сигнатуре eventbus.Handler, поэтому его можно
// передать в bus.Subscribe напрямую.
func (c *Console) Handle(_ context.Context, evt event.Event) {
	switch e := evt.(type) {
	case *event.NewsParsedEvent:
		fmt.Printf("[console] parsed: %s | %s\n", e.News.Title, e.News.URL)
	case *event.NewsParseFailedEvent:
		fmt.Printf("[console] FAILED: source=%s type=%s msg=%s\n", e.Source, e.ErrorType, e.Message)
	}
}