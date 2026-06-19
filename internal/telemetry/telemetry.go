package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var mu sync.Mutex

type Event struct {
	Time   string            `json:"time"`
	Source string            `json:"source"`
	Name   string            `json:"name"`
	Fields map[string]string `json:"fields,omitempty"`
}

func Log(source, name string, fields map[string]string) {
	event := Event{
		Time:   time.Now().UTC().Format(time.RFC3339Nano),
		Source: source,
		Name:   name,
		Fields: fields,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}

	path := filepath.Join(os.TempDir(), "git-graph-tui-events.jsonl")

	mu.Lock()
	defer mu.Unlock()

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer file.Close()

	_, _ = file.Write(append(payload, '\n'))
}
