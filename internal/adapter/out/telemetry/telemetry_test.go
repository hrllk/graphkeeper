package telemetry

import (
	"errors"
	"os"
	"strings"
	"sync"
	"testing"

	"hrllk/graphkeeper/internal/events"
)

func TestSinkWritesRedactedJSONL(t *testing.T) {
	path := t.TempDir() + "/events.jsonl"
	fields := map[string]string{"branch": "main", "PaSsWoRd": "do-not-write", "client_key": "also-secret", "keyboard": "normal"}
	if err := New(path).Publish(events.Event{Source: "app", Name: "loaded", Fields: fields}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	line := string(data)
	if !strings.HasSuffix(line, "\n") || !strings.Contains(line, `"source":"app"`) || !strings.Contains(line, `"branch":"main"`) || !strings.Contains(line, `"keyboard":"normal"`) {
		t.Fatalf("unexpected JSONL: %s", line)
	}
	if strings.Contains(line, "do-not-write") || strings.Contains(line, "client_key") {
		t.Fatalf("secret leaked: %s", line)
	}
	if fields["PaSsWoRd"] != "do-not-write" {
		t.Fatal("input fields mutated")
	}
}

func TestSinkRejectsInvalidEventWithoutWriting(t *testing.T) {
	path := t.TempDir() + "/events.jsonl"
	err := New(path).Publish(events.Event{Source: "", Name: "loaded"})
	if !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("error = %v, want ErrInvalidEvent", err)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("invalid event created output: %v", statErr)
	}
}

func TestSinkReturnsWriteFailure(t *testing.T) {
	err := New(t.TempDir()).Publish(events.Event{Source: "app", Name: "write_failure"})
	if err == nil {
		t.Fatal("expected write failure")
	}
}

func TestSinkConcurrentPublishesProduceCompleteLines(t *testing.T) {
	path := t.TempDir() + "/events.jsonl"
	sink := New(path)
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := sink.Publish(events.Event{Source: "test", Name: "concurrent", Fields: map[string]string{"value": "ok"}}); err != nil {
				t.Errorf("publish: %v", err)
			}
		}()
	}
	wg.Wait()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(data), "\n"); got != 32 {
		t.Fatalf("lines = %d, want 32", got)
	}
}
