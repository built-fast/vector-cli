package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/output"
)

const apiKeysBasePath = "/api/v1/vector/api-keys"

// NewAccountAPIKeyCmd creates the account api-key command group.
func NewAccountAPIKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api-key",
		Short: "Manage account API keys",
		Long:  "Manage API keys for controlling programmatic access to your account.",
	}

	cmd.AddCommand(newAccountAPIKeyListCmd())
	cmd.AddCommand(newAccountAPIKeyCreateCmd())
	cmd.AddCommand(newAccountAPIKeyDeleteCmd())

	return cmd
}

func newAccountAPIKeyListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List API keys",
		Long:  "Retrieve a paginated list of API keys for your account.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			page, perPage := getPagination(cmd)
			query := buildPaginationQuery(page, perPage)

			resp, err := app.Client.Get(cmd.Context(), apiKeysBasePath, query)
			if err != nil {
				return fmt.Errorf("failed to list API keys: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to list API keys: %w", err)
			}

			if app.Format == output.JSON {
				data, err := parseResponseData(body)
				if err != nil {
					return fmt.Errorf("failed to list API keys: %w", err)
				}
				return output.PrintJSON(cmd.OutOrStdout(), json.RawMessage(data))
			}

			data, meta, err := parseResponseWithMeta(body)
			if err != nil {
				return fmt.Errorf("failed to list API keys: %w", err)
			}

			var items []map[string]any
			if err := json.Unmarshal(data, &items); err != nil {
				return fmt.Errorf("failed to list API keys: %w", err)
			}

			headers := []string{"ID", "NAME", "ABILITIES", "LAST USED", "EXPIRES"}
			var rows [][]string
			for _, item := range items {
				rows = append(rows, []string{
					getString(item, "id"),
					getString(item, "name"),
					formatAbilities(item),
					formatString(getString(item, "last_used_at")),
					formatString(getString(item, "expires_at")),
				})
			}

			output.PrintTable(cmd.OutOrStdout(), headers, rows)
			printPaginationIfNeeded(cmd.OutOrStdout(), meta)
			return nil
		},
	}
	addPaginationFlags(cmd)
	return cmd
}

func newAccountAPIKeyCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an API key",
		Long:  "Create a new API key for programmatic access to your account.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			name, _ := cmd.Flags().GetString("name")

			reqBody := map[string]any{
				"name": name,
			}

			if cmd.Flags().Changed("abilities") {
				abilitiesStr, _ := cmd.Flags().GetString("abilities")
				abilities := strings.Split(abilitiesStr, ",")
				reqBody["abilities"] = abilities
			}

			if cmd.Flags().Changed("expires-at") {
				expiresAt, _ := cmd.Flags().GetString("expires-at")
				reqBody["expires_at"] = expiresAt
			}

			resp, err := app.Client.Post(cmd.Context(), apiKeysBasePath, reqBody)
			if err != nil {
				return fmt.Errorf("failed to create API key: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to create API key: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to create API key: %w", err)
			}

			if app.Format == output.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to create API key: %w", err)
			}

			w := cmd.OutOrStdout()
			output.PrintKeyValue(w, []output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Name", Value: getString(item, "name")},
				{Key: "Token", Value: getString(item, "token")},
				{Key: "Abilities", Value: formatAbilities(item)},
				{Key: "Expires", Value: formatString(getString(item, "expires_at"))},
				{Key: "Created", Value: getString(item, "created_at")},
			})
			output.PrintMessage(w, "")
			output.PrintMessage(w, "Save this token — it won't be shown again!")
			return nil
		},
	}

	cmd.Flags().String("name", "", "Name for the API key (required)")
	cmd.Flags().String("abilities", "", "Comma-separated abilities (e.g., \"site:read,site:write\")")
	cmd.Flags().String("expires-at", "", "ISO datetime for token expiration")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func newAccountAPIKeyDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <key-id>",
		Short: "Delete an API key",
		Long:  "Delete an API key. You cannot delete the token currently being used for authentication.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Delete(cmd.Context(), apiKeysBasePath+"/"+args[0])
			if err != nil {
				return fmt.Errorf("failed to delete API key: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to delete API key: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to delete API key: %w", err)
			}

			if app.Format == output.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), json.RawMessage(data))
			}

			output.PrintMessage(cmd.OutOrStdout(), "API key deleted successfully.")
			return nil
		},
	}
}

// formatAbilities joins the abilities array into a comma-separated string.
func formatAbilities(item map[string]any) string {
	abilities := getSlice(item, "abilities")
	if len(abilities) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(abilities))
	for _, a := range abilities {
		if s, ok := a.(string); ok {
			parts = append(parts, s)
		}
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ", ")
}
