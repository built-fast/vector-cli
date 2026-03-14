package commands

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/output"
)

// NewEnvDBCmd creates the env db command group.
func NewEnvDBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Manage environment database",
		Long:  "Manage database operations for an environment, including promotes.",
	}

	cmd.AddCommand(newEnvDBPromoteCmd())
	cmd.AddCommand(newEnvDBPromoteStatusCmd())

	return cmd
}

func newEnvDBPromoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "promote <env-id>",
		Short: "Promote database",
		Long:  "Initiate a database promote for an environment. Copies the development database to the environment.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			reqBody := map[string]any{}

			if cmd.Flags().Changed("drop-tables") {
				v, _ := cmd.Flags().GetBool("drop-tables")
				reqBody["drop_tables"] = v
			}
			if cmd.Flags().Changed("disable-foreign-keys") {
				v, _ := cmd.Flags().GetBool("disable-foreign-keys")
				reqBody["disable_foreign_keys"] = v
			}

			path := envsBasePath + "/" + args[0] + "/db/promote"
			resp, err := app.Client.Post(cmd.Context(), path, reqBody)
			if err != nil {
				return fmt.Errorf("failed to promote database: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to promote database: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to promote database: %w", err)
			}

			if app.Format == output.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to promote database: %w", err)
			}

			output.PrintKeyValue(cmd.OutOrStdout(), []output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Environment ID", Value: getString(item, "vector_environment_id")},
				{Key: "Status", Value: getString(item, "status")},
				{Key: "Created", Value: getString(item, "created_at")},
			})
			return nil
		},
	}

	cmd.Flags().Bool("drop-tables", true, "Drop existing tables before promote")
	cmd.Flags().Bool("disable-foreign-keys", true, "Disable foreign key checks during promote")

	return cmd
}

func newEnvDBPromoteStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "promote-status <env-id> <promote-id>",
		Short: "Check promote status",
		Long:  "Check the status of a database promote operation.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			path := envsBasePath + "/" + args[0] + "/db/promotes/" + args[1]
			resp, err := app.Client.Get(cmd.Context(), path, nil)
			if err != nil {
				return fmt.Errorf("failed to get promote status: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to get promote status: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to get promote status: %w", err)
			}

			if app.Format == output.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to get promote status: %w", err)
			}

			pairs := []output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Environment ID", Value: getString(item, "vector_environment_id")},
				{Key: "Status", Value: getString(item, "status")},
				{Key: "Created", Value: getString(item, "created_at")},
			}

			startedAt := getString(item, "started_at")
			if startedAt != "" {
				pairs = append(pairs, output.KeyValue{Key: "Started", Value: startedAt})
			}

			completedAt := getString(item, "completed_at")
			if completedAt != "" {
				pairs = append(pairs, output.KeyValue{Key: "Completed", Value: completedAt})
			}

			durationMs := getFloat(item, "duration_ms")
			if durationMs > 0 {
				pairs = append(pairs, output.KeyValue{Key: "Duration", Value: fmt.Sprintf("%.0fms", durationMs)})
			}

			errorMsg := getString(item, "error_message")
			if errorMsg != "" {
				pairs = append(pairs, output.KeyValue{Key: "Error", Value: errorMsg})
			}

			output.PrintKeyValue(cmd.OutOrStdout(), pairs)
			return nil
		},
	}
}
