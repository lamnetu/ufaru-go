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

// Селекторы ленты (реальный HTML /text/).
const (
	ufa1ListSel        = "div.announcementList_6xAkH > div.wrap_RL97A"
	ufa1TitleSel       = "a.header_RL97A"
	ufa1LinkSel        = "a.header_RL97A"
	ufa1DescriptionSel = "span.subtitle_RL97A"
	ufa1ImageSel       = "a.imgBg_RL97A img.image_RL97A"
	ufa1DateSel        = "span.text_Rd6P3[data-announcement-date]"
	ufa1CategorySel    = "a.rubric_fQns7"
)

// Селекторы детальной страницы (фолбэки — точный селектор зависит от вёрстки).
var ufa1DetailBodySelectors = []string{
	"div.article-body",
	"div.content_RL97A",
	"article",
	"[data-page='TYPE_ARTICLE'] .article-content",
	".text-block",
}

type Ufa1 struct {
	collector       *colly.Collector
	detailCollector *colly.Collector
}

func NewUfa1() *Ufa1 {
	c := colly.NewCollector(
		colly.MaxDepth(1),
		colly.UserAgent(userAgent),
		colly.AllowURLRevisit(),
	)
	c.SetRequestTimeout(ufa1Timeout)
	_ = c.Limit(&colly.LimitRule{DomainGlob: "ufa1.ru"})

	dc := colly.NewCollector(
		colly.MaxDepth(0),
		colly.UserAgent(userAgent),
	)
	dc.SetRequestTimeout(ufa1Timeout)
	_ = dc.Limit(&colly.LimitRule{DomainGlob: "ufa1.ru"})

	return &Ufa1{
		collector:       c,
		detailCollector: dc,
	}
}

func (p *Ufa1) Name() string { return ufa1Name }

func (p *Ufa1) ParseNewsList(ctx context.Context) ([]*domain.News, error) {
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

		// SENIOR NOTE: ходим на детальную страницу за полным текстом.
		// Rate limit 300ms между запросами — не долбим источник.
		time.Sleep(300 * time.Millisecond)
		if body, err := p.parseNewsDetail(link); err != nil {
			errs = append(errs, fmt.Errorf("detail parse %s: %w", link, err))
		} else {
			item.Content = body
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

// parseNewsDetail извлекает полный текст одной статьи с детальной страницы.
// SENIOR NOTE: пробуем несколько селекторов по порядку, берём первый непустой.
func (p *Ufa1) parseNewsDetail(url string) (string, error) {
	var body string
	var err error

	p.detailCollector.OnHTML("", func(e *colly.HTMLElement) {
		// Перебираем селекторы, ищем первый непустой контейнер
		for _, sel := range ufa1DetailBodySelectors {
			el := e.DOM.Find(sel).First()
			if el.Length() == 0 {
				continue
			}
			// Удаляем мусор
			el.Find("script, style, iframe, noscript, .advert, [data-ad-unit], [class*='adv']").Remove()
			text := strings.TrimSpace(el.Text())
			if text != "" && len(text) > 100 {
				body = text
				return
			}
		}
	})

	p.detailCollector.OnError(func(_ *colly.Response, e error) {
		err = e
	})

	if e := p.detailCollector.Visit(url); e != nil {
		return "", e
	}
	if err != nil {
		return "", err
	}
	if body == "" {
		return "", fmt.Errorf("empty body for %s", url)
	}
	return body, nil
}
