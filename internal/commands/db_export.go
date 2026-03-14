package commands

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/output"
)

// NewDbExportCmd creates the db export command group.
func NewDbExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Manage database exports",
		Long:  "Create and check database export requests to download SQL dumps of site databases.",
	}

	cmd.AddCommand(newDbExportCreateCmd())
	cmd.AddCommand(newDbExportStatusCmd())

	return cmd
}

func newDbExportCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <site-id>",
		Short: "Create a database export",
		Long:  "Create a new database export for a site. The export is created with a pending status and processed asynchronously.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			siteID := args[0]
			endpoint := fmt.Sprintf("%s/%s/db/export", sitesBasePath, siteID)

			format, _ := cmd.Flags().GetString("format")
			payload := map[string]any{
				"format": format,
			}

			resp, err := app.Client.Post(cmd.Context(), endpoint, payload)
			if err != nil {
				return fmt.Errorf("failed to create database export: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to create database export: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to create database export: %w", err)
			}

			if app.Format == output.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to create database export: %w", err)
			}

			w := cmd.OutOrStdout()
			exportID := getString(item, "id")
			status := getString(item, "status")

			output.PrintMessage(w, fmt.Sprintf("Export started: %s (%s)", exportID, status))
			output.PrintMessage(w, fmt.Sprintf("Check status with: vector db export status %s %s", siteID, exportID))

			return nil
		},
	}

	cmd.Flags().String("format", "sql", "Export format")

	return cmd
}

func newDbExportStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <site-id> <export-id>",
		Short: "Check database export status",
		Long:  "Retrieve the status of a database export. Includes a download URL when the export is completed.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			siteID := args[0]
			exportID := args[1]
			endpoint := fmt.Sprintf("%s/%s/db/exports/%s", sitesBasePath, siteID, exportID)

			resp, err := app.Client.Get(cmd.Context(), endpoint, nil)
			if err != nil {
				return fmt.Errorf("failed to get database export status: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to get database export status: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to get database export status: %w", err)
			}

			if app.Format == output.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to get database export status: %w", err)
			}

			status := getString(item, "status")

			kvs := []output.KeyValue{
				{Key: "Export ID", Value: getString(item, "id")},
				{Key: "Status", Value: status},
				{Key: "Format", Value: formatString(getString(item, "format"))},
				{Key: "Size", Value: formatFloat(getFloat(item, "size_bytes"))},
				{Key: "Duration", Value: formatFloat(getFloat(item, "duration_ms"))},
				{Key: "Error", Value: formatString(getString(item, "error_message"))},
			}

			if status == "completed" {
				kvs = append(kvs, output.KeyValue{Key: "Download URL", Value: getString(item, "download_url")})
				kvs = append(kvs, output.KeyValue{Key: "Download Expires", Value: formatString(getString(item, "download_expires_at"))})
			}

			kvs = append(kvs,
				output.KeyValue{Key: "Created", Value: getString(item, "created_at")},
				output.KeyValue{Key: "Completed", Value: formatString(getString(item, "completed_at"))},
			)

			output.PrintKeyValue(cmd.OutOrStdout(), kvs)
			return nil
		},
	}
}
