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
	prufyTimeout  = 20 * time.Second
)

// Селекторы под Bitrix-вёрстку. ВАЖНО: без BX-заголовков страница приходит
// пустой (контент грузится динамическими блоками bxdynamic_*).
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
		colly.UserAgent(userAgent),
		colly.AllowURLRevisit(),
	)
	c.SetRequestTimeout(prufyTimeout)
	_ = c.Limit(&colly.LimitRule{DomainGlob: "prufy.ru"})

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("BX-ACTION-TYPE", "get_dynamic")
		r.Headers.Set("X-Bitrix-Composite", "get_dynamic")
	})

	return &Prufy{collector: c}
}

func (p *Prufy) Name() string { return prufyName }

func (p *Prufy) ParseNewsList(ctx context.Context) ([]*domain.News, error) {
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
			if t, ok := parseRuDate(raw); ok {
				item.PublishedAt = t
			} else if t, ok := parseRelativeDate(raw); ok {
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
