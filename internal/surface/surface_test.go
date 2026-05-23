package surface

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	root := &cobra.Command{Use: "vector"}
	root.PersistentFlags().String("token", "", "API token")
	root.PersistentFlags().Bool("json", false, "JSON output")

	// Group command (no RunE).
	site := &cobra.Command{Use: "site", Short: "Manage sites"}
	root.AddCommand(site)

	// Leaf with positional args and local flags.
	list := &cobra.Command{
		Use:  "list",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	list.Flags().Int("page", 1, "Page number")
	site.AddCommand(list)

	show := &cobra.Command{
		Use:  "show <site-id>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	site.AddCommand(show)

	// Command with multiple positional args.
	clone := &cobra.Command{
		Use:  "clone <source-id> <name>",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	clone.Flags().String("php-version", "", "PHP version")
	site.AddCommand(clone)

	// Auth group.
	auth := &cobra.Command{Use: "auth", Short: "Authentication"}
	root.AddCommand(auth)

	login := &cobra.Command{
		Use:  "login",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	auth.AddCommand(login)

	// Built-in help command should be excluded.
	// Cobra adds help automatically; we just verify it's excluded.

	got := Generate(root)

	expected := `ARG vector site clone 0 source-id
ARG vector site clone 1 name
ARG vector site show 0 site-id
CMD vector
CMD vector auth
CMD vector auth login
CMD vector site
CMD vector site clone
CMD vector site list
CMD vector site show
FLAG vector --json type=bool
FLAG vector --token type=string
FLAG vector site clone --php-version type=string
FLAG vector site list --page type=int
`

	assert.Equal(t, expected, got)
}

func TestGenerateExcludesCompletion(t *testing.T) {
	root := &cobra.Command{Use: "vector"}

	// Add a completion command (Cobra adds one by default in some setups).
	completion := &cobra.Command{Use: "completion", Short: "Generate completions"}
	root.AddCommand(completion)

	realCmd := &cobra.Command{
		Use:  "status",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	root.AddCommand(realCmd)

	got := Generate(root)

	assert.Contains(t, got, "CMD vector\n")
	assert.Contains(t, got, "CMD vector status\n")
	assert.NotContains(t, got, "completion")
}

func TestGenerateExcludesHelpFlag(t *testing.T) {
	root := &cobra.Command{Use: "vector"}
	root.Flags().String("version", "", "Show version")

	got := Generate(root)

	assert.NotContains(t, got, "--help")
	assert.Contains(t, got, "--version")
}
