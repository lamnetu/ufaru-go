package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	EnabledPublishers []string
	KafkaBrokers      []string
	KafkaTopic        string
	KafkaTopicFailed  string
	APIEndpoint       string
	APIKey            string
	ParseInterval     time.Duration
	MaxPerHour        int
	MaxPerDay         int
}

func Load() Config {
	return Config{
		EnabledPublishers: splitEnv("ENABLED_PUBLISHERS", "console"),
		KafkaBrokers:      splitEnv("KAFKA_BROKERS", "kafka:9092"),
		KafkaTopic:        env("KAFKA_TOPIC", "news_parsed"),
		KafkaTopicFailed:  env("KAFKA_TOPIC_FAILED", "news_parse_failed"),
		APIEndpoint:       env("API_ENDPOINT", "http://symfony:80/api/internal/news"),
		APIKey:            env("API_KEY", ""),
		ParseInterval:     durationEnv("PARSE_INTERVAL", 10*time.Minute),
		MaxPerHour:        intEnv("MAX_PARSES_PER_HOUR", 6),
		MaxPerDay:         intEnv("MAX_PARSES_PER_DAY", 100),
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func splitEnv(key, def string) []string {
	raw := env(key, def)
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func durationEnv(key string, def time.Duration) time.Duration {
	if v, err := time.ParseDuration(os.Getenv(key)); err == nil {
		return v
	}
	return def
}

func intEnv(key string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return v
	}
	return def
}
