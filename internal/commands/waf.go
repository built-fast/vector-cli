package commands

import (
	"github.com/spf13/cobra"
)

// NewWafCmd creates the waf command group.
func NewWafCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "waf",
		Short: "Manage WAF rules",
		Long:  "Manage Web Application Firewall rules for your sites.",
	}

	cmd.AddCommand(NewWafRateLimitCmd())

	return cmd
}
