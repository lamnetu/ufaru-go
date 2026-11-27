package parsers

import (
	"context"
	"testing"
)

func TestUfa1Lab(t *testing.T) {
	news, err := NewUfa1().ParseNewsList(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range news {
		t.Logf("%s | content=%d байт", n.Title, len(n.Content))
	}
}
