package logger

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestSetVerbose(t *testing.T) {
	// Default should be false
	if IsVerbose() {
		t.Error("Default verbose should be false")
	}

	// Enable verbose
	SetVerbose(true)
	if !IsVerbose() {
		t.Error("Verbose should be true after SetVerbose(true)")
	}

	// Disable verbose
	SetVerbose(false)
	if IsVerbose() {
		t.Error("Verbose should be false after SetVerbose(false)")
	}
}

func TestDebug(t *testing.T) {
	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Test with verbose off
	SetVerbose(false)
	Debug("test message %s", "arg")

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stderr = old

	output := buf.String()
	if output != "" {
		t.Errorf("Debug should not output when verbose is false, got: %q", output)
	}

	// Test with verbose on
	r, w, _ = os.Pipe()
	os.Stderr = w

	SetVerbose(true)
	Debug("test message %s", "arg")

	w.Close()
	buf.Reset()
	io.Copy(&buf, r)
	os.Stderr = old

	output = buf.String()
	if !strings.Contains(output, "[DEBUG] test message arg") {
		t.Errorf("Debug output incorrect: got %q", output)
	}

	// Reset
	SetVerbose(false)
}

func TestInfo(t *testing.T) {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	Info("test info %d", 42)

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stderr = old

	output := buf.String()
	if !strings.Contains(output, "test info 42") {
		t.Errorf("Info output incorrect: got %q", output)
	}
}

func TestSuccess(t *testing.T) {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	Success("operation completed")

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stderr = old

	output := buf.String()
	if !strings.Contains(output, "✓ operation completed") {
		t.Errorf("Success output should have checkmark: got %q", output)
	}
}

func TestError(t *testing.T) {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	Error("error occurred: %s", "details")

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stderr = old

	output := buf.String()
	if !strings.Contains(output, "✗ error occurred: details") {
		t.Errorf("Error output should have X mark: got %q", output)
	}
}

func TestWarn(t *testing.T) {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	Warn("warning message")

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stderr = old

	output := buf.String()
	if !strings.Contains(output, "⚠ warning message") {
		t.Errorf("Warn output should have warning symbol: got %q", output)
	}
}

func TestMultipleArgs(t *testing.T) {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	Info("test %s %d %v", "string", 123, true)

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stderr = old

	output := buf.String()
	expected := "test string 123 true"
	if !strings.Contains(output, expected) {
		t.Errorf("Multiple args not formatted correctly: got %q, want substring %q", output, expected)
	}
}
