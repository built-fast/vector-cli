package commands

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWafCmd_HelpText(t *testing.T) {
	cmd := NewWafCmd()

	stdout := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "rate-limit")
	assert.Contains(t, out, "blocked-ip")
	assert.Contains(t, out, "blocked-referrer")
	assert.Contains(t, out, "allowed-referrer")
	assert.Contains(t, out, "WAF")
}
