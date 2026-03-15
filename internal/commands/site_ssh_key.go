package commands

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/output"
)

// NewSiteSSHKeyCmd creates the site ssh-key command group.
func NewSiteSSHKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh-key",
		Short: "Manage site SSH keys",
		Long:  "Manage SSH keys for a site's development container.",
	}

	cmd.AddCommand(newSSHKeyListCmd())
	cmd.AddCommand(newSSHKeyAddCmd())
	cmd.AddCommand(newSSHKeyRemoveCmd())

	return cmd
}

func newSSHKeyListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <site-id>",
		Short: "List SSH keys",
		Long: "Retrieve all SSH keys installed on a site's development container.",
		Example: `  # List SSH keys for a site
  vector site ssh-key list site-abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			page, perPage := getPagination(cmd)
			query := buildPaginationQuery(page, perPage)

			path := sitesBasePath + "/" + args[0] + "/ssh-keys"
			resp, err := app.Client.Get(cmd.Context(), path, query)
			if err != nil {
				return fmt.Errorf("failed to list SSH keys: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to list SSH keys: %w", err)
			}

			if app.Output.Format() == output.JSON {
				data, err := parseResponseData(body)
				if err != nil {
					return fmt.Errorf("failed to list SSH keys: %w", err)
				}
				return app.Output.JSON(json.RawMessage(data))
			}

			data, meta, err := parseResponseWithMeta(body)
			if err != nil {
				return fmt.Errorf("failed to list SSH keys: %w", err)
			}

			var items []map[string]any
			if err := json.Unmarshal(data, &items); err != nil {
				return fmt.Errorf("failed to list SSH keys: %w", err)
			}

			headers := []string{"ID", "NAME", "FINGERPRINT", "DEFAULT", "CREATED"}
			var rows [][]string
			for _, item := range items {
				rows = append(rows, []string{
					getString(item, "id"),
					getString(item, "name"),
					formatString(getString(item, "fingerprint")),
					formatBool(getBool(item, "is_account_default")),
					getString(item, "created_at"),
				})
			}

			app.Output.Table(headers, rows)
			if meta != nil {
				app.Output.Pagination(meta.CurrentPage, meta.LastPage, meta.Total)
			}
			return nil
		},
	}
	addPaginationFlags(cmd)
	return cmd
}

func newSSHKeyAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <site-id>",
		Short: "Add an SSH key",
		Long: "Add a new SSH key to a site's development container.",
		Example: `  # Add an SSH key to a site
  vector site ssh-key add site-abc123 --name "deploy-key" --public-key "ssh-ed25519 AAAA..."`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			name, _ := cmd.Flags().GetString("name")
			publicKey, _ := cmd.Flags().GetString("public-key")

			reqBody := map[string]any{
				"name":       name,
				"public_key": publicKey,
			}

			path := sitesBasePath + "/" + args[0] + "/ssh-keys"
			resp, err := app.Client.Post(cmd.Context(), path, reqBody)
			if err != nil {
				return fmt.Errorf("failed to add SSH key: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to add SSH key: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to add SSH key: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to add SSH key: %w", err)
			}

			app.Output.KeyValue([]output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Name", Value: getString(item, "name")},
				{Key: "Fingerprint", Value: formatString(getString(item, "fingerprint"))},
				{Key: "Default", Value: formatBool(getBool(item, "is_account_default"))},
			})
			return nil
		},
	}

	cmd.Flags().String("name", "", "Friendly name for the SSH key (required)")
	cmd.Flags().String("public-key", "", "SSH public key in OpenSSH format (required)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("public-key")

	return cmd
}

func newSSHKeyRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <site-id> <key-id>",
		Short: "Remove an SSH key",
		Long: "Remove an SSH key from a site's development container.",
		Example: `  # Remove an SSH key from a site
  vector site ssh-key remove site-abc123 key-456`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			path := sitesBasePath + "/" + args[0] + "/ssh-keys/" + args[1]
			resp, err := app.Client.Delete(cmd.Context(), path)
			if err != nil {
				return fmt.Errorf("failed to remove SSH key: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to remove SSH key: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to remove SSH key: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			app.Output.Message("SSH key removed successfully.")
			return nil
		},
	}
}
