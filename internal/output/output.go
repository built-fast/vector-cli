// Package output provides format detection, table/JSON writers, and error envelopes.
package output

import (
	"os"

	"golang.org/x/term"
)

// Format represents an output format.
type Format int

const (
	// Table is human-friendly tabular output.
	Table Format = iota
	// JSON is machine-friendly JSON output.
	JSON
)

// isTerminalFunc is the function used to check if a file descriptor is a terminal.
// It can be overridden in tests to simulate TTY/non-TTY environments.
var isTerminalFunc = func() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// DetectFormat determines the output format based on explicit flags and TTY detection.
// --json flag forces JSON output, --no-json flag forces Table output.
// When neither flag is set, it checks whether stdout is a terminal:
// TTY → Table, non-TTY (piped) → JSON.
func DetectFormat(jsonFlag, noJSONFlag bool) Format {
	if jsonFlag {
		return JSON
	}
	if noJSONFlag {
		return Table
	}
	if isTerminalFunc() {
		return Table
	}
	return JSON
}
