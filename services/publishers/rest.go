package publishers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/lamnetu/ufaru-go/domain"
	"github.com/lamnetu/ufaru-go/event"
)

// newsDTO — транспортное представление новости.
// SENIOR NOTE: домен не знает про JSON; маппинг домен->DTO это задача инфраструктуры.
type newsDTO struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	URL         string    `json:"url"`
	ImageURL    string    `json:"image_url,omitempty"`
	Source      string    `json:"source"`
	Category    string    `json:"category,omitempty"`
	PublishedAt time.Time `json:"published_at,omitempty"`
}

func toDTO(n *domain.News) newsDTO {
	return newsDTO{
		Title:       n.Title,
		Description: n.Description,
		Content:     n.Content,
		URL:         n.URL,
		ImageURL:    n.ImageURL,
		Source:      string(n.Source),
		Category:    string(n.Category),
		PublishedAt: n.PublishedAt,
	}
}

// Rest — publisher, отправляющий новости в Symfony API.
type Rest struct {
	client   *http.Client
	endpoint string
	apiKey   string
}

func NewRest(endpoint, apiKey string) *Rest {
	return &Rest{
		client:   &http.Client{Timeout: 10 * time.Second},
		endpoint: endpoint,
		apiKey:   apiKey,
	}
}

// Handle отправляет только NewsParsedEvent; остальные игнорирует.
func (r *Rest) Handle(ctx context.Context, evt event.Event) {
	parsed, ok := evt.(*event.NewsParsedEvent)
	if !ok {
		return
	}

	body, err := json.Marshal(toDTO(parsed.News))
	if err != nil {
		fmt.Printf("[rest] marshal error: %v\n", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if r.apiKey != "" {
		req.Header.Set("X-API-Key", r.apiKey)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		fmt.Printf("[rest] request error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		fmt.Printf("[rest] bad status %d for %s\n", resp.StatusCode, parsed.News.URL)
	}
}