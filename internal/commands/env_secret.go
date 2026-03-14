package commands

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/output"
)

const secretsBasePath = "/api/v1/vector/secrets"

// NewEnvSecretCmd creates the env secret command group.
func NewEnvSecretCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secret",
		Short: "Manage environment secrets",
		Long:  "Manage secrets and environment variables for an environment.",
	}

	cmd.AddCommand(newEnvSecretListCmd())
	cmd.AddCommand(newEnvSecretShowCmd())
	cmd.AddCommand(newEnvSecretCreateCmd())
	cmd.AddCommand(newEnvSecretUpdateCmd())
	cmd.AddCommand(newEnvSecretDeleteCmd())

	return cmd
}

func newEnvSecretListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <env-id>",
		Short: "List secrets for an environment",
		Long:  "Retrieve a paginated list of secrets and environment variables for an environment.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			page, perPage := getPagination(cmd)
			query := buildPaginationQuery(page, perPage)

			path := envsBasePath + "/" + args[0] + "/secrets"
			resp, err := app.Client.Get(cmd.Context(), path, query)
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

			headers := []string{"ID", "KEY", "SECRET", "CREATED"}
			var rows [][]string
			for _, item := range items {
				rows = append(rows, []string{
					getString(item, "id"),
					getString(item, "key"),
					formatBool(getBool(item, "is_secret")),
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

func newEnvSecretShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <secret-id>",
		Short: "Show secret details",
		Long:  "Retrieve details of a specific secret or environment variable.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Get(cmd.Context(), secretsBasePath+"/"+args[0], nil)
			if err != nil {
				return fmt.Errorf("failed to show secret: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to show secret: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to show secret: %w", err)
			}

			if app.Format == output.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to show secret: %w", err)
			}

			pairs := []output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Key", Value: getString(item, "key")},
				{Key: "Secret", Value: formatBool(getBool(item, "is_secret"))},
				{Key: "Created", Value: getString(item, "created_at")},
				{Key: "Updated", Value: getString(item, "updated_at")},
			}

			// Show value only for non-secret env vars
			if !getBool(item, "is_secret") {
				pairs = append(pairs, output.KeyValue{Key: "Value", Value: formatString(getString(item, "value"))})
			}

			output.PrintKeyValue(cmd.OutOrStdout(), pairs)
			return nil
		},
	}
}

func newEnvSecretCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <env-id>",
		Short: "Create a secret",
		Long:  "Create a new secret or environment variable for an environment.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			key, _ := cmd.Flags().GetString("key")
			value, _ := cmd.Flags().GetString("value")

			reqBody := map[string]any{
				"key":   key,
				"value": value,
			}

			if cmd.Flags().Changed("is-secret") {
				v, _ := cmd.Flags().GetBool("is-secret")
				reqBody["is_secret"] = v
			}

			path := envsBasePath + "/" + args[0] + "/secrets"
			resp, err := app.Client.Post(cmd.Context(), path, reqBody)
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

			output.PrintKeyValue(cmd.OutOrStdout(), []output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Key", Value: getString(item, "key")},
				{Key: "Secret", Value: formatBool(getBool(item, "is_secret"))},
				{Key: "Created", Value: getString(item, "created_at")},
			})
			return nil
		},
	}

	cmd.Flags().String("key", "", "Secret key name (required)")
	cmd.Flags().String("value", "", "Secret value (required)")
	cmd.Flags().Bool("is-secret", true, "Whether the value is a secret (default: true)")
	_ = cmd.MarkFlagRequired("key")
	_ = cmd.MarkFlagRequired("value")

	return cmd
}

func newEnvSecretUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <secret-id>",
		Short: "Update a secret",
		Long:  "Update an existing secret or environment variable.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			reqBody := map[string]any{}

			if cmd.Flags().Changed("key") {
				v, _ := cmd.Flags().GetString("key")
				reqBody["key"] = v
			}
			if cmd.Flags().Changed("value") {
				v, _ := cmd.Flags().GetString("value")
				reqBody["value"] = v
			}
			if cmd.Flags().Changed("is-secret") {
				v, _ := cmd.Flags().GetBool("is-secret")
				reqBody["is_secret"] = v
			}

			resp, err := app.Client.Put(cmd.Context(), secretsBasePath+"/"+args[0], reqBody)
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

			output.PrintKeyValue(cmd.OutOrStdout(), []output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Key", Value: getString(item, "key")},
				{Key: "Secret", Value: formatBool(getBool(item, "is_secret"))},
				{Key: "Updated", Value: getString(item, "updated_at")},
			})
			return nil
		},
	}

	cmd.Flags().String("key", "", "New secret key name")
	cmd.Flags().String("value", "", "New secret value")
	cmd.Flags().Bool("is-secret", false, "Whether the value is a secret")

	return cmd
}

func newEnvSecretDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <secret-id>",
		Short: "Delete a secret",
		Long:  "Delete a secret or environment variable.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			force, _ := cmd.Flags().GetBool("force")
			if !force {
				if !confirmAction(cmd, fmt.Sprintf("Are you sure you want to delete secret %s?", args[0])) {
					output.PrintMessage(cmd.OutOrStdout(), "Aborted.")
					return nil
				}
			}

			resp, err := app.Client.Delete(cmd.Context(), secretsBasePath+"/"+args[0])
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

	cmd.Flags().Bool("force", false, "Skip confirmation prompt")

	return cmd
}
