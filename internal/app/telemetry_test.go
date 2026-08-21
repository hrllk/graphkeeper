package app

import (
	"testing"

	"hrllk/graphkeeper/internal/events"
)

type recordingEventSink struct {
	events []events.Event
	err    error
}

func (s *recordingEventSink) Publish(event events.Event) error {
	s.events = append(s.events, event.Copy())
	return s.err
}

func TestModelPublishesThroughInjectedEventSink(t *testing.T) {
	sink := &recordingEventSink{}
	m := model{eventSink: sink}
	m.publish("app", "test_event", map[string]string{"key": "value"})
	if len(sink.events) != 1 {
		t.Fatalf("events = %d, want 1", len(sink.events))
	}
	if got := sink.events[0]; got.Source != "app" || got.Name != "test_event" || got.Fields["key"] != "value" {
		t.Fatalf("unexpected event: %#v", got)
	}
}

func TestModelIgnoresSinkFailure(t *testing.T) {
	sink := &recordingEventSink{err: errSinkUnavailable{}}
	m := model{eventSink: sink}
	m.publish("app", "test_event", nil)
}

type errSinkUnavailable struct{}

func (errSinkUnavailable) Error() string { return "unavailable" }
