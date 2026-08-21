package events

import "testing"

func TestEventSinkContractCopiesFields(t *testing.T) {
	fields := map[string]string{"branch": "main"}
	event := (Event{Source: "app", Name: "loaded", Fields: fields}).Copy()
	fields["branch"] = "changed"
	if event.Fields["branch"] != "main" {
		t.Fatalf("event fields were not copied: %#v", event.Fields)
	}
}

func TestNoopSinkAcceptsEvents(t *testing.T) {
	if err := (NoopSink{}).Publish(Event{Source: "app", Name: "loaded"}); err != nil {
		t.Fatalf("no-op sink returned error: %v", err)
	}
}
