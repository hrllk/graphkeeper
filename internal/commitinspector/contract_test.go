package commitinspector

import "testing"

// The diff window's limits and its clamp were written down three times each --
// here, in internal/app, and in the adapter -- and bypassed by literals in
// three more places, so the copy in this package went unused while the others
// decided the behaviour. Now there is one, and this is where it is tested; the
// test came from the adapter, which no longer owns the rule.
func TestNormalizeDiffWindowFillsDefaults(t *testing.T) {
	window, err := NormalizeDiffWindow(DiffWindowRequest{})
	if err != nil {
		t.Fatalf("an empty request is a request for the defaults, got %v", err)
	}
	if window.MaxLines != DefaultDiffWindowLines || window.MaxBytes != DefaultDiffWindowBytes {
		t.Fatalf("defaults = %#v", window)
	}
	if got := DefaultDiffWindow(); got.MaxLines != DefaultDiffWindowLines || got.MaxBytes != DefaultDiffWindowBytes || got.StartLine != 0 {
		t.Fatalf("DefaultDiffWindow disagrees with the clamp: %#v", got)
	}
}

func TestNormalizeDiffWindowRejectsWhatCannotWork(t *testing.T) {
	for name, request := range map[string]DiffWindowRequest{
		"negative start":     {StartLine: -1},
		"negative lines":     {MaxLines: -1},
		"negative bytes":     {MaxBytes: -1},
		"lines over the max": {MaxLines: MaxDiffWindowLines + 1},
		"bytes over the max": {MaxBytes: MaxDiffWindowBytes + 1},
		// Too small to hold one structural record, which is a configuration
		// error rather than simply a small window.
		"bytes under a record": {MaxBytes: 1},
	} {
		if _, err := NormalizeDiffWindow(request); err == nil || err.Kind != "configuration" {
			t.Errorf("%s: expected a configuration error, got %v", name, err)
		}
	}
}

func TestNormalizeDiffWindowKeepsWhatTheCallerAsked(t *testing.T) {
	request := DiffWindowRequest{StartLine: 40, MaxLines: 10, MaxBytes: 4096}
	window, err := NormalizeDiffWindow(request)
	if err != nil {
		t.Fatalf("a valid window was rejected: %v", err)
	}
	if window != request {
		t.Fatalf("the clamp changed a valid window: %#v -> %#v", request, window)
	}
}
