package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecuteHelpOutsideRepository(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code, err := execute([]string{"--help"}, &stdout, &stderr)

	if code != 0 || err != nil {
		t.Fatalf("execute(--help) = code %d, err %v", code, err)
	}
	if !strings.Contains(stdout.String(), "Usage: graphkeeper [options]") {
		t.Fatalf("help output missing usage: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("help wrote to stderr: %q", stderr.String())
	}
}

func TestExecuteVersionOutsideRepository(t *testing.T) {
	var stdout, stderr bytes.Buffer
	previous := version
	version = "test-version"
	t.Cleanup(func() { version = previous })

	code, err := execute([]string{"--version"}, &stdout, &stderr)

	if code != 0 || err != nil {
		t.Fatalf("execute(--version) = code %d, err %v", code, err)
	}
	if got := stdout.String(); got != "graphkeeper test-version\n" {
		t.Fatalf("version output = %q", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("version wrote to stderr: %q", stderr.String())
	}
}

func TestExecuteRejectsUnknownOption(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code, err := execute([]string{"--unknown"}, &stdout, &stderr)

	if code != 2 || err == nil {
		t.Fatalf("execute(--unknown) = code %d, err %v", code, err)
	}
	if !strings.Contains(err.Error(), "Usage: graphkeeper [options]") {
		t.Fatalf("error missing usage: %v", err)
	}
}
