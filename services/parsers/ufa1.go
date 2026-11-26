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
	ufa1Name     = "ufa1"
	ufa1StartURL = "https://ufa1.ru/text/"
	ufa1Timeout  = 20 * time.Second
)

// Селекторы по реальному HTML ленты /text/.
const (
	ufa1ListSel        = "div.announcementList_6xAkH > div.wrap_RL97A"
	ufa1TitleSel       = "a.header_RL97A"
	ufa1LinkSel        = "a.header_RL97A"
	ufa1DescriptionSel = "span.subtitle_RL97A"
	ufa1ImageSel       = "a.imgBg_RL97A img.image_RL97A"
	ufa1DateSel        = "span.text_Rd6P3[data-announcement-date]"
	ufa1CategorySel    = "a.rubric_fQns7"
)

type Ufa1 struct {
	collector *colly.Collector
}

func NewUfa1() *Ufa1 {
	c := colly.NewCollector(
		colly.MaxDepth(1),
		colly.UserAgent(userAgent),
		colly.AllowURLRevisit(),
	)
	c.SetRequestTimeout(ufa1Timeout)
	_ = c.Limit(&colly.LimitRule{DomainGlob: "ufa1.ru"})

	return &Ufa1{collector: c}
}

func (p *Ufa1) Name() string { return ufa1Name }

func (p *Ufa1) Parse(ctx context.Context) ([]*domain.News, error) {
	var (
		news []*domain.News
		errs []error
	)

	p.collector.OnHTML(ufa1ListSel, func(e *colly.HTMLElement) {
		title := strings.TrimSpace(e.ChildText(ufa1TitleSel))
		if title == "" {
			return
		}

		link := resolveURL(ufa1StartURL, e.ChildAttr(ufa1LinkSel, "href"))
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
			Category:    parseUfa1Category(e.ChildText(ufa1CategorySel)),
			ParsedAt:    time.Now().UTC(),
		}

		if raw := strings.TrimSpace(e.ChildAttr(ufa1DateSel, "data-announcement-date")); raw != "" {
			if t, ok := parseUfa1Date(raw); ok {
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

	if len(news) == 0 && len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return news, nil
}

// parseUfa1Date парсит "26 ноября, 2026, 19:40" (время уфимское, UTC+5).
func parseUfa1Date(raw string) (time.Time, bool) {
	s := strings.ToLower(strings.TrimSpace(raw))
	for ru, en := range ruMonths {
		s = strings.ReplaceAll(s, ru, en)
	}
	s = strings.ReplaceAll(s, ",", "")

	for _, layout := range []string{
		"2 January 2006 15:04",
		"02 January 2006 15:04",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			ufa := time.FixedZone("UFA", 5*3600)
			return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, ufa).UTC(), true
		}
	}
	return time.Time{}, false
}

// parseUfa1Category маппит русские рубрики ufa1 в domain.Category.
// ufa1 использует "Происшествия", "Культура" и т.д. — нужно свести к нашему enum.
func parseUfa1Category(raw string) domain.Category {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "экономика":
		return domain.CategoryEconomy
	case "политика":
		return domain.CategoryPolitics
	case "спорт":
		return domain.CategorySport
	case "культура":
		return domain.CategoryCulture
	case "происшествия", "криминал":
		return domain.CategoryIncidents
	case "город":
		return domain.CategoryCity
	default:
		return ""
	}
}
