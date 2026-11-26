package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lamnetu/ufaru-go/event"
	"github.com/lamnetu/ufaru-go/eventbus"
	"github.com/lamnetu/ufaru-go/services/parsers"
	"github.com/lamnetu/ufaru-go/services/parsing"
	"github.com/lamnetu/ufaru-go/services/publishers"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	bus := eventbus.New()

	// Читаем env
	enabledPublishers := strings.Split(getEnv("ENABLED_PUBLISHERS", "console"), ",")
	kafkaBrokers := strings.Split(getEnv("KAFKA_BROKERS", "kafka:9092"), ",")
	kafkaTopic := getEnv("KAFKA_TOPIC", "news_parsed")
	kafkaTopicFailed := getEnv("KAFKA_TOPIC_FAILED", "news_parse_failed")

	var closers []interface{ Close() error }

	for _, name := range enabledPublishers {
		name = strings.TrimSpace(name)
		switch name {
		case "console":
			console := publishers.NewConsole()
			bus.Subscribe(&event.NewsParsedEvent{}, console.Handle)
			bus.Subscribe(&event.NewsParseFailedEvent{}, console.Handle)
			log.Println("[main] enabled publisher: console")

		case "kafka":
			kafka, err := publishers.NewKafka(kafkaBrokers, kafkaTopic, kafkaTopicFailed)
			if err != nil {
				log.Fatalf("[main] kafka init: %v", err)
			}
			bus.Subscribe(&event.NewsParsedEvent{}, kafka.Handle)
			bus.Subscribe(&event.NewsParseFailedEvent{}, kafka.Handle)
			closers = append(closers, kafka)
			log.Println("[main] enabled publisher: kafka")
		case "rest":
			rest := publishers.NewRest(
				getEnv("API_ENDPOINT", "http://symfony:80/api/internal/news"),
				getEnv("API_KEY", ""),
			)
			bus.Subscribe(&event.NewsParsedEvent{}, rest.Handle)
			bus.Subscribe(&event.NewsParseFailedEvent{}, rest.Handle)
			log.Println("[main] enabled publisher: rest")
		}
	}

	svc := parsing.NewService(bus, parsers.NewUfa1(), parsers.NewPrufy())

	interval := getDuration("PARSE_INTERVAL", 10*time.Minute)
	maxPerHour := getInt("MAX_PARSES_PER_HOUR", 6)
	maxPerDay := getInt("MAX_PARSES_PER_DAY", 100)

	log.Printf("[main] parse interval=%s, max/hour=%d, max/day=%d", interval, maxPerHour, maxPerDay)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Rate limiter: счётчики парсинга
	var (
		hourlyCount int
		dailyCount  int
		hourReset   = time.NewTicker(1 * time.Hour)
		dayReset    = time.NewTicker(24 * time.Hour)
	)
	defer hourReset.Stop()
	defer dayReset.Stop()

	svc.Run(ctx) // первый проход сразу
	hourlyCount++
	dailyCount++

	log.Printf("[main] after svc.Run(ctx) hourlyCount=%d, dailyCount=%d", hourlyCount, dailyCount)

	for {
		select {
		case <-ctx.Done():
			log.Println("[main] shutdown")
			for _, c := range closers {
				c.Close()
			}
			return

		case <-hourReset.C:
			log.Printf("[main] hourly reset (was %d)", hourlyCount)
			hourlyCount = 0

		case <-dayReset.C:
			log.Printf("[main] daily reset (was %d)", dailyCount)
			dailyCount = 0

		case <-ticker.C:
			if hourlyCount >= maxPerHour {
				log.Printf("[main] hourly limit reached (%d/%d), skipping", hourlyCount, maxPerHour)
				continue
			}
			if dailyCount >= maxPerDay {
				log.Printf("[main] daily limit reached (%d/%d), skipping", dailyCount, maxPerDay)
				continue
			}
			svc.Run(ctx)
			hourlyCount++
			dailyCount++
		}
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getDuration(key string, def time.Duration) time.Duration {
	if v, err := time.ParseDuration(os.Getenv(key)); err == nil {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	var v int
	if _, err := fmt.Sscanf(os.Getenv(key), "%d", &v); err == nil {
		return v
	}
	return def
}
