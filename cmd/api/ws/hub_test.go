package ws

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestPublishEvent(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()
	sub := rdb.Subscribe(ctx, "events")
	defer sub.Close()
	ch := sub.Channel()

	ev := Event{Type: "test", Data: map[string]string{"a": "b"}}
	PublishEvent(ctx, rdb, ev)

	msg := <-ch
	var got Event
	if err := json.Unmarshal([]byte(msg.Payload), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Type != ev.Type {
		t.Fatalf("want %s got %s", ev.Type, got.Type)
	}
}
