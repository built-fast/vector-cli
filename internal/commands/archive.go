package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/appctx"
	"github.com/built-fast/vector-cli/internal/output"
)

func uploadMultipart(cmd *cobra.Command, app *appctx.App, file *os.File, fileSize int64, filename string, uploadParts []any) ([]map[string]any, error) {
	w := cmd.ErrOrStderr()
	partCount := int64(len(uploadParts))
	baseSize := fileSize / partCount
	lastSize := fileSize - baseSize*(partCount-1)

	sizeMB := float64(fileSize) / (1024 * 1024)
	_, _ = fmt.Fprintf(w, "Uploading %s (%.1f MB) in %d parts...\n", filename, sizeMB, partCount)

	completedParts := make([]map[string]any, 0, partCount)

	for i, part := range uploadParts {
		partMap, ok := part.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid upload part at index %d", i)
		}

		partNumber := int(getFloat(partMap, "part_number"))
		partURL := getString(partMap, "url")
		if partURL == "" {
			return nil, fmt.Errorf("upload part %d missing URL", partNumber)
		}

		chunkSize := baseSize
		if i == len(uploadParts)-1 {
			chunkSize = lastSize
		}
		offset := baseSize * int64(i)

		_, _ = fmt.Fprintf(w, "  Uploading part %d/%d...\n", partNumber, partCount)

		section := io.NewSectionReader(file, offset, chunkSize)
		etag, err := app.Client.PutFilePart(cmd.Context(), partURL, section, chunkSize)
		if err != nil {
			return nil, fmt.Errorf("failed to upload part %d: %w", partNumber, err)
		}

		completedParts = append(completedParts, map[string]any{
			"part_number": partNumber,
			"etag":        etag,
		})
	}

	_, _ = fmt.Fprintln(w, "Upload complete.")
	return completedParts, nil
}

// NewArchiveCmd creates the archive command group.
func NewArchiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive",
		Short: "Manage site archives",
		Long:  "Manage site archives including importing archive files.",
	}

	cmd.AddCommand(newArchiveImportCmd())

	return cmd
}

func newArchiveImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import <site-id> <file>",
		Short: "Import a site archive from a local file",
		Long:  "Import a site archive from a local file. Creates an import session, uploads the file to a presigned URL, and triggers the import.",
		Example: `  # Import an archive
  vector archive import site-abc123 site-backup.tar.gz

  # Import with search and replace
  vector archive import site-abc123 site.tar.gz --search-replace-from old.example.com --search-replace-to new.example.com`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			siteID := args[0]
			filePath := args[1]

			// Open and stat the file
			file, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("cannot open file: %w", err)
			}
			defer func() { _ = file.Close() }()

			fi, err := file.Stat()
			if err != nil {
				return fmt.Errorf("cannot read file info: %w", err)
			}

			filename := filepath.Base(filePath)
			fileSize := fi.Size()
			w := cmd.ErrOrStderr()

			// Step 1: Create import session
			_, _ = fmt.Fprintln(w, "Creating import session...")

			reqBody := map[string]any{
				"filename":       filename,
				"content_length": fileSize,
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

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to create import session: %w", err)
			}

			importID := getString(item, "id")
			if importID == "" {
				return fmt.Errorf("import session response missing import ID")
			}

			// Step 2: Upload file
			var runBody any

			if getBool(item, "is_multipart") {
				uploadParts := getSlice(item, "upload_parts")
				if len(uploadParts) == 0 {
					return fmt.Errorf("multipart import session response missing upload parts")
				}

				completedParts, uploadErr := uploadMultipart(cmd, app, file, fileSize, filename, uploadParts)
				if uploadErr != nil {
					return uploadErr
				}

				runBody = map[string]any{"parts": completedParts}
			} else {
				uploadURL := getString(item, "upload_url")
				if uploadURL == "" {
					return fmt.Errorf("import session response missing upload URL")
				}

				sizeMB := float64(fileSize) / (1024 * 1024)
				_, _ = fmt.Fprintf(w, "Uploading %s (%.1f MB)...\n", filename, sizeMB)

				uploadResp, uploadErr := app.Client.PutFile(cmd.Context(), uploadURL, file)
				if uploadErr != nil {
					return fmt.Errorf("failed to upload file: %w", uploadErr)
				}
				defer func() { _ = uploadResp.Body.Close() }()

				_, _ = fmt.Fprintln(w, "Upload complete.")
			}

			// Step 3: Trigger import
			_, _ = fmt.Fprintln(w, "Starting import...")

			runEndpoint := fmt.Sprintf("%s/%s/run", importsPath(siteID), importID)
			runResp, err := app.Client.Post(cmd.Context(), runEndpoint, runBody)
			if err != nil {
				return fmt.Errorf("failed to start import: %w", err)
			}
			defer func() { _ = runResp.Body.Close() }()

			runRespBody, err := io.ReadAll(runResp.Body)
			if err != nil {
				return fmt.Errorf("failed to start import: %w", err)
			}

			runData, err := parseResponseData(runRespBody)
			if err != nil {
				return fmt.Errorf("failed to start import: %w", err)
			}

			_, _ = fmt.Fprintln(w, "Import started.")

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(runData)
			}

			var runItem map[string]any
			if err := json.Unmarshal(runData, &runItem); err != nil {
				return fmt.Errorf("failed to parse import result: %w", err)
			}

			app.Output.KeyValue([]output.KeyValue{
				{Key: "Import ID", Value: getString(runItem, "id")},
				{Key: "Status", Value: getString(runItem, "status")},
			})

			return nil
		},
	}

	cmd.Flags().Bool("drop-tables", false, "Drop existing tables before import")
	cmd.Flags().Bool("disable-foreign-keys", false, "Disable foreign key checks during import")
	cmd.Flags().String("search-replace-from", "", "Value to search for (used with --search-replace-to)")
	cmd.Flags().String("search-replace-to", "", "Replacement value (used with --search-replace-from)")

	return cmd
}
