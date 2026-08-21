package events

// EventSink is the application-owned boundary for best-effort telemetry.
type EventSink interface {
	Publish(Event) error
}

// Event is a neutral telemetry event produced by application and domain code.
type Event struct {
	Source string
	Name   string
	Fields map[string]string
}

// NoopSink disables telemetry without affecting business operations.
type NoopSink struct{}

func (NoopSink) Publish(Event) error { return nil }

// Copy returns an event with an independent fields map.
func (e Event) Copy() Event {
	fields := make(map[string]string, len(e.Fields))
	for key, value := range e.Fields {
		fields[key] = value
	}
	e.Fields = fields
	return e
}
