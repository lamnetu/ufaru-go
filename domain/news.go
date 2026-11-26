package domain

import "time"

// Source — типизированный идентификатор источника новости.
type Source string

const (
	SourceUfa1  Source = "ufa1"
	SourcePrufy Source = "prufy"
)

// Category — типизированная категория новости.
type Category string

const (
	CategoryCity      Category = "city"
	CategoryPolitics  Category = "politics"
	CategoryEconomy   Category = "economy"
	CategorySport     Category = "sport"
	CategoryCulture   Category = "culture"
	CategoryIncidents Category = "incidents"
)

// News — чистая доменная модель новости.
// Без json-тегов и привязки к БД: (де)сериализация — забота инфраструктуры.
type News struct {
	Title       string
	Description string
	Content     string // полный текст статьи
	URL         string // ссылка на первоисточник
	ImageURL    string
	Source      Source
	Category    Category
	PublishedAt time.Time // дата публикации на источнике
	ParsedAt    time.Time // дата парсинга нами
}

// Valid — доменное правило: новость можно публиковать,
// только если есть заголовок и ссылка на первоисточник.
func (n *News) Valid() bool {
	return n != nil && n.Title != "" && n.URL != ""
}