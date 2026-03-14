package commands

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/output"
)

const restoresBasePath = "/api/v1/vector/restores"

// NewRestoreCmd creates the restore command group.
func NewRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Manage restores",
		Long:  "Manage restores to recover site data from backups.",
	}

	cmd.AddCommand(newRestoreListCmd())
	cmd.AddCommand(newRestoreShowCmd())
	cmd.AddCommand(newRestoreCreateCmd())

	return cmd
}

func newRestoreListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List restores",
		Long:  "Retrieve a paginated list of restores, optionally filtered by type, site, environment, or backup.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			page, perPage := getPagination(cmd)
			query := buildPaginationQuery(page, perPage)

			if cmd.Flags().Changed("site-id") {
				v, _ := cmd.Flags().GetString("site-id")
				if v != "" {
					query.Set("site_id", v)
				}
			}
			if cmd.Flags().Changed("environment-id") {
				v, _ := cmd.Flags().GetString("environment-id")
				if v != "" {
					query.Set("environment_id", v)
				}
			}
			if cmd.Flags().Changed("type") {
				v, _ := cmd.Flags().GetString("type")
				if v != "" {
					query.Set("type", v)
				}
			}
			if cmd.Flags().Changed("backup-id") {
				v, _ := cmd.Flags().GetString("backup-id")
				if v != "" {
					query.Set("backup_id", v)
				}
			}

			resp, err := app.Client.Get(cmd.Context(), restoresBasePath, query)
			if err != nil {
				return fmt.Errorf("failed to list restores: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to list restores: %w", err)
			}

			if app.Format == output.JSON {
				data, err := parseResponseData(body)
				if err != nil {
					return fmt.Errorf("failed to list restores: %w", err)
				}
				return output.PrintJSON(cmd.OutOrStdout(), json.RawMessage(data))
			}

			data, meta, err := parseResponseWithMeta(body)
			if err != nil {
				return fmt.Errorf("failed to list restores: %w", err)
			}

			var items []map[string]any
			if err := json.Unmarshal(data, &items); err != nil {
				return fmt.Errorf("failed to list restores: %w", err)
			}

			headers := []string{"ID", "MODEL", "BACKUP ID", "SCOPE", "STATUS", "CREATED"}
			var rows [][]string
			for _, item := range items {
				rows = append(rows, []string{
					getString(item, "id"),
					formatArchivableType(getString(item, "archivable_type")),
					getString(item, "vector_backup_id"),
					getString(item, "scope"),
					getString(item, "status"),
					getString(item, "created_at"),
				})
			}

			output.PrintTable(cmd.OutOrStdout(), headers, rows)
			printPaginationIfNeeded(cmd.OutOrStdout(), meta)
			return nil
		},
	}
	addPaginationFlags(cmd)
	cmd.Flags().String("site-id", "", "Filter by site ID")
	cmd.Flags().String("environment-id", "", "Filter by environment ID")
	cmd.Flags().String("type", "", "Filter by type (site/environment)")
	cmd.Flags().String("backup-id", "", "Filter by backup ID")
	return cmd
}

func newRestoreShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a restore",
		Long:  "Display details of a specific restore.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Get(cmd.Context(), restoresBasePath+"/"+args[0], nil)
			if err != nil {
				return fmt.Errorf("failed to get restore: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to get restore: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to get restore: %w", err)
			}

			if app.Format == output.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to get restore: %w", err)
			}

			output.PrintKeyValue(cmd.OutOrStdout(), []output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Model", Value: formatArchivableType(getString(item, "archivable_type"))},
				{Key: "Model ID", Value: getString(item, "archivable_id")},
				{Key: "Backup ID", Value: getString(item, "vector_backup_id")},
				{Key: "Scope", Value: getString(item, "scope")},
				{Key: "Trigger", Value: getString(item, "trigger")},
				{Key: "Status", Value: getString(item, "status")},
				{Key: "Error Message", Value: formatString(getString(item, "error_message"))},
				{Key: "Duration", Value: formatFloat(getFloat(item, "duration_ms"))},
				{Key: "Started At", Value: formatString(getString(item, "started_at"))},
				{Key: "Completed At", Value: formatString(getString(item, "completed_at"))},
				{Key: "Created At", Value: getString(item, "created_at")},
				{Key: "Updated At", Value: getString(item, "updated_at")},
			})
			return nil
		},
	}
}

func newRestoreCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <backup-id>",
		Short: "Create a restore",
		Long:  "Create a new restore from a backup.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			reqBody := map[string]any{
				"vector_backup_id": args[0],
			}

			dropTables, _ := cmd.Flags().GetBool("drop-tables")
			if dropTables {
				reqBody["drop_tables"] = true
			}

			disableForeignKeys, _ := cmd.Flags().GetBool("disable-foreign-keys")
			if disableForeignKeys {
				reqBody["disable_foreign_keys"] = true
			}

			srFrom, _ := cmd.Flags().GetString("search-replace-from")
			srTo, _ := cmd.Flags().GetString("search-replace-to")
			if srFrom != "" && srTo != "" {
				reqBody["search_replace"] = []map[string]string{
					{"from": srFrom, "to": srTo},
				}
			}

			resp, err := app.Client.Post(cmd.Context(), restoresBasePath, reqBody)
			if err != nil {
				return fmt.Errorf("failed to create restore: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to create restore: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to create restore: %w", err)
			}

			if app.Format == output.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to create restore: %w", err)
			}

			w := cmd.OutOrStdout()
			output.PrintMessage(w, fmt.Sprintf("Restore initiated. Use 'vector restore show %s' to check progress.", getString(item, "id")))
			output.PrintMessage(w, "")

			output.PrintKeyValue(w, []output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Model", Value: formatArchivableType(getString(item, "archivable_type"))},
				{Key: "Model ID", Value: getString(item, "archivable_id")},
				{Key: "Backup ID", Value: getString(item, "vector_backup_id")},
				{Key: "Scope", Value: getString(item, "scope")},
				{Key: "Trigger", Value: getString(item, "trigger")},
				{Key: "Status", Value: getString(item, "status")},
				{Key: "Error Message", Value: formatString(getString(item, "error_message"))},
				{Key: "Duration", Value: formatFloat(getFloat(item, "duration_ms"))},
				{Key: "Started At", Value: formatString(getString(item, "started_at"))},
				{Key: "Completed At", Value: formatString(getString(item, "completed_at"))},
				{Key: "Created At", Value: getString(item, "created_at")},
				{Key: "Updated At", Value: getString(item, "updated_at")},
			})
			return nil
		},
	}

	cmd.Flags().Bool("drop-tables", false, "Drop existing tables before restore")
	cmd.Flags().Bool("disable-foreign-keys", false, "Disable foreign key checks during restore")
	cmd.Flags().String("search-replace-from", "", "URL to search for (used with --search-replace-to)")
	cmd.Flags().String("search-replace-to", "", "URL to replace with (used with --search-replace-from)")

	return cmd
}
