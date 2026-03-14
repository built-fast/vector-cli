package commands

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/output"
)

func wafBlockedIPsPath(siteID string) string {
	return sitesBasePath + "/" + siteID + "/waf/blocked-ips"
}

// NewWafBlockedIPCmd creates the waf blocked-ip command group.
func NewWafBlockedIPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blocked-ip",
		Short: "Manage WAF blocked IPs",
		Long:  "Manage blocked IP addresses for a site's Web Application Firewall.",
	}

	cmd.AddCommand(newWafBlockedIPListCmd())
	cmd.AddCommand(newWafBlockedIPAddCmd())
	cmd.AddCommand(newWafBlockedIPRemoveCmd())

	return cmd
}

func newWafBlockedIPListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <site-id>",
		Short: "List blocked IPs",
		Long:  "List all blocked IP addresses for a site.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Get(cmd.Context(), wafBlockedIPsPath(args[0]), nil)
			if err != nil {
				return fmt.Errorf("failed to list blocked IPs: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to list blocked IPs: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to list blocked IPs: %w", err)
			}

			if app.Format == output.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), json.RawMessage(data))
			}

			var items []map[string]any
			if err := json.Unmarshal(data, &items); err != nil {
				return fmt.Errorf("failed to list blocked IPs: %w", err)
			}

			headers := []string{"IP"}
			var rows [][]string
			for _, item := range items {
				rows = append(rows, []string{
					getString(item, "ip"),
				})
			}

			output.PrintTable(cmd.OutOrStdout(), headers, rows)
			return nil
		},
	}
}

func newWafBlockedIPAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <site-id> <ip>",
		Short: "Add a blocked IP",
		Long:  "Add an IP address to the blocklist for a site.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			reqBody := map[string]any{
				"ip": args[1],
			}

			resp, err := app.Client.Post(cmd.Context(), wafBlockedIPsPath(args[0]), reqBody)
			if err != nil {
				return fmt.Errorf("failed to add blocked IP: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to add blocked IP: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to add blocked IP: %w", err)
			}

			if app.Format == output.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), json.RawMessage(data))
			}

			output.PrintMessage(cmd.OutOrStdout(), fmt.Sprintf("IP %s added to blocklist.", args[1]))
			return nil
		},
	}
}

func newWafBlockedIPRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <site-id> <ip>",
		Short: "Remove a blocked IP",
		Long:  "Remove an IP address from the blocklist for a site.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Delete(cmd.Context(), wafBlockedIPsPath(args[0])+"/"+args[1])
			if err != nil {
				return fmt.Errorf("failed to remove blocked IP: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to remove blocked IP: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to remove blocked IP: %w", err)
			}

			if app.Format == output.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), json.RawMessage(data))
			}

			output.PrintMessage(cmd.OutOrStdout(), fmt.Sprintf("IP %s removed from blocklist.", args[1]))
			return nil
		},
	}
}
