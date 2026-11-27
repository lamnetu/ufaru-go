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

// Селекторы ленты Bitrix.
const (
	prufyItemSel        = "div.news-item"
	prufyTitleSel       = "a.news-title"
	prufyLinkSel        = "a.news-title"
	prufyDescriptionSel = "div.news-excerpt"
	prufyImageSel       = "img.news-img"
	prufyDateSel        = "span.news-date"
)

// Селекторы детальной страницы prufy (из реального HTML 26.11.2026).
const (
	prufyDetailBodySel = "div.news-text-full"
	prufyDetailTextSel = "div.news-text"
	prufyDetailTitle   = "h1.header-details"
	prufyDetailDate    = "span.date-details"
	prufyDetailAuthor  = "span.author-name a"
	prufyDetailLead    = "p.prew-text"
)

type Prufy struct {
	collector       *colly.Collector
	detailCollector *colly.Collector
	maxItems        int
}

func NewPrufy(maxItems int) *Prufy {
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

	dc := colly.NewCollector(
		colly.MaxDepth(0),
		colly.UserAgent(userAgent),
	)
	dc.SetRequestTimeout(prufyTimeout)
	_ = dc.Limit(&colly.LimitRule{DomainGlob: "prufy.ru"})
	dc.OnRequest(func(r *colly.Request) {
		r.Headers.Set("BX-ACTION-TYPE", "get_dynamic")
		r.Headers.Set("X-Bitrix-Composite", "get_dynamic")
	})

	return &Prufy{
		collector:       c,
		detailCollector: dc,
		maxItems:        maxItems,
	}
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

		// Ходим на детальную за полным текстом.
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

	if err := p.collector.Visit(prufyStartURL); err != nil {
		return nil, fmt.Errorf("prufy: visit %s: %w", prufyStartURL, err)
	}

	if len(news) == 0 && len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return news, nil
}

// parseNewsDetail извлекает полный текст статьи с детальной страницы prufy.
// SENIOR NOTE: текст лежит в div.news-text внутри div.news-text-full.
// Bitrix отдаёт HTML с <p>, <h2>, <img> — берём innerHTML, чистим от рекламы.
func (p *Prufy) parseNewsDetail(url string) (string, error) {
	var (
		body string
		err  error
	)

	p.detailCollector.OnHTML(prufyDetailBodySel, func(e *colly.HTMLElement) {
		textBlock := e.DOM.Find(prufyDetailTextSel).First()
		if textBlock.Length() == 0 {
			return
		}
		// Удаляем мусор: реклама, комментарии, "читайте также", соцсети
		textBlock.Find("script, style, iframe, noscript, .advert, [data-ad-unit], [class*='adv'], .readmore, .article-tags, .tolstoy, .recomend, .podelitsa, .ya-share2, .all-news, .news-lenta, #Tolstoy, #smi24, #adfox_162463494964389116, #adfox_162463498674745378, #adfox_162918969521962546").Remove()
		html, _ := textBlock.Html()
		body = strings.TrimSpace(html)
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
