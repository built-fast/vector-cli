package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"

	"github.com/built-fast/vector-cli/internal/appctx"
	"github.com/built-fast/vector-cli/internal/config"
	"github.com/built-fast/vector-cli/internal/output"
	"github.com/built-fast/vector-cli/internal/version"
)

func TestMain(m *testing.M) {
	keyring.MockInit()
	os.Exit(m.Run())
}

func TestNewRootCmd_Use(t *testing.T) {
	cmd := NewRootCmd()
	assert.Equal(t, "vector", cmd.Use)
}

func TestNewRootCmd_VersionFlag(t *testing.T) {
	keyring.MockInit()
	origVersion, origCommit, origDate := version.Version, version.Commit, version.Date
	t.Cleanup(func() {
		version.Version = origVersion
		version.Commit = origCommit
		version.Date = origDate
	})

	version.Version = "1.2.3"
	version.Commit = "abc1234"
	version.Date = "2026-01-01"

	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())

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
		{"token flag", "token", true, ""},
		{"json flag", "json", true, "false"},
		{"no-json flag", "no-json", true, "false"},
		{"jq flag", "jq", true, ""},
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
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Usage:")
	assert.Contains(t, out, "vector")
	assert.Contains(t, out, "--json")
	assert.Contains(t, out, "--no-json")
	assert.Contains(t, out, "--version")
	assert.Contains(t, out, "--token")
}

func TestPersistentPreRunE_LoadsDefaultConfig(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())

	var captured *appctx.App
	cmd := NewRootCmd()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		captured = appctx.FromContext(cmd.Context())
		return nil
	}
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, "https://api.builtfast.com", captured.Config.ApiURL)
}

func TestPersistentPreRunE_TokenFromFlag(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())

	var captured *appctx.App
	cmd := NewRootCmd()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		captured = appctx.FromContext(cmd.Context())
		return nil
	}
	cmd.SetArgs([]string{"--token", "flag-token"})

	err := cmd.Execute()
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, "flag-token", captured.Client.Token)
	assert.Equal(t, "flag", captured.TokenSource)
}

func TestPersistentPreRunE_TokenFromEnv(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())
	t.Setenv("VECTOR_API_KEY", "env-token")

	var captured *appctx.App
	cmd := NewRootCmd()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		captured = appctx.FromContext(cmd.Context())
		return nil
	}
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, "env-token", captured.Client.Token)
	assert.Equal(t, "env", captured.TokenSource)
}

func TestPersistentPreRunE_TokenFromKeyring(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())
	t.Setenv("VECTOR_NO_KEYRING", "")

	// Store token in keyring
	require.NoError(t, config.Save("stored-token"))

	var captured *appctx.App
	cmd := NewRootCmd()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		captured = appctx.FromContext(cmd.Context())
		return nil
	}
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, "stored-token", captured.Client.Token)
	assert.Equal(t, "keyring", captured.TokenSource)
}

func TestPersistentPreRunE_TokenPrecedence(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())
	t.Setenv("VECTOR_API_KEY", "env-token")
	t.Setenv("VECTOR_NO_KEYRING", "")

	// Store token in keyring
	require.NoError(t, config.Save("stored-token"))

	var captured *appctx.App
	cmd := NewRootCmd()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		captured = appctx.FromContext(cmd.Context())
		return nil
	}
	// --token flag takes precedence over env and stored credentials
	cmd.SetArgs([]string{"--token", "flag-token"})

	err := cmd.Execute()
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, "flag-token", captured.Client.Token)
	assert.Equal(t, "flag", captured.TokenSource)
}

func TestPersistentPreRunE_NoTokenIsOK(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())

	var captured *appctx.App
	cmd := NewRootCmd()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		captured = appctx.FromContext(cmd.Context())
		return nil
	}
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Empty(t, captured.Client.Token)
	assert.Empty(t, captured.TokenSource)
}

func TestPersistentPreRunE_KeyringDisabledNoToken(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())
	t.Setenv("VECTOR_NO_KEYRING", "1")

	var captured *appctx.App
	cmd := NewRootCmd()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		captured = appctx.FromContext(cmd.Context())
		return nil
	}
	cmd.SetArgs([]string{})

	// Commands that don't require auth still work without a token
	err := cmd.Execute()
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Empty(t, captured.Client.Token)
	assert.Empty(t, captured.TokenSource)
}

func TestPersistentPreRunE_KeyringDisabledFlagTokenWorks(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())
	t.Setenv("VECTOR_NO_KEYRING", "1")

	var captured *appctx.App
	cmd := NewRootCmd()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		captured = appctx.FromContext(cmd.Context())
		return nil
	}
	cmd.SetArgs([]string{"--token", "flag-token"})

	err := cmd.Execute()
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, "flag-token", captured.Client.Token)
	assert.Equal(t, "flag", captured.TokenSource)
}

func TestPersistentPreRunE_KeyringDisabledEnvTokenWorks(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())
	t.Setenv("VECTOR_NO_KEYRING", "1")
	t.Setenv("VECTOR_API_KEY", "env-token")

	var captured *appctx.App
	cmd := NewRootCmd()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		captured = appctx.FromContext(cmd.Context())
		return nil
	}
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, "env-token", captured.Client.Token)
	assert.Equal(t, "env", captured.TokenSource)
}

func TestPersistentPreRunE_DetectsOutputFormat(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())

	tests := []struct {
		name     string
		args     []string
		expected output.Format
	}{
		{"json flag", []string{"--json"}, output.JSON},
		{"no-json flag", []string{"--no-json"}, output.Table},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var captured *appctx.App
			cmd := NewRootCmd()
			cmd.RunE = func(cmd *cobra.Command, args []string) error {
				captured = appctx.FromContext(cmd.Context())
				return nil
			}
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			require.NoError(t, err)
			require.NotNil(t, captured)
			assert.Equal(t, tt.expected, captured.Output.Format())
		})
	}
}

func TestPersistentPreRunE_InvalidConfigJSON(t *testing.T) {
	keyring.MockInit()
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)

	// Write invalid JSON to config file
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte("{invalid"), 0o644))

	cmd := NewRootCmd()
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON")
}

func TestPersistentPreRunE_CustomAPIURL(t *testing.T) {
	keyring.MockInit()
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)

	// Write custom config
	cfg := config.Config{ApiURL: "https://custom.api.com"}
	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "config.json"), data, 0o644))

	var captured *appctx.App
	cmd := NewRootCmd()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		captured = appctx.FromContext(cmd.Context())
		return nil
	}
	cmd.SetArgs([]string{})

	err = cmd.Execute()
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, "https://custom.api.com", captured.Client.BaseURL)
}

func TestPersistentPreRunE_HelpWorksWithoutCredentials(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Usage:")
}

func TestPersistentPreRunE_VersionWorksWithoutCredentials(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--version"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "vector v")
}

func TestPersistentPreRunE_JQCompilesWithoutError(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())

	var captured *appctx.App
	cmd := NewRootCmd()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		captured = appctx.FromContext(cmd.Context())
		return nil
	}
	cmd.SetArgs([]string{"--jq", ".name"})

	err := cmd.Execute()
	require.NoError(t, err)
	require.NotNil(t, captured)
	require.NotNil(t, captured.Output)
	assert.True(t, captured.Output.HasJQ())
}

func TestPersistentPreRunE_JQForcesJSON(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())

	var captured *appctx.App
	cmd := NewRootCmd()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		captured = appctx.FromContext(cmd.Context())
		return nil
	}
	cmd.SetArgs([]string{"--jq", ".name"})

	err := cmd.Execute()
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, output.JSON, captured.Output.Format())
}

func TestPersistentPreRunE_JQAndNoJSONError(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())

	cmd := NewRootCmd()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return nil
	}
	cmd.SetArgs([]string{"--jq", ".name", "--no-json"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Equal(t, "--jq and --no-json cannot be used together", err.Error())
}

func TestPersistentPreRunE_JQInvalidExpression(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())

	cmd := NewRootCmd()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return nil
	}
	cmd.SetArgs([]string{"--jq", ".[["})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid jq expression")
}

func TestPersistentPreRunE_JQIdentityFilter(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())

	var captured *appctx.App
	cmd := NewRootCmd()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		captured = appctx.FromContext(cmd.Context())
		return nil
	}
	cmd.SetArgs([]string{"--jq", "."})

	err := cmd.Execute()
	require.NoError(t, err)
	require.NotNil(t, captured)
	require.NotNil(t, captured.Output)
	assert.True(t, captured.Output.HasJQ())
	assert.Equal(t, output.JSON, captured.Output.Format())
}

func TestPersistentPreRunE_OutputSetWithoutJQ(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())

	var captured *appctx.App
	cmd := NewRootCmd()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		captured = appctx.FromContext(cmd.Context())
		return nil
	}
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)
	require.NotNil(t, captured)
	require.NotNil(t, captured.Output)
	assert.False(t, captured.Output.HasJQ())
}
