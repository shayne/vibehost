package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestPromptCreateWithReaderDefaultsYesOnEmptyInput(t *testing.T) {
	var out bytes.Buffer
	if !promptCreateWithReader("demo", strings.NewReader("\n"), &out) {
		t.Fatalf("expected empty input to default to yes")
	}
	if got := out.String(); !strings.Contains(got, "App demo does not exist") {
		t.Fatalf("expected prompt to mention app name, got %q", got)
	}
}

func TestPromptCreateWithReaderAcceptsYesVariants(t *testing.T) {
	cases := []string{"y\n", "Y\n", "yes\n", "YES\n", " Yes \n"}
	for _, input := range cases {
		var out bytes.Buffer
		if !promptCreateWithReader("demo", strings.NewReader(input), &out) {
			t.Fatalf("expected %q to be accepted", strings.TrimSpace(input))
		}
	}
}

func TestPromptCreateWithReaderRejectsNo(t *testing.T) {
	cases := []string{"n\n", "no\n", "NO\n", "  no  \n"}
	for _, input := range cases {
		var out bytes.Buffer
		if promptCreateWithReader("demo", strings.NewReader(input), &out) {
			t.Fatalf("expected %q to be rejected", strings.TrimSpace(input))
		}
	}
}

func TestPromptCreateWithReaderEOFAcceptsDefault(t *testing.T) {
	var out bytes.Buffer
	if !promptCreateWithReader("demo", strings.NewReader(""), &out) {
		t.Fatalf("expected EOF to default to yes")
	}
}
