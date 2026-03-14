package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/built-fast/vector-cli/internal/version"
)

func TestNewRootCmd_Use(t *testing.T) {
	cmd := NewRootCmd()
	assert.Equal(t, "vector", cmd.Use)
}

func TestNewRootCmd_VersionFlag(t *testing.T) {
	origVersion, origCommit, origDate := version.Version, version.Commit, version.Date
	t.Cleanup(func() {
		version.Version = origVersion
		version.Commit = origCommit
		version.Date = origDate
	})

	version.Version = "1.2.3"
	version.Commit = "abc1234"
	version.Date = "2026-01-01"

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--version"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "vector v1.2.3 (abc1234) built 2026-01-01", strings.TrimSpace(buf.String()))
}

func TestNewRootCmd_FlagsRegistered(t *testing.T) {
	cmd := NewRootCmd()

	tests := []struct {
		name       string
		flag       string
		persistent bool
		defValue   string
	}{
		{"version flag", "version", false, "false"},
		{"json flag", "json", true, "false"},
		{"no-json flag", "no-json", true, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f = cmd.Flags().Lookup(tt.flag)
			if tt.persistent {
				f = cmd.PersistentFlags().Lookup(tt.flag)
			}
			require.NotNil(t, f, "--%s flag should be registered", tt.flag)
			assert.Equal(t, tt.defValue, f.DefValue)
		})
	}
}

func TestNewRootCmd_NoArgsShowsHelp(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Usage:")
	assert.Contains(t, output, "vector")
	assert.Contains(t, output, "--json")
	assert.Contains(t, output, "--no-json")
	assert.Contains(t, output, "--version")
}
