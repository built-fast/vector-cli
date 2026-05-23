package commands

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/output"
)

func wafBlockedReferrersPath(siteID string) string {
	return sitesBasePath + "/" + siteID + "/waf/blocked-referrers"
}

// NewWafBlockedReferrerCmd creates the waf blocked-referrer command group.
func NewWafBlockedReferrerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blocked-referrer",
		Short: "Manage WAF blocked referrers",
		Long:  "Manage blocked referrer hostnames for a site's Web Application Firewall.",
	}

	cmd.AddCommand(newWafBlockedReferrerListCmd())
	cmd.AddCommand(newWafBlockedReferrerAddCmd())
	cmd.AddCommand(newWafBlockedReferrerRemoveCmd())

	return cmd
}

func newWafBlockedReferrerListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <site-id>",
		Short: "List blocked referrers",
		Long:  "List all blocked referrer hostnames for a site.",
		Example: `  # List blocked referrers
  vector waf blocked-referrer list site-abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Get(cmd.Context(), wafBlockedReferrersPath(args[0]), nil)
			if err != nil {
				return fmt.Errorf("failed to list blocked referrers: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to list blocked referrers: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to list blocked referrers: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(data)
			}

			var items []map[string]any
			if err := json.Unmarshal(data, &items); err != nil {
				return fmt.Errorf("failed to list blocked referrers: %w", err)
			}

			headers := []string{"HOSTNAME"}
			var rows [][]string
			for _, item := range items {
				rows = append(rows, []string{
					getString(item, "hostname"),
				})
			}

			app.Output.Table(headers, rows)
			return nil
		},
	}
}

func newWafBlockedReferrerAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <site-id> <hostname>",
		Short: "Add a blocked referrer",
		Long:  "Add a hostname to the blocked referrers list for a site.",
		Example: `  # Block a referrer
  vector waf blocked-referrer add site-abc123 spam.example.com`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			reqBody := map[string]any{
				"hostname": args[1],
			}

			resp, err := app.Client.Post(cmd.Context(), wafBlockedReferrersPath(args[0]), reqBody)
			if err != nil {
				return fmt.Errorf("failed to add blocked referrer: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to add blocked referrer: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to add blocked referrer: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(data)
			}

			app.Output.Message(fmt.Sprintf("Hostname %s added to blocked referrers.", args[1]))
			return nil
		},
	}
}

func newWafBlockedReferrerRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <site-id> <hostname>",
		Short: "Remove a blocked referrer",
		Long:  "Remove a hostname from the blocked referrers list for a site.",
		Example: `  # Unblock a referrer
  vector waf blocked-referrer remove site-abc123 spam.example.com`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Delete(cmd.Context(), wafBlockedReferrersPath(args[0])+"/"+args[1])
			if err != nil {
				return fmt.Errorf("failed to remove blocked referrer: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to remove blocked referrer: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to remove blocked referrer: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(data)
			}

			app.Output.Message(fmt.Sprintf("Hostname %s removed from blocked referrers.", args[1]))
			return nil
		},
	}
}
