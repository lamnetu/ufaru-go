package parsers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/IBM/sarama"
)

func TestParser(t *testing.T) {
	ctx := context.Background()

	news, err := NewUfa1(1).ParseNewsList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("type=%T len=%d cap=%d\n", news, len(news), cap(news))

	// С хоста идём в EXTERNAL listener localhost:9093
	sp, err := sarama.NewSyncProducer([]string{"localhost:9093"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer sp.Close()

	now := time.Now().UTC()
	for _, n := range news {
		msg := map[string]interface{}{
			"type":        "news_parsed",
			"source":      string(n.Source),
			"title":       n.Title,
			"content":     n.Content,
			"url":         n.URL,
			"category":    string(n.Category),
			"occurred_at": now,
		}
		body, _ := json.Marshal(msg)

		partition, offset, err := sp.SendMessage(&sarama.ProducerMessage{
			Topic: "news_parsed",
			Key:   sarama.StringEncoder(n.URL),
			Value: sarama.ByteEncoder(body),
		})
		if err != nil {
			t.Fatalf("send %s: %v", n.URL, err)
		}
		t.Logf("delivered: partition=%d offset=%d title=%s", partition, offset, n.Title)
	}

	fmt.Printf("published %d news\n", len(news))
}
