package cli

import (
	"github.com/spf13/cobra"
)

// NewRootCmd creates and returns the root cobra command.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vector",
		Short: "Vector CLI — manage your Vector.dev hosting",
	}

	return cmd
}
