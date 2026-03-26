package commands

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/output"
)

func importsPath(siteID string) string {
	return sitesBasePath + "/" + siteID + "/imports"
}

// NewDbImportSessionCmd creates the db import-session command group.
func NewDbImportSessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import-session",
		Short: "Manage database import sessions",
		Long:  "Manage database import sessions to import SQL dumps into your sites via a presigned upload URL.",
	}

	cmd.AddCommand(newDbImportSessionCreateCmd())
	cmd.AddCommand(newDbImportSessionRunCmd())
	cmd.AddCommand(newDbImportSessionStatusCmd())

	return cmd
}

func newDbImportSessionCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <site-id>",
		Short: "Create a database import session",
		Long:  "Create a new database import session. Returns a presigned upload URL for uploading a SQL dump file.",
		Example: `  # Create an import session
  vector db import-session create site-abc123 --content-length 52428800

  # Create with options
  vector db import-session create site-abc123 --content-length 52428800 --filename dump.sql --drop-tables`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			siteID := args[0]

			contentLength, _ := cmd.Flags().GetInt64("content-length")

			reqBody := map[string]any{
				"scope":          "database",
				"content_length": contentLength,
			}

			if cmd.Flags().Changed("filename") {
				v, _ := cmd.Flags().GetString("filename")
				if v != "" {
					reqBody["filename"] = v
				}
			}

			options := map[string]any{}

			dropTables, _ := cmd.Flags().GetBool("drop-tables")
			if dropTables {
				options["drop_tables"] = true
			}

			disableForeignKeys, _ := cmd.Flags().GetBool("disable-foreign-keys")
			if disableForeignKeys {
				options["disable_foreign_keys"] = true
			}

			srFrom, _ := cmd.Flags().GetString("search-replace-from")
			srTo, _ := cmd.Flags().GetString("search-replace-to")
			if srFrom != "" && srTo != "" {
				options["search_replace"] = map[string]string{
					"from": srFrom,
					"to":   srTo,
				}
			}

			if len(options) > 0 {
				reqBody["options"] = options
			}

			resp, err := app.Client.Post(cmd.Context(), importsPath(siteID), reqBody)
			if err != nil {
				return fmt.Errorf("failed to create import session: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to create import session: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to create import session: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to create import session: %w", err)
			}

			importID := getString(item, "id")

			if getBool(item, "is_multipart") {
				app.Output.KeyValue([]output.KeyValue{
					{Key: "Import ID", Value: importID},
					{Key: "Status", Value: getString(item, "status")},
					{Key: "Multipart", Value: "Yes"},
					{Key: "Part Count", Value: fmt.Sprintf("%.0f", getFloat(item, "part_count"))},
					{Key: "Upload ID", Value: getString(item, "upload_id")},
					{Key: "Expires", Value: formatString(getString(item, "upload_expires_at"))},
				})

				app.Output.Message("")
				app.Output.Message("This is a multipart upload. Use --json to get the presigned URLs for each part.")
				app.Output.Message(fmt.Sprintf("After uploading all parts, run: vector db import-session run %s %s --parts '<json>'", siteID, importID))
			} else {
				app.Output.KeyValue([]output.KeyValue{
					{Key: "Import ID", Value: importID},
					{Key: "Status", Value: getString(item, "status")},
					{Key: "Upload URL", Value: getString(item, "upload_url")},
					{Key: "Expires", Value: formatString(getString(item, "upload_expires_at"))},
				})

				app.Output.Message("")
				app.Output.Message(fmt.Sprintf("Upload your SQL file to the URL above, then run: vector db import-session run %s %s", siteID, importID))
			}

			return nil
		},
	}

	cmd.Flags().String("filename", "", "Name of the SQL dump file")
	cmd.Flags().Int64("content-length", 0, "File size in bytes (required)")
	_ = cmd.MarkFlagRequired("content-length")
	cmd.Flags().Bool("drop-tables", false, "Drop existing tables before import")
	cmd.Flags().Bool("disable-foreign-keys", false, "Disable foreign key checks during import")
	cmd.Flags().String("search-replace-from", "", "Value to search for (used with --search-replace-to)")
	cmd.Flags().String("search-replace-to", "", "Replacement value (used with --search-replace-from)")

	return cmd
}

func newDbImportSessionRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <site-id> <import-id>",
		Short: "Run a database import",
		Long:  "Execute a database import after the SQL dump has been uploaded to the presigned URL. For multipart uploads, pass the completed parts with --parts.",
		Example: `  # Run a single-part import
  vector db import-session run site-abc123 import-456

  # Run a multipart import
  vector db import-session run site-abc123 import-456 --parts '[{"part_number":1,"etag":"\"abc\""},{"part_number":2,"etag":"\"def\""}]'`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			siteID := args[0]
			importID := args[1]
			endpoint := fmt.Sprintf("%s/%s/run", importsPath(siteID), importID)

			partsJSON, _ := cmd.Flags().GetString("parts")
			var reqBody any
			if partsJSON != "" {
				var parts []map[string]any
				if err := json.Unmarshal([]byte(partsJSON), &parts); err != nil {
					return fmt.Errorf("failed to parse --parts: %w", err)
				}
				reqBody = map[string]any{"parts": parts}
			}

			resp, err := app.Client.Post(cmd.Context(), endpoint, reqBody)
			if err != nil {
				return fmt.Errorf("failed to run import: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to run import: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to run import: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to run import: %w", err)
			}

			app.Output.KeyValue([]output.KeyValue{
				{Key: "Import ID", Value: getString(item, "id")},
				{Key: "Status", Value: getString(item, "status")},
			})

			return nil
		},
	}

	cmd.Flags().String("parts", "", `Completed parts JSON for multipart uploads (e.g., '[{"part_number":1,"etag":"\"...\""}]')`)

	return cmd
}

func newDbImportSessionStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <site-id> <import-id>",
		Short: "Check database import status",
		Long:  "Retrieve the current status of a database import session.",
		Example: `  # Check import status
  vector db import-session status site-abc123 import-456`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			siteID := args[0]
			importID := args[1]
			endpoint := fmt.Sprintf("%s/%s", importsPath(siteID), importID)

			resp, err := app.Client.Get(cmd.Context(), endpoint, nil)
			if err != nil {
				return fmt.Errorf("failed to get import status: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to get import status: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to get import status: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to get import status: %w", err)
			}

			app.Output.KeyValue([]output.KeyValue{
				{Key: "Import ID", Value: getString(item, "id")},
				{Key: "Status", Value: getString(item, "status")},
				{Key: "Filename", Value: formatString(getString(item, "filename"))},
				{Key: "Duration", Value: formatFloat(getFloat(item, "duration_ms"))},
				{Key: "Error", Value: formatString(getString(item, "error_message"))},
				{Key: "Created", Value: getString(item, "created_at")},
				{Key: "Completed", Value: formatString(getString(item, "completed_at"))},
			})

			return nil
		},
	}
}
