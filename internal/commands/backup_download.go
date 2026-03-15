package commands

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/output"
)

// NewBackupDownloadCmd creates the backup download command group.
func NewBackupDownloadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Manage backup downloads",
		Long:  "Create and check backup download requests to retrieve backup archives.",
	}

	cmd.AddCommand(newBackupDownloadCreateCmd())
	cmd.AddCommand(newBackupDownloadStatusCmd())

	return cmd
}

func newBackupDownloadCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <backup-id>",
		Short: "Create a backup download",
		Long:  "Create a new download request for a backup. The download is created with a pending status and processed asynchronously.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			backupID := args[0]
			endpoint := fmt.Sprintf("%s/%s/downloads", backupsBasePath, backupID)

			resp, err := app.Client.Post(cmd.Context(), endpoint, nil)
			if err != nil {
				return fmt.Errorf("failed to create backup download: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to create backup download: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to create backup download: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to create backup download: %w", err)
			}

			downloadID := getString(item, "id")

			app.Output.KeyValue([]output.KeyValue{
				{Key: "ID", Value: downloadID},
				{Key: "Status", Value: getString(item, "status")},
			})

			w := cmd.OutOrStdout()
			output.PrintMessage(w, "")
			output.PrintMessage(w, fmt.Sprintf("Check download status with: vector backup download status %s %s", backupID, downloadID))

			return nil
		},
	}
}

func newBackupDownloadStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <backup-id> <download-id>",
		Short: "Check backup download status",
		Long:  "Retrieve the status of a backup download. Includes a download URL when the download is completed.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			backupID := args[0]
			downloadID := args[1]
			endpoint := fmt.Sprintf("%s/%s/downloads/%s", backupsBasePath, backupID, downloadID)

			resp, err := app.Client.Get(cmd.Context(), endpoint, nil)
			if err != nil {
				return fmt.Errorf("failed to get backup download status: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to get backup download status: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to get backup download status: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to get backup download status: %w", err)
			}

			status := getString(item, "status")

			kvs := []output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Status", Value: status},
				{Key: "Size", Value: formatFloat(getFloat(item, "size_bytes"))},
				{Key: "Duration", Value: formatFloat(getFloat(item, "duration_ms"))},
				{Key: "Error", Value: formatString(getString(item, "error_message"))},
			}

			if status == "completed" {
				kvs = append(kvs, output.KeyValue{Key: "Download URL", Value: getString(item, "download_url")})
				kvs = append(kvs, output.KeyValue{Key: "Download Expires", Value: formatString(getString(item, "download_expires_at"))})
			}

			kvs = append(kvs,
				output.KeyValue{Key: "Started At", Value: formatString(getString(item, "started_at"))},
				output.KeyValue{Key: "Completed At", Value: formatString(getString(item, "completed_at"))},
				output.KeyValue{Key: "Created At", Value: getString(item, "created_at")},
			)

			app.Output.KeyValue(kvs)
			return nil
		},
	}
}

// formatFloat formats a float64 for display, returning "-" for zero values.
func formatFloat(v float64) string {
	if v == 0 {
		return "-"
	}
	return fmt.Sprintf("%.0f", v)
}
