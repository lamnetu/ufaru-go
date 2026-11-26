package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
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

	console := publishers.NewConsole()
	rest := publishers.NewRest(
		getEnv("API_ENDPOINT", "http://symfony:80/api/internal/news"),
		getEnv("API_KEY", ""),
	)

	// Подписки: образец типа определяет, на какое событие реагирует handler.
	bus.Subscribe(&event.NewsParsedEvent{}, console.Handle)
	bus.Subscribe(&event.NewsParsedEvent{}, rest.Handle)
	bus.Subscribe(&event.NewsParseFailedEvent{}, console.Handle)

	log.Println("Start parsing")
	svc := parsing.NewService(bus, parsers.NewUfa1(), parsers.NewPrufy())

	go healthServer(getEnv("HEALTH_ADDR", ":8081"))

	interval := getDuration("PARSE_INTERVAL", 5*time.Minute)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	svc.Run(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("shutdown")
			return
		case <-ticker.C:
			svc.Run(ctx)
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

// healthServer поднимает /healthz для Docker HEALTHCHECK.
func healthServer(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		log.Printf("health server: %v", err)
	}
}