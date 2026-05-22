package commands

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/output"
)

const sshKeysBasePath = "/api/v1/vector/ssh-keys"

// NewAccountSSHKeyCmd creates the account ssh-key command group.
func NewAccountSSHKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh-key",
		Short: "Manage account SSH keys",
		Long:  "Manage account-level SSH keys for controlling SSH access.",
	}

	cmd.AddCommand(newAccountSSHKeyListCmd())
	cmd.AddCommand(newAccountSSHKeyShowCmd())
	cmd.AddCommand(newAccountSSHKeyCreateCmd())
	cmd.AddCommand(newAccountSSHKeyDeleteCmd())

	return cmd
}

func newAccountSSHKeyListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List account SSH keys",
		Long:  "Retrieve a paginated list of account-level SSH keys.",
		Example: `  # List account SSH keys
  vector account ssh-key list`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			page, perPage := getPagination(cmd)
			query := buildPaginationQuery(page, perPage)

			resp, err := app.Client.Get(cmd.Context(), sshKeysBasePath, query)
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

			headers := []string{"ID", "NAME", "FINGERPRINT", "CREATED"}
			var rows [][]string
			for _, item := range items {
				rows = append(rows, []string{
					getString(item, "id"),
					getString(item, "name"),
					formatString(getString(item, "fingerprint")),
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

func newAccountSSHKeyShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <key-id>",
		Short: "Show SSH key details",
		Long:  "Retrieve details of a specific account-level SSH key.",
		Example: `  # Show SSH key details
  vector account ssh-key show key-456`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Get(cmd.Context(), sshKeysBasePath+"/"+args[0], nil)
			if err != nil {
				return fmt.Errorf("failed to show SSH key: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to show SSH key: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to show SSH key: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to show SSH key: %w", err)
			}

			app.Output.KeyValue([]output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Name", Value: getString(item, "name")},
				{Key: "Fingerprint", Value: formatString(getString(item, "fingerprint"))},
				{Key: "Public Key Preview", Value: formatString(getString(item, "public_key_preview"))},
				{Key: "Account Default", Value: formatBool(getBool(item, "is_account_default"))},
				{Key: "Created", Value: getString(item, "created_at")},
			})
			return nil
		},
	}
}

func newAccountSSHKeyCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an SSH key",
		Long:  "Create a new account-level SSH key.",
		Example: `  # Create an SSH key
  vector account ssh-key create --name "deploy-key" --public-key "ssh-ed25519 AAAA..."`,
		Args: cobra.NoArgs,
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

			resp, err := app.Client.Post(cmd.Context(), sshKeysBasePath, reqBody)
			if err != nil {
				return fmt.Errorf("failed to create SSH key: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to create SSH key: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to create SSH key: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to create SSH key: %w", err)
			}

			app.Output.KeyValue([]output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Name", Value: getString(item, "name")},
				{Key: "Fingerprint", Value: formatString(getString(item, "fingerprint"))},
				{Key: "Public Key Preview", Value: formatString(getString(item, "public_key_preview"))},
				{Key: "Account Default", Value: formatBool(getBool(item, "is_account_default"))},
				{Key: "Created", Value: getString(item, "created_at")},
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

func newAccountSSHKeyDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <key-id>",
		Short: "Delete an SSH key",
		Long:  "Delete an account-level SSH key.",
		Example: `  # Delete an SSH key
  vector account ssh-key delete key-456`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Delete(cmd.Context(), sshKeysBasePath+"/"+args[0])
			if err != nil {
				return fmt.Errorf("failed to delete SSH key: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to delete SSH key: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to delete SSH key: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			output.PrintMessage(cmd.OutOrStdout(), "SSH key deleted successfully.")
			return nil
		},
	}
}
