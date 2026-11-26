package parsers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/lamnetu/ufaru-go/domain"
)

const (
	prufyName     = "prufy"
	prufyStartURL = "https://prufy.ru/news/"
	prufyTimeout  = 15 * time.Second
)

const (
	prufyItemSel        = "div.news-item"
	prufyTitleSel       = "a.news-title"
	prufyLinkSel        = "a.news-title"
	prufyDescriptionSel = "div.news-excerpt"
	prufyImageSel       = "img.news-img"
	prufyDateSel        = "span.news-date"
)

type Prufy struct {
	collector *colly.Collector
}

func NewPrufy() *Prufy {
	c := colly.NewCollector(
		colly.MaxDepth(1),
		colly.UserAgent("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36"),
		colly.AllowURLRevisit(),
	)
	c.SetRequestTimeout(prufyTimeout)
	_ = c.Limit(&colly.LimitRule{DomainGlob: "prufy.ru"})

	return &Prufy{collector: c}
}

func (p *Prufy) Name() string { return prufyName }

// Parse обходит ленту prufy.ru.
// SENIOR NOTE: resolveURL переиспользуется из ufa1.go — оба файла в пакете parsers,
// поэтому хелпер доступен без импорта. Это и есть смысл общего пакета.
func (p *Prufy) Parse(ctx context.Context) ([]*domain.News, error) {
	var (
		news []*domain.News
		errs []error
	)

	p.collector.OnHTML(prufyItemSel, func(e *colly.HTMLElement) {
		title := strings.TrimSpace(e.ChildText(prufyTitleSel))
		if title == "" {
			return
		}

		link := resolveURL(prufyStartURL, e.ChildAttr(prufyLinkSel, "href"))
		if link == "" {
			errs = append(errs, fmt.Errorf("empty link for title %q", title))
			return
		}

		item := &domain.News{
			Title:       title,
			URL:         link,
			Source:      domain.SourcePrufy,
			Description: strings.TrimSpace(e.ChildText(prufyDescriptionSel)),
			ImageURL:    e.ChildAttr(prufyImageSel, "src"),
			ParsedAt:    time.Now().UTC(),
		}

		if raw := strings.TrimSpace(e.ChildText(prufyDateSel)); raw != "" {
			// prufy часто пишет относительные даты ("2 часа назад"),
			// поэтому сначала пробуем относительную, потом абсолютную.
			if t, ok := parseRelativeDate(raw); ok {
				item.PublishedAt = t
			} else if t, ok := parseRuDate(raw); ok {
				item.PublishedAt = t
			}
		}

		news = append(news, item)
	})

	p.collector.OnError(func(_ *colly.Response, err error) {
		errs = append(errs, err)
	})

	if err := p.collector.Visit(prufyStartURL); err != nil {
		return nil, fmt.Errorf("prufy: visit %s: %w", prufyStartURL, err)
	}

	if len(news) == 0 && len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return news, nil
}

// parseRelativeDate обрабатывает относительные даты prufy:
// "5 минут назад", "2 часа назад", "вчера", "сегодня".
// SENIOR NOTE: относительная дата = now минус дельта. time.Now() берём в UTC,
// чтобы PublishedAt был согласован с остальными источниками.
func parseRelativeDate(raw string) (time.Time, bool) {
	s := strings.ToLower(strings.TrimSpace(raw))
	now := time.Now().UTC()

	switch {
	case strings.Contains(s, "минут"):
		return now.Add(-time.Duration(extractNum(s)) * time.Minute), true
	case strings.Contains(s, "час"):
		return now.Add(-time.Duration(extractNum(s)) * time.Hour), true
	case strings.Contains(s, "вчера"):
		return now.Add(-24 * time.Hour), true
	case strings.Contains(s, "сегодня"):
		return now, true
	}
	return time.Time{}, false
}

// extractNum вытасывает первое число из строки ("2 часа назад" -> 2).
// Если числа нет, возвращает 0 (Sscanf не находит совпадение).
func extractNum(s string) int {
	var n int
	_, _ = fmt.Sscanf(s, "%d", &n)
	return n
}