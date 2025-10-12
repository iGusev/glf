package logger

import (
	"fmt"
	"os"
)

var verbose bool

// SetVerbose enables or disables verbose logging
func SetVerbose(v bool) {
	verbose = v
}

// IsVerbose returns true if verbose logging is enabled
func IsVerbose() bool {
	return verbose
}

// Debug prints debug messages only when verbose mode is enabled
func Debug(format string, args ...interface{}) {
	if verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// Info prints informational messages
func Info(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// Success prints success messages with checkmark
func Success(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "✓ "+format+"\n", args...)
}

// Error prints error messages
func Error(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "✗ "+format+"\n", args...)
}

// Warn prints warning messages
func Warn(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "⚠ "+format+"\n", args...)
}
