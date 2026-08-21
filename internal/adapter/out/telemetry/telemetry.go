package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"hrllk/graphkeeper/internal/events"
)

const (
	maxSourceNameBytes = 64
	maxFieldKeyBytes   = 64
	maxFieldValueBytes = 512
)

var ErrInvalidEvent = errors.New("invalid telemetry event")

// ValidationError classifies events rejected before any output is written.
type ValidationError struct{ Field string }

func (e *ValidationError) Error() string { return fmt.Sprintf("%s: %s", ErrInvalidEvent, e.Field) }
func (e *ValidationError) Unwrap() error { return ErrInvalidEvent }

type eventLine struct {
	Time   string            `json:"time"`
	Source string            `json:"source"`
	Name   string            `json:"name"`
	Fields map[string]string `json:"fields"`
}

type Sink struct {
	path string
	mu   sync.Mutex
}

func New(path string) *Sink { return &Sink{path: path} }

func (s *Sink) Publish(event events.Event) error {
	event = event.Copy()
	if err := validate(event); err != nil {
		return err
	}
	fields := make(map[string]string, len(event.Fields))
	for key, value := range event.Fields {
		if isSecret(key) {
			continue
		}
		fields[key] = value
	}
	payload, err := json.Marshal(eventLine{Time: time.Now().UTC().Format(time.RFC3339Nano), Source: event.Source, Name: event.Name, Fields: fields})
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	file, err := os.OpenFile(filepath.Clean(s.path), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(append(payload, '\n'))
	return err
}

func validate(event events.Event) error {
	if event.Source == "" || len([]byte(event.Source)) > maxSourceNameBytes {
		return &ValidationError{Field: "source"}
	}
	if event.Name == "" || len([]byte(event.Name)) > maxSourceNameBytes {
		return &ValidationError{Field: "name"}
	}
	for key, value := range event.Fields {
		if key == "" || len([]byte(key)) > maxFieldKeyBytes {
			return &ValidationError{Field: "field key"}
		}
		if len([]byte(value)) > maxFieldValueBytes {
			return &ValidationError{Field: "field value"}
		}
	}
	return nil
}

func isSecret(key string) bool {
	key = strings.ToLower(key)
	switch key {
	case "password", "token", "secret", "credential", "authorization", "api_key", "access_token":
		return true
	}
	for _, suffix := range []string{"_token", "_secret", "_password", "_key"} {
		if strings.HasSuffix(key, suffix) {
			return true
		}
	}
	return false
}
