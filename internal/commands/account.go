package commands

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/output"
)

const accountBasePath = "/api/v1/vector/account"

// NewAccountCmd creates the account command group.
func NewAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "Manage account",
		Long:  "View account details and manage account-level resources.",
	}

	cmd.AddCommand(newAccountShowCmd())

	return cmd
}

func newAccountShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show account summary",
		Long:  "Display account details including owner information and resource usage.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Get(cmd.Context(), accountBasePath, nil)
			if err != nil {
				return fmt.Errorf("failed to get account: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to get account: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to get account: %w", err)
			}

			if app.Format == output.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to get account: %w", err)
			}

			owner := getMap(item, "owner")
			account := getMap(item, "account")
			sites := getMap(item, "sites")
			envs := getMap(item, "environments")

			output.PrintKeyValue(cmd.OutOrStdout(), []output.KeyValue{
				{Key: "Owner Name", Value: getString(owner, "name")},
				{Key: "Owner Email", Value: getString(owner, "email")},
				{Key: "Account Name", Value: getString(account, "name")},
				{Key: "Company", Value: getString(account, "company")},
				{Key: "Total Sites", Value: fmt.Sprintf("%.0f", getFloat(sites, "total"))},
				{Key: "Active Sites", Value: fmt.Sprintf("%.0f", getFloat(getMap(sites, "by_status"), "active"))},
				{Key: "Total Environments", Value: fmt.Sprintf("%.0f", getFloat(envs, "total"))},
				{Key: "Active Environments", Value: fmt.Sprintf("%.0f", getFloat(getMap(envs, "by_status"), "active"))},
			})
			return nil
		},
	}
}
