package commands

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func TestSkillPrintsEmbeddedContent(t *testing.T) {
	cmd, stdout := buildSkillCmd()
	cmd.SetArgs([]string{"skill"})

	err := cmd.Execute()
	require.NoError(t, err)

	expected, err := skills.Content.ReadFile("vector/SKILL.md")
	require.NoError(t, err)

	assert.Equal(t, string(expected), stdout.String())
}
