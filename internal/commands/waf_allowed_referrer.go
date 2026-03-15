package commands

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/output"
)

func wafAllowedReferrersPath(siteID string) string {
	return sitesBasePath + "/" + siteID + "/waf/allowed-referrers"
}

// NewWafAllowedReferrerCmd creates the waf allowed-referrer command group.
func NewWafAllowedReferrerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "allowed-referrer",
		Short: "Manage WAF allowed referrers",
		Long:  "Manage allowed referrer hostnames for a site's Web Application Firewall.",
	}

	cmd.AddCommand(newWafAllowedReferrerListCmd())
	cmd.AddCommand(newWafAllowedReferrerAddCmd())
	cmd.AddCommand(newWafAllowedReferrerRemoveCmd())

	return cmd
}

func newWafAllowedReferrerListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <site-id>",
		Short: "List allowed referrers",
		Long: "List all allowed referrer hostnames for a site.",
		Example: `  # List allowed referrers
  vector waf allowed-referrer list site-abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Get(cmd.Context(), wafAllowedReferrersPath(args[0]), nil)
			if err != nil {
				return fmt.Errorf("failed to list allowed referrers: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to list allowed referrers: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to list allowed referrers: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var items []map[string]any
			if err := json.Unmarshal(data, &items); err != nil {
				return fmt.Errorf("failed to list allowed referrers: %w", err)
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

func newWafAllowedReferrerAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <site-id> <hostname>",
		Short: "Add an allowed referrer",
		Long: "Add a hostname to the allowed referrers list for a site.",
		Example: `  # Allow a referrer
  vector waf allowed-referrer add site-abc123 trusted.example.com`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			reqBody := map[string]any{
				"hostname": args[1],
			}

			resp, err := app.Client.Post(cmd.Context(), wafAllowedReferrersPath(args[0]), reqBody)
			if err != nil {
				return fmt.Errorf("failed to add allowed referrer: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to add allowed referrer: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to add allowed referrer: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			app.Output.Message(fmt.Sprintf("Hostname %s added to allowed referrers.", args[1]))
			return nil
		},
	}
}

func newWafAllowedReferrerRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <site-id> <hostname>",
		Short: "Remove an allowed referrer",
		Long: "Remove a hostname from the allowed referrers list for a site.",
		Example: `  # Remove an allowed referrer
  vector waf allowed-referrer remove site-abc123 trusted.example.com`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Delete(cmd.Context(), wafAllowedReferrersPath(args[0])+"/"+args[1])
			if err != nil {
				return fmt.Errorf("failed to remove allowed referrer: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to remove allowed referrer: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to remove allowed referrer: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			app.Output.Message(fmt.Sprintf("Hostname %s removed from allowed referrers.", args[1]))
			return nil
		},
	}
}
