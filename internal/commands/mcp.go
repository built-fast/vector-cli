package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/api"
)

const mcpServerURL = "https://api.builtfast.com/mcp/vector"

// claudeDesktopConfigPathFn is the function used to resolve the desktop config path.
// Override in tests for deterministic paths.
var claudeDesktopConfigPathFn = claudeDesktopConfigPath

// NewMcpCmd creates the mcp command group.
func NewMcpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "MCP server configuration",
		Long:  "Configure the Vector MCP server for use with Claude Desktop or Claude Code.",
	}

	cmd.AddCommand(newMcpSetupCmd())

	return cmd
}

func newMcpSetupCmd() *cobra.Command {
	var target string
	var global bool
	var force bool

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Configure Vector MCP server",
		Long:  "Configure the Vector MCP server in Claude Desktop or Claude Code for AI-assisted site management.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			token := app.Client.Token

			// Validate --global only with --target code
			if global && target != "code" {
				return fmt.Errorf("--global flag only applies when --target is \"code\"")
			}

			// Determine config path
			configPath, err := mcpConfigPath(target, global)
			if err != nil {
				return err
			}

			// Build the MCP server entry
			serverEntry := buildMcpServerEntry(target, token)

			// Read existing config or start fresh
			configData, err := readJSONFile(configPath)
			if err != nil {
				return err
			}

			// Get or create mcpServers
			mcpServers, _ := configData["mcpServers"].(map[string]any)
			if mcpServers == nil {
				mcpServers = map[string]any{}
			}

			// Check if already configured
			action := "added"
			if _, exists := mcpServers["vector"]; exists {
				if !force {
					return &api.APIError{
					Message:  "Vector MCP server already configured. Use --force to overwrite.",
					ExitCode: 1,
				}
				}
				action = "updated"
			}

			// Set the vector entry
			mcpServers["vector"] = serverEntry
			configData["mcpServers"] = mcpServers

			// Create parent directories
			dir := filepath.Dir(configPath)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}

			// Write config
			data, err := json.MarshalIndent(configData, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal config: %w", err)
			}
			data = append(data, '\n')

			if err := os.WriteFile(configPath, data, 0o644); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}

			// Success messages
			w := cmd.OutOrStdout()
			targetLabel := "Claude Desktop"
			if target == "code" {
				targetLabel = "Claude Code"
			}
			_, _ = fmt.Fprintf(w, "Vector MCP server %s in %s config.\n", action, targetLabel)
			_, _ = fmt.Fprintf(w, "Config written to: %s\n", configPath)

			// Restart message: omitted for Code project-level config
			if target == "desktop" || global {
				_, _ = fmt.Fprintf(w, "Restart %s to apply changes.\n", targetLabel)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&target, "target", "desktop", "Target application: \"desktop\" or \"code\"")
	cmd.Flags().BoolVar(&global, "global", false, "Write to global config (~/.claude.json) instead of project-level .mcp.json (only for --target code)")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing Vector MCP configuration")

	return cmd
}

// mcpConfigPath returns the config file path for the given target.
func mcpConfigPath(target string, global bool) (string, error) {
	switch target {
	case "desktop":
		return claudeDesktopConfigPathFn()
	case "code":
		if global {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("unable to determine home directory: %w", err)
			}
			return filepath.Join(home, ".claude.json"), nil
		}
		return ".mcp.json", nil
	default:
		return "", fmt.Errorf("invalid target %q: must be \"desktop\" or \"code\"", target)
	}
}

// claudeDesktopConfigPath returns the platform-specific Claude Desktop config path.
func claudeDesktopConfigPath() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to determine home directory: %w", err)
		}
		return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json"), nil
	case "linux":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to determine home directory: %w", err)
		}
		return filepath.Join(home, ".config", "Claude", "claude_desktop_config.json"), nil
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("%%APPDATA%% is not set")
		}
		return filepath.Join(appData, "Claude", "claude_desktop_config.json"), nil
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// buildMcpServerEntry returns the MCP server config for the given target.
func buildMcpServerEntry(target, token string) map[string]any {
	entry := map[string]any{
		"command": "npx",
		"args": []string{
			"-y",
			"mcp-remote",
			mcpServerURL,
			"--header",
			fmt.Sprintf("Authorization: Bearer %s", token),
		},
	}
	if target == "code" {
		entry["type"] = "stdio"
	}
	return entry
}

// readJSONFile reads a JSON file into a map. Returns an empty map if the file doesn't exist.
func readJSONFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]any{}, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	return result, nil
}
