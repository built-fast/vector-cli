package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/version"
	"github.com/built-fast/vector-cli/skills"
)

// skillInstallDir is the base directory for installed skills. Override in tests.
var skillInstallDir = ""

// claudeSkillsDir is the Claude Code skills directory. Override in tests.
var claudeSkillsDir = ""

// symlinkFunc is the function used to create symlinks. Override in tests.
var symlinkFunc = os.Symlink

// defaultSkillInstallDir returns ~/.agents/skills.
func defaultSkillInstallDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory: %w", err)
	}
	return filepath.Join(home, ".agents", "skills"), nil
}

// defaultClaudeSkillsDir returns ~/.claude/skills.
func defaultClaudeSkillsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory: %w", err)
	}
	return filepath.Join(home, ".claude", "skills"), nil
}

// getSkillInstallDir returns the skill install directory, using the override if set.
func getSkillInstallDir() (string, error) {
	if skillInstallDir != "" {
		return skillInstallDir, nil
	}
	return defaultSkillInstallDir()
}

// getClaudeSkillsDir returns the Claude skills directory, using the override if set.
func getClaudeSkillsDir() (string, error) {
	if claudeSkillsDir != "" {
		return claudeSkillsDir, nil
	}
	return defaultClaudeSkillsDir()
}

// NewSkillCmd creates the skill command group.
func NewSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Agent skill document",
		Long:  "View or manage the embedded SKILL.md agent reference document.",
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := skills.Content.ReadFile("vector/SKILL.md")
			if err != nil {
				return fmt.Errorf("failed to read embedded skill: %w", err)
			}
			_, err = cmd.OutOrStdout().Write(content)
			return err
		},
	}

	cmd.AddCommand(newSkillInstallCmd())
	cmd.AddCommand(newSkillUninstallCmd())

	return cmd
}

// newSkillInstallCmd creates the skill install leaf command.
func newSkillInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install skill document for AI agents",
		Long:  "Install the SKILL.md agent reference to ~/.agents/skills/vector/ and link it into ~/.claude/skills/vector/ for automatic discovery by Claude Code.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillInstall(cmd)
		},
	}
}

// installSkillFiles installs the skill document and version stamp to the given directory.
// Returns the path to the installed SKILL.md.
func installSkillFiles(installDir string) (string, error) {
	vectorDir := filepath.Join(installDir, "vector")
	if err := os.MkdirAll(vectorDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create skill directory: %w", err)
	}

	content, err := skills.Content.ReadFile("vector/SKILL.md")
	if err != nil {
		return "", fmt.Errorf("failed to read embedded skill: %w", err)
	}

	skillPath := filepath.Join(vectorDir, "SKILL.md")
	if err := os.WriteFile(skillPath, content, 0o644); err != nil {
		return "", fmt.Errorf("failed to write skill file: %w", err)
	}

	versionPath := filepath.Join(vectorDir, ".version")
	if err := os.WriteFile(versionPath, []byte(version.Version), 0o644); err != nil {
		return "", fmt.Errorf("failed to write version stamp: %w", err)
	}

	return skillPath, nil
}

// linkClaudeSkill creates a symlink (or copies as fallback) from the Claude skills
// directory to the installed skill file.
func linkClaudeSkill(claudeDir, installedPath string) error {
	claudeVectorDir := filepath.Join(claudeDir, "vector")
	if err := os.MkdirAll(claudeVectorDir, 0o755); err != nil {
		return fmt.Errorf("failed to create Claude skills directory: %w", err)
	}

	linkPath := filepath.Join(claudeVectorDir, "SKILL.md")

	// Remove existing file/symlink for idempotency.
	_ = os.Remove(linkPath)

	// Try symlink first.
	if err := symlinkFunc(installedPath, linkPath); err != nil {
		// Fallback to copy.
		content, readErr := os.ReadFile(installedPath)
		if readErr != nil {
			return fmt.Errorf("failed to read installed skill for copy: %w", readErr)
		}
		if writeErr := os.WriteFile(linkPath, content, 0o644); writeErr != nil {
			return fmt.Errorf("failed to copy skill file: %w", writeErr)
		}
	}

	return nil
}

// newSkillUninstallCmd creates the skill uninstall leaf command.
func newSkillUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall skill document",
		Long:  "Remove the installed SKILL.md agent reference from ~/.agents/skills/vector/ and ~/.claude/skills/vector/.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillUninstall(cmd)
		},
	}
}

// runSkillUninstall removes the installed skill files and Claude symlink/copy.
func runSkillUninstall(cmd *cobra.Command) error {
	installDir, err := getSkillInstallDir()
	if err != nil {
		return err
	}

	claudeDir, err := getClaudeSkillsDir()
	if err != nil {
		return err
	}

	// Remove ~/.agents/skills/vector/
	vectorInstallDir := filepath.Join(installDir, "vector")
	if err := os.RemoveAll(vectorInstallDir); err != nil {
		return fmt.Errorf("failed to remove skill directory: %w", err)
	}

	// Remove ~/.claude/skills/vector/
	claudeVectorDir := filepath.Join(claudeDir, "vector")
	if err := os.RemoveAll(claudeVectorDir); err != nil {
		return fmt.Errorf("failed to remove Claude skill directory: %w", err)
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Skill uninstalled successfully.")

	return nil
}

// runSkillInstall performs the full install sequence.
func runSkillInstall(cmd *cobra.Command) error {
	installDir, err := getSkillInstallDir()
	if err != nil {
		return err
	}

	installedPath, err := installSkillFiles(installDir)
	if err != nil {
		return err
	}

	claudeDir, err := getClaudeSkillsDir()
	if err != nil {
		return err
	}

	if err := linkClaudeSkill(claudeDir, installedPath); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Skill installed successfully.")
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Installed to: %s\n", filepath.Join(installDir, "vector", "SKILL.md"))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Linked from:  %s\n", filepath.Join(claudeDir, "vector", "SKILL.md"))

	return nil
}
