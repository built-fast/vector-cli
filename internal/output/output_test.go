package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectFormat_JSONFlag(t *testing.T) {
	// --json flag always returns JSON, regardless of TTY state.
	isTerminalFunc = func() bool { return true }
	t.Cleanup(func() { isTerminalFunc = func() bool { return false } })

	assert.Equal(t, JSON, DetectFormat(true, false))
}

func TestDetectFormat_NoJSONFlag(t *testing.T) {
	// --no-json flag always returns Table, regardless of TTY state.
	isTerminalFunc = func() bool { return false }
	t.Cleanup(func() { isTerminalFunc = func() bool { return false } })

	assert.Equal(t, Table, DetectFormat(false, true))
}

func TestDetectFormat_BothFlags_JSONWins(t *testing.T) {
	// When both flags are set, --json takes precedence.
	assert.Equal(t, JSON, DetectFormat(true, true))
}

func TestDetectFormat_NoFlags_TTY(t *testing.T) {
	// No flags, stdout is a TTY → Table.
	isTerminalFunc = func() bool { return true }
	t.Cleanup(func() { isTerminalFunc = func() bool { return false } })

	assert.Equal(t, Table, DetectFormat(false, false))
}

func TestDetectFormat_NoFlags_NonTTY(t *testing.T) {
	// No flags, stdout is not a TTY (piped) → JSON.
	isTerminalFunc = func() bool { return false }

	assert.Equal(t, JSON, DetectFormat(false, false))
}

func TestFormatConstants(t *testing.T) {
	// Verify constants have distinct values.
	assert.NotEqual(t, Table, JSON)
}
