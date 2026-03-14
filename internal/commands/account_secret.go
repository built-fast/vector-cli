package commands

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/output"
)

const globalSecretsBasePath = "/api/v1/vector/global-secrets"

// NewAccountSecretCmd creates the account secret command group.
func NewAccountSecretCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secret",
		Short: "Manage account secrets",
		Long:  "Manage account-level secrets and environment variables shared across sites.",
	}

	cmd.AddCommand(newAccountSecretListCmd())
	cmd.AddCommand(newAccountSecretShowCmd())
	cmd.AddCommand(newAccountSecretCreateCmd())
	cmd.AddCommand(newAccountSecretUpdateCmd())
	cmd.AddCommand(newAccountSecretDeleteCmd())

	return cmd
}

func newAccountSecretListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List global secrets",
		Long:  "Retrieve a paginated list of account-level secrets and environment variables.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			page, perPage := getPagination(cmd)
			query := buildPaginationQuery(page, perPage)

			resp, err := app.Client.Get(cmd.Context(), globalSecretsBasePath, query)
			if err != nil {
				return fmt.Errorf("failed to list secrets: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to list secrets: %w", err)
			}

			if app.Format == output.JSON {
				data, err := parseResponseData(body)
				if err != nil {
					return fmt.Errorf("failed to list secrets: %w", err)
				}
				return output.PrintJSON(cmd.OutOrStdout(), json.RawMessage(data))
			}

			data, meta, err := parseResponseWithMeta(body)
			if err != nil {
				return fmt.Errorf("failed to list secrets: %w", err)
			}

			var items []map[string]any
			if err := json.Unmarshal(data, &items); err != nil {
				return fmt.Errorf("failed to list secrets: %w", err)
			}

			headers := []string{"ID", "KEY", "SECRET", "VALUE", "CREATED"}
			var rows [][]string
			for _, item := range items {
				isSecret := getBool(item, "is_secret")
				value := getString(item, "value")
				if isSecret {
					value = "-"
				}
				rows = append(rows, []string{
					getString(item, "id"),
					getString(item, "key"),
					formatBool(isSecret),
					formatString(value),
					getString(item, "created_at"),
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

func newAccountSecretShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a secret",
		Long:  "Display details of an account-level secret or environment variable.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Get(cmd.Context(), globalSecretsBasePath+"/"+args[0], nil)
			if err != nil {
				return fmt.Errorf("failed to get secret: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to get secret: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to get secret: %w", err)
			}

			if app.Format == output.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to get secret: %w", err)
			}

			isSecret := getBool(item, "is_secret")
			value := getString(item, "value")
			if isSecret {
				value = "-"
			}

			output.PrintKeyValue(cmd.OutOrStdout(), []output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Key", Value: getString(item, "key")},
				{Key: "Secret", Value: formatBool(isSecret)},
				{Key: "Value", Value: formatString(value)},
				{Key: "Created", Value: getString(item, "created_at")},
				{Key: "Updated", Value: getString(item, "updated_at")},
			})
			return nil
		},
	}
}

func newAccountSecretCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a secret",
		Long:  "Create a new account-level secret or environment variable.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			key, _ := cmd.Flags().GetString("key")
			value, _ := cmd.Flags().GetString("value")
			noSecret, _ := cmd.Flags().GetBool("no-secret")

			reqBody := map[string]any{
				"key":       key,
				"value":     value,
				"is_secret": !noSecret,
			}

			resp, err := app.Client.Post(cmd.Context(), globalSecretsBasePath, reqBody)
			if err != nil {
				return fmt.Errorf("failed to create secret: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to create secret: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to create secret: %w", err)
			}

			if app.Format == output.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to create secret: %w", err)
			}

			isSecret := getBool(item, "is_secret")
			displayValue := getString(item, "value")
			if isSecret {
				displayValue = "-"
			}

			output.PrintKeyValue(cmd.OutOrStdout(), []output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Key", Value: getString(item, "key")},
				{Key: "Secret", Value: formatBool(isSecret)},
				{Key: "Value", Value: formatString(displayValue)},
				{Key: "Created", Value: getString(item, "created_at")},
			})
			return nil
		},
	}

	cmd.Flags().String("key", "", "Key name for the secret (required)")
	cmd.Flags().String("value", "", "Value for the secret (required)")
	cmd.Flags().Bool("no-secret", false, "Store as plain environment variable (not secret)")
	_ = cmd.MarkFlagRequired("key")
	_ = cmd.MarkFlagRequired("value")

	return cmd
}

func newAccountSecretUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a secret",
		Long:  "Update an account-level secret or environment variable.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			reqBody := map[string]any{}

			if cmd.Flags().Changed("value") {
				value, _ := cmd.Flags().GetString("value")
				reqBody["value"] = value
			}

			if cmd.Flags().Changed("no-secret") {
				noSecret, _ := cmd.Flags().GetBool("no-secret")
				reqBody["is_secret"] = !noSecret
			}

			resp, err := app.Client.Put(cmd.Context(), globalSecretsBasePath+"/"+args[0], reqBody)
			if err != nil {
				return fmt.Errorf("failed to update secret: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to update secret: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to update secret: %w", err)
			}

			if app.Format == output.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to update secret: %w", err)
			}

			isSecret := getBool(item, "is_secret")
			displayValue := getString(item, "value")
			if isSecret {
				displayValue = "-"
			}

			output.PrintKeyValue(cmd.OutOrStdout(), []output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Key", Value: getString(item, "key")},
				{Key: "Secret", Value: formatBool(isSecret)},
				{Key: "Value", Value: formatString(displayValue)},
				{Key: "Created", Value: getString(item, "created_at")},
				{Key: "Updated", Value: getString(item, "updated_at")},
			})
			return nil
		},
	}

	cmd.Flags().String("value", "", "New value for the secret")
	cmd.Flags().Bool("no-secret", false, "Store as plain environment variable (not secret)")

	return cmd
}

func newAccountSecretDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a secret",
		Long:  "Delete an account-level secret or environment variable.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Delete(cmd.Context(), globalSecretsBasePath+"/"+args[0])
			if err != nil {
				return fmt.Errorf("failed to delete secret: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to delete secret: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to delete secret: %w", err)
			}

			if app.Format == output.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), json.RawMessage(data))
			}

			output.PrintMessage(cmd.OutOrStdout(), "Secret deleted successfully.")
			return nil
		},
	}
}
