package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/version"
)

// NewRootCmd creates and returns the root cobra command.
func NewRootCmd() *cobra.Command {
	var showVersion bool

	cmd := &cobra.Command{
		Use:   "vector",
		Short: "Vector CLI — manage your Vector.dev hosting",
		Long:  "Vector CLI — manage your Vector.dev hosting\n\nA command-line tool for managing sites, deployments, and configurations on Vector.dev.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Stub: will be populated in Milestone 1 for config/auth loading.
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				fmt.Fprintln(cmd.OutOrStdout(), version.FullVersion())
				return nil
			}
			return cmd.Help()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().BoolVar(&showVersion, "version", false, "Print version information and exit")
	cmd.PersistentFlags().Bool("json", false, "Force JSON output")
	cmd.PersistentFlags().Bool("no-json", false, "Force table output")

	return cmd
}
