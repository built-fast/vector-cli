package commands

import (
	"github.com/spf13/cobra"
)

// NewDBCmd creates the db command group.
func NewDBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Manage database operations",
		Long:  "Manage database operations including import sessions and exports.",
	}

	cmd.AddCommand(NewDBImportSessionCmd())
	cmd.AddCommand(NewDBExportCmd())

	return cmd
}
