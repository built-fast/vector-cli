package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/output"
)

const backupsBasePath = "/api/v1/vector/backups"

// NewBackupCmd creates the backup command group.
func NewBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Manage backups",
		Long:  "Manage backups to protect your site data.",
	}

	cmd.AddCommand(newBackupListCmd())
	cmd.AddCommand(newBackupShowCmd())
	cmd.AddCommand(newBackupCreateCmd())
	cmd.AddCommand(NewBackupDownloadCmd())

	return cmd
}

// formatArchivableType formats the archivable_type for display.
// e.g., "vector_site" becomes "Site", "vector_environment" becomes "Environment".
func formatArchivableType(raw string) string {
	raw = strings.TrimPrefix(raw, "vector_")
	if raw == "" {
		return "-"
	}
	// Capitalize first letter
	return strings.ToUpper(raw[:1]) + raw[1:]
}

func newBackupListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List backups",
		Long:  "Retrieve a paginated list of backups, optionally filtered by type, site, or environment.",
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

			resp, err := app.Client.Get(cmd.Context(), backupsBasePath, query)
			if err != nil {
				return fmt.Errorf("failed to list backups: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to list backups: %w", err)
			}

			if app.Format == output.JSON {
				data, err := parseResponseData(body)
				if err != nil {
					return fmt.Errorf("failed to list backups: %w", err)
				}
				return output.PrintJSON(cmd.OutOrStdout(), json.RawMessage(data))
			}

			data, meta, err := parseResponseWithMeta(body)
			if err != nil {
				return fmt.Errorf("failed to list backups: %w", err)
			}

			var items []map[string]any
			if err := json.Unmarshal(data, &items); err != nil {
				return fmt.Errorf("failed to list backups: %w", err)
			}

			headers := []string{"ID", "MODEL", "TYPE", "SCOPE", "STATUS", "DESCRIPTION", "CREATED"}
			var rows [][]string
			for _, item := range items {
				rows = append(rows, []string{
					getString(item, "id"),
					formatArchivableType(getString(item, "archivable_type")),
					getString(item, "type"),
					getString(item, "scope"),
					getString(item, "status"),
					formatString(getString(item, "description")),
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
	cmd.Flags().String("type", "", "Filter by backup type (site/environment)")
	return cmd
}

func newBackupShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a backup",
		Long:  "Display details of a specific backup.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Get(cmd.Context(), backupsBasePath+"/"+args[0], nil)
			if err != nil {
				return fmt.Errorf("failed to get backup: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to get backup: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to get backup: %w", err)
			}

			if app.Format == output.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to get backup: %w", err)
			}

			output.PrintKeyValue(cmd.OutOrStdout(), []output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Model", Value: formatArchivableType(getString(item, "archivable_type"))},
				{Key: "Model ID", Value: getString(item, "archivable_id")},
				{Key: "Type", Value: getString(item, "type")},
				{Key: "Scope", Value: getString(item, "scope")},
				{Key: "Status", Value: getString(item, "status")},
				{Key: "Description", Value: formatString(getString(item, "description"))},
				{Key: "File Snapshot ID", Value: formatString(getString(item, "file_snapshot_id"))},
				{Key: "Database Snapshot ID", Value: formatString(getString(item, "database_snapshot_id"))},
				{Key: "Started At", Value: formatString(getString(item, "started_at"))},
				{Key: "Completed At", Value: formatString(getString(item, "completed_at"))},
				{Key: "Created At", Value: getString(item, "created_at")},
				{Key: "Updated At", Value: getString(item, "updated_at")},
			})
			return nil
		},
	}
}

func newBackupCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a backup",
		Long:  "Create a new backup for a site or environment.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			siteID, _ := cmd.Flags().GetString("site-id")
			envID, _ := cmd.Flags().GetString("environment-id")

			if siteID == "" && envID == "" {
				return fmt.Errorf("either --site-id or --environment-id is required")
			}

			scope, _ := cmd.Flags().GetString("scope")
			description, _ := cmd.Flags().GetString("description")

			reqBody := map[string]any{
				"type":  "manual",
				"scope": scope,
			}

			if siteID != "" {
				reqBody["site_id"] = siteID
			}
			if envID != "" {
				reqBody["environment_id"] = envID
			}
			if description != "" {
				reqBody["description"] = description
			}

			resp, err := app.Client.Post(cmd.Context(), backupsBasePath, reqBody)
			if err != nil {
				return fmt.Errorf("failed to create backup: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to create backup: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to create backup: %w", err)
			}

			if app.Format == output.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to create backup: %w", err)
			}

			w := cmd.OutOrStdout()
			output.PrintMessage(w, fmt.Sprintf("Backup created: %s (%s)", getString(item, "id"), getString(item, "status")))
			output.PrintMessage(w, "")

			output.PrintKeyValue(w, []output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Model", Value: formatArchivableType(getString(item, "archivable_type"))},
				{Key: "Model ID", Value: getString(item, "archivable_id")},
				{Key: "Type", Value: getString(item, "type")},
				{Key: "Scope", Value: getString(item, "scope")},
				{Key: "Status", Value: getString(item, "status")},
				{Key: "Description", Value: formatString(getString(item, "description"))},
				{Key: "File Snapshot ID", Value: formatString(getString(item, "file_snapshot_id"))},
				{Key: "Database Snapshot ID", Value: formatString(getString(item, "database_snapshot_id"))},
				{Key: "Started At", Value: formatString(getString(item, "started_at"))},
				{Key: "Completed At", Value: formatString(getString(item, "completed_at"))},
				{Key: "Created At", Value: getString(item, "created_at")},
				{Key: "Updated At", Value: getString(item, "updated_at")},
			})
			return nil
		},
	}

	cmd.Flags().String("site-id", "", "Site ID to back up")
	cmd.Flags().String("environment-id", "", "Environment ID to back up")
	cmd.Flags().String("scope", "full", "Backup scope (full/database/files)")
	cmd.Flags().String("description", "", "Description for the backup")

	return cmd
}
