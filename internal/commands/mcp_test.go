package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/appctx"
	"github.com/built-fast/vector-cli/internal/config"
	"github.com/built-fast/vector-cli/internal/output"
)

func buildMcpCmd(token string, format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	root := &cobra.Command{
		Use: "vector",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			client := api.NewClient("http://localhost", token, "test-agent")
			app := appctx.NewApp(
				config.DefaultConfig(),
				client,
				"",
			)
			cmd.SetContext(appctx.WithApp(cmd.Context(), app))
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	mcpCmd := NewMcpCmd()
	root.AddCommand(mcpCmd)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

func buildMcpCmdNoAuth(format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	root := &cobra.Command{
		Use: "vector",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			client := api.NewClient("http://localhost", "", "test-agent")
			app := appctx.NewApp(
				config.DefaultConfig(),
				client,
				"",
			)
			cmd.SetContext(appctx.WithApp(cmd.Context(), app))
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	mcpCmd := NewMcpCmd()
	root.AddCommand(mcpCmd)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

// --- Auth Tests ---

func TestMcpSetupCmd_NoAuthToken(t *testing.T) {
	cmd, _, _ := buildMcpCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"mcp", "setup"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

// --- Desktop Target Tests ---

func TestMcpSetupCmd_DesktopNewConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "Claude", "claude_desktop_config.json")

	// Patch claudeDesktopConfigPath for test
	origFn := claudeDesktopConfigPathFn
	claudeDesktopConfigPathFn = func() (string, error) { return configPath, nil }
	defer func() { claudeDesktopConfigPathFn = origFn }()

	cmd, stdout, _ := buildMcpCmd("test-token-123", output.Table)
	cmd.SetArgs([]string{"mcp", "setup"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "Vector MCP server added in Claude Desktop config.")
	assert.Contains(t, out, "Config written to: "+configPath)
	assert.Contains(t, out, "Restart Claude Desktop to apply changes.")

	// Verify file contents
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var cfg map[string]any
	require.NoError(t, json.Unmarshal(data, &cfg))

	mcpServers := cfg["mcpServers"].(map[string]any)
	vector := mcpServers["vector"].(map[string]any)
	assert.Equal(t, "npx", vector["command"])
	assert.Nil(t, vector["type"]) // Desktop should NOT have type field

	args := vector["args"].([]any)
	assert.Equal(t, "-y", args[0])
	assert.Equal(t, "mcp-remote", args[1])
	assert.Equal(t, "https://api.builtfast.com/mcp/vector", args[2])
	assert.Equal(t, "--header", args[3])
	assert.Equal(t, "Authorization: Bearer test-token-123", args[4])
}

func TestMcpSetupCmd_DesktopPreservesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "Claude", "claude_desktop_config.json")

	// Create existing config with another MCP server
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0o755))
	existing := map[string]any{
		"mcpServers": map[string]any{
			"other-server": map[string]any{
				"command": "other",
				"args":    []string{"arg1"},
			},
		},
		"otherSetting": "preserved",
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	require.NoError(t, os.WriteFile(configPath, data, 0o644))

	origFn := claudeDesktopConfigPathFn
	claudeDesktopConfigPathFn = func() (string, error) { return configPath, nil }
	defer func() { claudeDesktopConfigPathFn = origFn }()

	cmd, _, _ := buildMcpCmd("test-token", output.Table)
	cmd.SetArgs([]string{"mcp", "setup"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify both servers exist and other settings preserved
	fileData, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var cfg map[string]any
	require.NoError(t, json.Unmarshal(fileData, &cfg))

	assert.Equal(t, "preserved", cfg["otherSetting"])
	mcpServers := cfg["mcpServers"].(map[string]any)
	assert.Contains(t, mcpServers, "other-server")
	assert.Contains(t, mcpServers, "vector")
}

func TestMcpSetupCmd_DesktopAlreadyConfigured(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "Claude", "claude_desktop_config.json")

	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0o755))
	existing := map[string]any{
		"mcpServers": map[string]any{
			"vector": map[string]any{
				"command": "npx",
				"args":    []string{"old-args"},
			},
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	require.NoError(t, os.WriteFile(configPath, data, 0o644))

	origFn := claudeDesktopConfigPathFn
	claudeDesktopConfigPathFn = func() (string, error) { return configPath, nil }
	defer func() { claudeDesktopConfigPathFn = origFn }()

	cmd, _, _ := buildMcpCmd("test-token", output.Table)
	cmd.SetArgs([]string{"mcp", "setup"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Vector MCP server already configured. Use --force to overwrite")
}

func TestMcpSetupCmd_DesktopForceOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "Claude", "claude_desktop_config.json")

	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0o755))
	existing := map[string]any{
		"mcpServers": map[string]any{
			"vector": map[string]any{
				"command": "npx",
				"args":    []string{"old-args"},
			},
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	require.NoError(t, os.WriteFile(configPath, data, 0o644))

	origFn := claudeDesktopConfigPathFn
	claudeDesktopConfigPathFn = func() (string, error) { return configPath, nil }
	defer func() { claudeDesktopConfigPathFn = origFn }()

	cmd, stdout, _ := buildMcpCmd("new-token", output.Table)
	cmd.SetArgs([]string{"mcp", "setup", "--force"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "Vector MCP server updated in Claude Desktop config.")

	// Verify updated config
	fileData, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var cfg map[string]any
	require.NoError(t, json.Unmarshal(fileData, &cfg))

	mcpServers := cfg["mcpServers"].(map[string]any)
	vector := mcpServers["vector"].(map[string]any)
	args := vector["args"].([]any)
	assert.Equal(t, "Authorization: Bearer new-token", args[4])
}

// --- Code Target Tests ---

func TestMcpSetupCmd_CodeProjectLevel(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".mcp.json")

	// Change to temp dir so .mcp.json is created there
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	cmd, stdout, _ := buildMcpCmd("test-token-456", output.Table)
	cmd.SetArgs([]string{"mcp", "setup", "--target", "code"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "Vector MCP server added in Claude Code config.")
	assert.Contains(t, out, "Config written to: .mcp.json")
	// Should NOT contain restart message for project-level
	assert.NotContains(t, out, "Restart")

	// Verify file contents
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var cfg map[string]any
	require.NoError(t, json.Unmarshal(data, &cfg))

	mcpServers := cfg["mcpServers"].(map[string]any)
	vector := mcpServers["vector"].(map[string]any)
	assert.Equal(t, "stdio", vector["type"])
	assert.Equal(t, "npx", vector["command"])

	args := vector["args"].([]any)
	assert.Equal(t, "Authorization: Bearer test-token-456", args[4])
}

func TestMcpSetupCmd_CodeGlobal(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".claude.json")

	// Override HOME for the test
	origHome := os.Getenv("HOME")
	require.NoError(t, os.Setenv("HOME", tmpDir))
	defer func() { _ = os.Setenv("HOME", origHome) }()

	cmd, stdout, _ := buildMcpCmd("test-token-789", output.Table)
	cmd.SetArgs([]string{"mcp", "setup", "--target", "code", "--global"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "Vector MCP server added in Claude Code config.")
	assert.Contains(t, out, configPath)
	assert.Contains(t, out, "Restart Claude Code to apply changes.")

	// Verify file contents
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var cfg map[string]any
	require.NoError(t, json.Unmarshal(data, &cfg))

	mcpServers := cfg["mcpServers"].(map[string]any)
	vector := mcpServers["vector"].(map[string]any)
	assert.Equal(t, "stdio", vector["type"])
}

func TestMcpSetupCmd_CodeAlreadyConfigured(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".mcp.json")

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	existing := map[string]any{
		"mcpServers": map[string]any{
			"vector": map[string]any{
				"type":    "stdio",
				"command": "npx",
				"args":    []string{"old-args"},
			},
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	require.NoError(t, os.WriteFile(configPath, data, 0o644))

	cmd, _, _ := buildMcpCmd("test-token", output.Table)
	cmd.SetArgs([]string{"mcp", "setup", "--target", "code"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Vector MCP server already configured. Use --force to overwrite")
}

func TestMcpSetupCmd_CodeForceOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".mcp.json")

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	existing := map[string]any{
		"mcpServers": map[string]any{
			"vector": map[string]any{
				"type":    "stdio",
				"command": "npx",
				"args":    []string{"old-args"},
			},
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	require.NoError(t, os.WriteFile(configPath, data, 0o644))

	cmd, stdout, _ := buildMcpCmd("new-token", output.Table)
	cmd.SetArgs([]string{"mcp", "setup", "--target", "code", "--force"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "Vector MCP server updated in Claude Code config.")
}

// --- Flag Validation Tests ---

func TestMcpSetupCmd_GlobalWithDesktopErrors(t *testing.T) {
	cmd, _, _ := buildMcpCmd("test-token", output.Table)
	cmd.SetArgs([]string{"mcp", "setup", "--target", "desktop", "--global"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--global flag only applies when --target is \"code\"")
}

func TestMcpSetupCmd_InvalidTarget(t *testing.T) {
	cmd, _, _ := buildMcpCmd("test-token", output.Table)
	cmd.SetArgs([]string{"mcp", "setup", "--target", "invalid"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid target")
}

// --- Help Text Tests ---

func TestMcpCmd_HelpText(t *testing.T) {
	cmd, stdout, _ := buildMcpCmd("test-token", output.Table)
	cmd.SetArgs([]string{"mcp", "--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "setup")
}

func TestMcpSetupCmd_HelpText(t *testing.T) {
	cmd, stdout, _ := buildMcpCmd("test-token", output.Table)
	cmd.SetArgs([]string{"mcp", "setup", "--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "--target")
	assert.Contains(t, out, "--global")
	assert.Contains(t, out, "--force")
}
