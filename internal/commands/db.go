package commands

import (
	"github.com/spf13/cobra"
)

// NewDbCmd creates the db command group.
func NewDbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Manage database operations",
		Long:  "Manage database operations including import sessions and exports.",
	}

	cmd.AddCommand(NewDbImportSessionCmd())
	cmd.AddCommand(NewDbExportCmd())

	return cmd
}
