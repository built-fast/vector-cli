package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/skills"
)

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

	return cmd
}
