package commands

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/built-fast/vector-cli/internal/version"
	"github.com/built-fast/vector-cli/skills"
)

func buildSkillCmd() (*cobra.Command, *bytes.Buffer) {
	root := &cobra.Command{
		Use:           "vector",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(NewSkillCmd())

	stdout := new(bytes.Buffer)
	root.SetOut(stdout)

	return root, stdout
}

func setSkillTestDirs(t *testing.T) (installDir, claudeDir string) {
	t.Helper()
	installDir = filepath.Join(t.TempDir(), "agents", "skills")
	claudeDir = filepath.Join(t.TempDir(), "claude", "skills")

	oldInstall := skillInstallDir
	oldClaude := claudeSkillsDir
	skillInstallDir = installDir
	claudeSkillsDir = claudeDir
	t.Cleanup(func() {
		skillInstallDir = oldInstall
		claudeSkillsDir = oldClaude
	})
	return installDir, claudeDir
}

func TestSkillPrintsEmbeddedContent(t *testing.T) {
	cmd, stdout := buildSkillCmd()
	cmd.SetArgs([]string{"skill"})

	err := cmd.Execute()
	require.NoError(t, err)

	expected, err := skills.Content.ReadFile("vector/SKILL.md")
	require.NoError(t, err)

	assert.Equal(t, string(expected), stdout.String())
}

func TestSkillInstallCreatesFile(t *testing.T) {
	installDir, claudeDir := setSkillTestDirs(t)

	cmd, stdout := buildSkillCmd()
	cmd.SetArgs([]string{"skill", "install"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify SKILL.md was installed.
	installed, err := os.ReadFile(filepath.Join(installDir, "vector", "SKILL.md"))
	require.NoError(t, err)

	expected, err := skills.Content.ReadFile("vector/SKILL.md")
	require.NoError(t, err)
	assert.Equal(t, string(expected), string(installed))

	// Verify Claude skills link/copy exists and is readable.
	linked, err := os.ReadFile(filepath.Join(claudeDir, "vector", "SKILL.md"))
	require.NoError(t, err)
	assert.Equal(t, string(expected), string(linked))

	// Verify output message.
	assert.Contains(t, stdout.String(), "Skill installed successfully.")
}

func TestSkillInstallIdempotent(t *testing.T) {
	installDir, _ := setSkillTestDirs(t)

	cmd1, _ := buildSkillCmd()
	cmd1.SetArgs([]string{"skill", "install"})
	require.NoError(t, cmd1.Execute())

	cmd2, stdout := buildSkillCmd()
	cmd2.SetArgs([]string{"skill", "install"})
	require.NoError(t, cmd2.Execute())

	// File still exists and is correct after second run.
	installed, err := os.ReadFile(filepath.Join(installDir, "vector", "SKILL.md"))
	require.NoError(t, err)

	expected, err := skills.Content.ReadFile("vector/SKILL.md")
	require.NoError(t, err)
	assert.Equal(t, string(expected), string(installed))
	assert.Contains(t, stdout.String(), "Skill installed successfully.")
}

func TestSkillInstallSymlink(t *testing.T) {
	installDir, claudeDir := setSkillTestDirs(t)

	cmd, _ := buildSkillCmd()
	cmd.SetArgs([]string{"skill", "install"})
	require.NoError(t, cmd.Execute())

	linkPath := filepath.Join(claudeDir, "vector", "SKILL.md")
	target, err := os.Readlink(linkPath)
	require.NoError(t, err, "expected a symlink at %s", linkPath)
	assert.Equal(t, filepath.Join(installDir, "vector", "SKILL.md"), target)
}

func TestSkillInstallCopyFallback(t *testing.T) {
	installDir, claudeDir := setSkillTestDirs(t)

	// Override symlinkFunc to always fail, forcing the copy fallback.
	oldSymlink := symlinkFunc
	symlinkFunc = func(_, _ string) error {
		return errors.New("symlink not supported")
	}
	t.Cleanup(func() { symlinkFunc = oldSymlink })

	cmd, stdout := buildSkillCmd()
	cmd.SetArgs([]string{"skill", "install"})
	require.NoError(t, cmd.Execute())

	// Verify the Claude skills file is a regular file (not a symlink).
	linkPath := filepath.Join(claudeDir, "vector", "SKILL.md")
	_, err := os.Readlink(linkPath)
	assert.Error(t, err, "expected a regular file, not a symlink")

	// Verify content matches.
	expected, err := skills.Content.ReadFile("vector/SKILL.md")
	require.NoError(t, err)

	copied, err := os.ReadFile(linkPath)
	require.NoError(t, err)
	assert.Equal(t, string(expected), string(copied))

	// Verify installed file also exists.
	installed, err := os.ReadFile(filepath.Join(installDir, "vector", "SKILL.md"))
	require.NoError(t, err)
	assert.Equal(t, string(expected), string(installed))

	assert.Contains(t, stdout.String(), "Skill installed successfully.")
}

func TestSkillUninstallRemovesFiles(t *testing.T) {
	installDir, claudeDir := setSkillTestDirs(t)

	// Install first.
	cmd1, _ := buildSkillCmd()
	cmd1.SetArgs([]string{"skill", "install"})
	require.NoError(t, cmd1.Execute())

	// Verify files exist.
	require.FileExists(t, filepath.Join(installDir, "vector", "SKILL.md"))
	require.FileExists(t, filepath.Join(claudeDir, "vector", "SKILL.md"))

	// Uninstall.
	cmd2, stdout := buildSkillCmd()
	cmd2.SetArgs([]string{"skill", "uninstall"})
	require.NoError(t, cmd2.Execute())

	// Verify files are removed.
	assert.NoDirExists(t, filepath.Join(installDir, "vector"))
	assert.NoDirExists(t, filepath.Join(claudeDir, "vector"))
	assert.Contains(t, stdout.String(), "Skill uninstalled successfully.")
}

func TestSkillUninstallNoOpWhenNotInstalled(t *testing.T) {
	setSkillTestDirs(t)

	// Uninstall without installing first — should be a no-op.
	cmd, stdout := buildSkillCmd()
	cmd.SetArgs([]string{"skill", "uninstall"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Skill uninstalled successfully.")
}

func TestSkillInstallVersionStamp(t *testing.T) {
	installDir, _ := setSkillTestDirs(t)

	cmd, _ := buildSkillCmd()
	cmd.SetArgs([]string{"skill", "install"})
	require.NoError(t, cmd.Execute())

	stamp, err := os.ReadFile(filepath.Join(installDir, "vector", ".version"))
	require.NoError(t, err)
	assert.Equal(t, version.Version, string(stamp))
}
