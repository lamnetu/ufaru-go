package parsers

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/lamnetu/ufaru-go/domain"
)

const userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36"

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

var ruMonths = map[string]string{
	"января": "January", "февраля": "February", "марта": "March",
	"апреля": "April", "мая": "May", "июня": "June",
	"июля": "July", "августа": "August", "сентября": "September",
	"октября": "October", "ноября": "November", "декабря": "December",
}

func parseRuDate(raw string) (time.Time, bool) {
	s := strings.ToLower(raw)
	for ru, en := range ruMonths {
		s = strings.ReplaceAll(s, ru, en)
	}
	for _, layout := range []string{
		"2 January 2006, 15:04",
		"02 January 2006, 15:04",
		"2 January 2006",
		"02 January 2006",
		time.RFC3339,
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

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

func parseRelativeDate(raw string) (time.Time, bool) {
	s := strings.ToLower(strings.TrimSpace(raw))
	now := time.Now().UTC()
	switch {
	case strings.Contains(s, "минут"):
		return now.Add(-time.Duration(extractNumber(s)) * time.Minute), true
	case strings.Contains(s, "час"):
		return now.Add(-time.Duration(extractNumber(s)) * time.Hour), true
	case strings.Contains(s, "вчера"):
		return now.Add(-24 * time.Hour), true
	case strings.Contains(s, "сегодня"):
		return now, true
	}
	return time.Time{}, false
}

func extractNumber(s string) int {
	var n int
	_, _ = fmt.Sscanf(s, "%d", &n)
	return n
}

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
