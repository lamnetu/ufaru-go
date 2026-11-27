# Ufaru Go Parser

Микросервис для автоматического парсинга новостей с городских порталов Уфы. Часть системы городского портала с автономным сбором и обработкой новостей.

## Архитектура

Go-парсер собирает новости с источников → публикует в Kafka → Python-сервис переписывает через GigaChat → Symfony отображает на портале.

## Возможности

- Парсинг нескольких источников (ufa1.ru, prufy.ru)
- Извлечение полного текста статей с детальных страниц
- Нормализация дат (разные форматы, часовые пояса)
- Категоризация новостей
- Rate limiting (лимиты на час/день)
- Graceful shutdown
- Интеграция с Kafka для асинхронной обработки

## Технологии

- Go 1.27 — основной язык
- Colly — веб-скрейпинг
- Sarama — Kafka клиент
- Docker — контейнеризация
- Kafka — брокер сообщений
- AKHQ — UI для Kafka

## Структура проекта

```text
.
├── cmd/parser/          # Точка входа
├── config/              # Конфигурация (env)
├── domain/              # Доменные модели
├── event/               # События
├── services/
│   ├── app/             # Композиционный корень
│   ├── eventbus/        # Шина событий
│   ├── parsers/         # Парсеры источников
│   ├── parsing/         # Сервис оркестрации
│   ├── publishers/      # Публикаторы (Console, Kafka, REST)
│   └── ratelimiter/     # Лимиты запросов
├── docker-compose.yml   # Оркестрация
└── Makefile             # Команды
```

## Быстрый старт

### Требования

- Docker & Docker Compose
- Go 1.27+ (для локальной разработки)

### Запуск

```bash
git clone https://github.com/lamnetu/ufaru-go.git
cd ufaru-go
cp .env.example .env
make dev
```

### Проверка

```bash
docker compose logs -f parser
```

Открыть AKHQ: http://localhost:8080

### Тесты

```bash
go test -v -count=1 -timeout 300s ./services/parsers
```

## Конфигурация

Переменные окружения (.env):

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| ENABLED_PUBLISHERS | Список паблишеров через запятую | console |
| KAFKA_BROKERS | Адреса Kafka брокеров | kafka:9092 |
| KAFKA_TOPIC | Топик для успешных новостей | news_parsed |
| KAFKA_TOPIC_FAILED | Топик для ошибок | news_parse_failed |
| PARSE_INTERVAL | Интервал парсинга | 10m |
| MAX_PARSES_PER_HOUR | Лимит парсинга в час | 6 |
| MAX_PARSES_PER_DAY | Лимит парсинга в день | 100 |
| API_ENDPOINT | Endpoint Symfony API | http://symfony:80/api/internal/news |
| API_KEY | Ключ API | - |

Пример .env:

```env
ENABLED_PUBLISHERS=console,kafka
KAFKA_BROKERS=kafka:9092
KAFKA_TOPIC=news_parsed
KAFKA_TOPIC_FAILED=news_parse_failed
PARSE_INTERVAL=10m
MAX_PARSES_PER_HOUR=6
MAX_PARSES_PER_DAY=100
```

## Разработка

### Локальный запуск (без Docker)

```bash
ENABLED_PUBLISHERS=console go run ./cmd/parser
ENABLED_PUBLISHERS=console,kafka go run ./cmd/parser
```

### Тестирование

```bash
go test -v -count=1 ./...
go test -v -count=1 -run TestParser ./services/parsers
go test -cover ./services/parsers
```

### Лабораторные эксперименты

```bash
git checkout -b lab/experiment-name
git checkout main
```

## Архитектурные решения

### Почему Go?

- Высокая производительность парсинга (горутины)
- Статическая типизация — меньше багов в продакшене
- Простая контейнеризация (один бинарник)
- Отличная работа с сетью и конкурентностью

### Почему микросервисы?

- Разделение ответственности: парсер не знает про AI, AI не знает про вёрстку
- Независимое масштабирование: можно запустить 10 парсеров и 1 AI-сервис
- Отказоустойчивость: падение AI не останавливает парсинг
- Разные языки: Go для парсинга, Python для AI, PHP для веба

### Почему Kafka?

- Буферизация между парсером и AI (AI может быть медленнее)
- Гарантированная доставка (retry, DLQ)
- Возможность подключить новых консьюмеров без изменения парсера
- Просмотр сообщений через AKHQ для отладки

## Обработка ошибок

- Сбой одного источника не валит остальные
- Rate limiting защищает от банов
- Graceful shutdown при SIGTERM (Docker stop)
- Circuit breaker для Kafka producer

## Производительность

- ~40 новостей за проход (одна страница ленты)
- ~30-90 секунд на проход (с обходом деталей)
- 6 проходов в час (настройка по умолчанию)
- ~240 новостей в час, ~5760 в день (теоретический максимум)

## Лицензия

MIT

## Автор

Timur Giliazov