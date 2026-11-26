package parsers

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/lamnetu/ufaru-go/domain"
)

const (
	ufa1Name     = "ufa1"
	ufa1StartURL = "https://ufa1.ru/text/"
	ufa1Timeout  = 15 * time.Second
)

// CSS-селекторы для ufa1.ru. При смене верстки правим здесь, код парсинга не меняется.
const (
	ufa1ItemSel        = "article.news-card"
	ufa1TitleSel       = "h2.title a"
	ufa1LinkSel        = "h2.title a"
	ufa1DescriptionSel = "p.lead"
	ufa1ImageSel       = "img.cover"
	ufa1DateSel        = "time.published"
)

// Ufa1 — парсер сайта ufa1.ru.
type Ufa1 struct {
	collector *colly.Collector
}

func NewUfa1() *Ufa1 {
	c := colly.NewCollector(
		colly.MaxDepth(1),
		colly.UserAgent("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36"),
	)
	c.SetRequestTimeout(ufa1Timeout)

	// Ограничиваем домен, чтобы случайно не уйти по внешним ссылкам.
	_ = c.Limit(&colly.LimitRule{DomainGlob: "ufa1.ru"})

	return &Ufa1{collector: c}
}

func (p *Ufa1) Name() string { return ufa1Name }

// Parse обходит ленту и возвращает список новостей.
// SENIOR NOTE: ошибка возвращается, только если упал весь сайт (нет ни одной новости).
// Ошибки отдельных элементов логируем, но не прерываем — одна битая карточка
// не должна обнулять результат.
func (p *Ufa1) Parse(ctx context.Context) ([]*domain.News, error) {
	var (
		news []*domain.News
		errs []error
	)

	p.collector.OnHTML(ufa1ItemSel, func(e *colly.HTMLElement) {
		title := strings.TrimSpace(e.ChildText(ufa1TitleSel))
		if title == "" {
			return // пропускаем пустые карточки
		}

		href := e.ChildAttr(ufa1LinkSel, "href")
		link := resolveURL(ufa1StartURL, href)
		if link == "" {
			errs = append(errs, fmt.Errorf("empty link for title %q", title))
			return
		}

		item := &domain.News{
			Title:       title,
			URL:         link,
			Source:      domain.SourceUfa1,
			Description: strings.TrimSpace(e.ChildText(ufa1DescriptionSel)),
			ImageURL:    e.ChildAttr(ufa1ImageSel, "src"),
			ParsedAt:    time.Now().UTC(),
		}

		if raw := strings.TrimSpace(e.ChildText(ufa1DateSel)); raw != "" {
			if t, ok := parseRuDate(raw); ok {
				item.PublishedAt = t
			}
		}

		news = append(news, item)
	})

	p.collector.OnError(func(_ *colly.Response, err error) {
		errs = append(errs, err)
	})

	if err := p.collector.Visit(ufa1StartURL); err != nil {
		return nil, fmt.Errorf("ufa1: visit %s: %w", ufa1StartURL, err)
	}

	// Если ничего не собрали, но были ошибки — значит сайт упал.
	if len(news) == 0 && len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return news, nil
}

// resolveURL превращает относительный /news/123 в абсолютный https://ufa1.ru/news/123.
// SENIOR NOTE: частая ошибка парсеров — забыть резолвить относительные ссылки.
// net/url.ResolveReference делает это по RFC 3986.
func resolveURL(base, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	b, err := url.Parse(base)
	if err != nil {
		return ""
	}
	r, err := url.Parse(ref)
	if err != nil {
		return ""
	}
	return b.ResolveReference(r).String()
}

// parseRuDate парсит русские даты типа "26 ноября 2026, 14:30".
// SENIOR NOTE: time.Parse не знает русских месяцев — нужна ручная замена.
// Альтернатива: парсить через regexp и собирать time.Date() вручную.
func parseRuDate(raw string) (time.Time, bool) {
	s := strings.ToLower(raw)
	for ru, en := range ruMonths {
		s = strings.ReplaceAll(s, ru, en)
	}
	for _, layout := range ruDateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

// Маппинг русских месяцев в английские для time.Parse.
var ruMonths = map[string]string{
	"января":   "January",
	"февраля":  "February",
	"марта":    "March",
	"апреля":   "April",
	"мая":      "May",
	"июня":     "June",
	"июля":     "July",
	"августа":  "August",
	"сентября": "September",
	"октября":  "October",
	"ноября":   "November",
	"декабря":  "December",
}

// Покрытые форматы дат. Добавлять по мере встречи на сайте.
var ruDateLayouts = []string{
	"02 January 2006, 15:04",
	"2 January 2006, 15:04",
	"02 January 2006",
	"2 January 2006",
	time.RFC3339,
}