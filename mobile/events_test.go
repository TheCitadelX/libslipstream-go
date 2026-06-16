package mobile

import (
	"strings"
	"testing"
)

func TestEventQueueNext(t *testing.T) {
	queue := NewEventQueue(2)
	if event := queue.Next(0); event != nil {
		t.Fatalf("expected empty queue, got %#v", event)
	}

	queue.emit("info", "test", "hello")
	event := queue.Next(10)
	if event == nil {
		t.Fatalf("expected event")
	}
	if event.Level != "info" || event.Source != "test" || event.Message != "hello" {
		t.Fatalf("unexpected event: %#v", event)
	}
	if event.AtUnixMillis <= 0 {
		t.Fatalf("expected event timestamp")
	}
}

func TestEventQueueDropsOldestWhenFull(t *testing.T) {
	queue := NewEventQueue(1)
	queue.emit("info", "test", "old")
	queue.emit("info", "test", "new")

	event := queue.Next(0)
	if event == nil {
		t.Fatalf("expected event")
	}
	if event.Message != "new" {
		t.Fatalf("expected newest event, got %q", event.Message)
	}
}

func TestClientEvents(t *testing.T) {
	client, err := NewClient(&ClientConfig{
		ResolversCSV:      "127.0.0.1:1",
		Domain:            "test.com",
		AllowInsecure:     true,
		InitialPacketSize: 1200,
		EventQueueSize:    4,
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if event := client.Events().Next(0); event == nil || event.Message != "client created" {
		t.Fatalf("expected client created event, got %#v", event)
	}

	err = client.Ping(1)
	if err == nil {
		t.Fatalf("expected ping error")
	}
	event := client.Events().Next(0)
	if event == nil {
		t.Fatalf("expected ping failure event")
	}
	if event.Level != "error" || !strings.Contains(event.Message, "client not connected") {
		t.Fatalf("unexpected ping event: %#v", event)
	}
}
