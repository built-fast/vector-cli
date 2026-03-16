package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/output"
)

const deploysBasePath = "/api/v1/vector/deployments"

// NewDeployCmd creates the deploy command group.
func NewDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Manage deployments",
		Long:  "Manage Vector deployments including listing, viewing, triggering, and rolling back deployments.",
	}

	cmd.AddCommand(newDeployListCmd())
	cmd.AddCommand(newDeployShowCmd())
	cmd.AddCommand(newDeployTriggerCmd())
	cmd.AddCommand(newDeployRollbackCmd())

	return cmd
}

func newDeployListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <env-id>",
		Short: "List deployments for an environment",
		Long: "Retrieve a paginated list of deployments for an environment.",
		Example: `  # List deployments for an environment
  vector deploy list env-abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			page, perPage := getPagination(cmd)
			query := buildPaginationQuery(page, perPage)

			path := envsBasePath + "/" + args[0] + "/deployments"
			resp, err := app.Client.Get(cmd.Context(), path, query)
			if err != nil {
				return fmt.Errorf("failed to list deployments: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to list deployments: %w", err)
			}

			if app.Output.Format() == output.JSON {
				data, err := parseResponseData(body)
				if err != nil {
					return fmt.Errorf("failed to list deployments: %w", err)
				}
				return app.Output.JSON(json.RawMessage(data))
			}

			data, meta, err := parseResponseWithMeta(body)
			if err != nil {
				return fmt.Errorf("failed to list deployments: %w", err)
			}

			var items []map[string]any
			if err := json.Unmarshal(data, &items); err != nil {
				return fmt.Errorf("failed to list deployments: %w", err)
			}

			headers := []string{"ID", "STATUS", "ACTOR", "CREATED"}
			var rows [][]string
			for _, item := range items {
				rows = append(rows, []string{
					getString(item, "id"),
					getString(item, "status"),
					getString(item, "actor"),
					getString(item, "created_at"),
				})
			}

			app.Output.Table(headers, rows)
			if meta != nil {
				app.Output.Pagination(meta.CurrentPage, meta.LastPage, meta.Total)
			}
			return nil
		},
	}
	addPaginationFlags(cmd)
	return cmd
}

func newDeployShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <deploy-id>",
		Short: "Show deployment details",
		Long: "Retrieve details of a specific deployment, including stdout and stderr output.",
		Example: `  # Show deployment details
  vector deploy show deploy-456`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Get(cmd.Context(), deploysBasePath+"/"+args[0], nil)
			if err != nil {
				return fmt.Errorf("failed to show deployment: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to show deployment: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to show deployment: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to show deployment: %w", err)
			}

			w := cmd.OutOrStdout()
			app.Output.KeyValue([]output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Environment ID", Value: getString(item, "vector_environment_id")},
				{Key: "Status", Value: getString(item, "status")},
				{Key: "Actor", Value: getString(item, "actor")},
				{Key: "Created", Value: getString(item, "created_at")},
				{Key: "Updated", Value: getString(item, "updated_at")},
			})

			stdoutStr := getString(item, "stdout")
			if stdoutStr != "" {
				_, _ = fmt.Fprintln(w)
				_, _ = fmt.Fprintln(w, "Stdout:")
				_, _ = fmt.Fprintln(w, stdoutStr)
			}

			stderrStr := getString(item, "stderr")
			if stderrStr != "" {
				_, _ = fmt.Fprintln(w)
				_, _ = fmt.Fprintln(w, "Stderr:")
				_, _ = fmt.Fprintln(w, stderrStr)
			}

			return nil
		},
	}
}

func newDeployTriggerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trigger <env-id>",
		Short: "Trigger a deployment",
		Long:  "Initiate a new deployment for an environment.",
		Example: `  # Trigger a deployment
  vector deploy trigger env-abc123

  # Include uploads
  vector deploy trigger env-abc123 --include-uploads

  # Trigger and wait for completion
  vector deploy trigger env-abc123 --wait`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			waitEnabled, interval, timeout, err := getWaitConfig(cmd)
			if err != nil {
				return err
			}

			reqBody := map[string]any{}

			if cmd.Flags().Changed("include-uploads") {
				v, _ := cmd.Flags().GetBool("include-uploads")
				reqBody["include_uploads"] = v
			}
			if cmd.Flags().Changed("include-database") {
				v, _ := cmd.Flags().GetBool("include-database")
				reqBody["include_database"] = v
			}

			path := envsBasePath + "/" + args[0] + "/deployments"
			resp, err := app.Client.Post(cmd.Context(), path, reqBody)
			if err != nil {
				return fmt.Errorf("failed to trigger deployment: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to trigger deployment: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to trigger deployment: %w", err)
			}

			if !waitEnabled {
				if app.Output.Format() == output.JSON {
					return app.Output.JSON(json.RawMessage(data))
				}

				var item map[string]any
				if err := json.Unmarshal(data, &item); err != nil {
					return fmt.Errorf("failed to trigger deployment: %w", err)
				}

				app.Output.KeyValue([]output.KeyValue{
					{Key: "ID", Value: getString(item, "id")},
					{Key: "Environment ID", Value: getString(item, "vector_environment_id")},
					{Key: "Status", Value: getString(item, "status")},
					{Key: "Actor", Value: getString(item, "actor")},
					{Key: "Created", Value: getString(item, "created_at")},
				})
				return nil
			}

			var triggerItem map[string]any
			if err := json.Unmarshal(data, &triggerItem); err != nil {
				return fmt.Errorf("failed to trigger deployment: %w", err)
			}

			deployID := getString(triggerItem, "id")
			if deployID == "" {
				return fmt.Errorf("failed to trigger deployment: response missing deployment ID")
			}

			cfg := &waitConfig{
				ResourceID:       deployID,
				PollPath:         deploysBasePath + "/" + deployID,
				Interval:         interval,
				Timeout:          timeout,
				TerminalStatuses: map[string]bool{"deployed": true},
				FailedStatuses:   map[string]bool{"failed": true, "cancelled": true},
				Noun:             "Deployment",
				FormatDisplay:    deployFormatDisplay,
			}

			result, err := waitForResource(cmd.Context(), app, cfg)
			if err != nil {
				return err
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(result.FinalData)
			}

			var finalItem map[string]any
			if err := json.Unmarshal(result.FinalData, &finalItem); err != nil {
				return fmt.Errorf("failed to trigger deployment: %w", err)
			}

			app.Output.Message(fmt.Sprintf("Deployment %s %s in %s", deployID, result.Status, result.Elapsed.Truncate(time.Second)))
			app.Output.KeyValue([]output.KeyValue{
				{Key: "ID", Value: getString(finalItem, "id")},
				{Key: "Environment ID", Value: getString(finalItem, "vector_environment_id")},
				{Key: "Status", Value: getString(finalItem, "status")},
				{Key: "Actor", Value: getString(finalItem, "actor")},
				{Key: "Created", Value: getString(finalItem, "created_at")},
				{Key: "Updated", Value: getString(finalItem, "updated_at")},
			})
			return nil
		},
	}

	cmd.Flags().Bool("include-uploads", false, "Include wp-content/uploads in deployment")
	cmd.Flags().Bool("include-database", true, "Include database in deployment")
	addWaitFlags(cmd)

	return cmd
}

// deployFormatDisplay formats deployment data for the alternate screen display.
func deployFormatDisplay(data map[string]any) []string {
	return []string{
		fmt.Sprintf("%16s: %s", "ID", getString(data, "id")),
		fmt.Sprintf("%16s: %s", "Environment ID", getString(data, "vector_environment_id")),
		fmt.Sprintf("%16s: %s", "Status", getString(data, "status")),
		fmt.Sprintf("%16s: %s", "Actor", getString(data, "actor")),
		fmt.Sprintf("%16s: %s", "Created", getString(data, "created_at")),
		fmt.Sprintf("%16s: %s", "Updated", getString(data, "updated_at")),
	}
}

func newDeployRollbackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollback <env-id>",
		Short: "Rollback a deployment",
		Long: "Initiate a rollback for an environment. Rolls back to the last successful deployment unless a target is specified.",
		Example: `  # Rollback to the last successful deployment
  vector deploy rollback env-abc123

  # Rollback to a specific deployment
  vector deploy rollback env-abc123 --target deploy-789

  # Rollback and wait for completion
  vector deploy rollback env-abc123 --wait`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			waitEnabled, interval, timeout, err := getWaitConfig(cmd)
			if err != nil {
				return err
			}

			reqBody := map[string]any{}

			if cmd.Flags().Changed("target") {
				v, _ := cmd.Flags().GetString("target")
				reqBody["target_deployment_id"] = v
			}

			path := envsBasePath + "/" + args[0] + "/rollback"
			resp, err := app.Client.Post(cmd.Context(), path, reqBody)
			if err != nil {
				return fmt.Errorf("failed to rollback deployment: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to rollback deployment: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to rollback deployment: %w", err)
			}

			if !waitEnabled {
				if app.Output.Format() == output.JSON {
					return app.Output.JSON(json.RawMessage(data))
				}

				var item map[string]any
				if err := json.Unmarshal(data, &item); err != nil {
					return fmt.Errorf("failed to rollback deployment: %w", err)
				}

				app.Output.KeyValue([]output.KeyValue{
					{Key: "ID", Value: getString(item, "id")},
					{Key: "Environment ID", Value: getString(item, "vector_environment_id")},
					{Key: "Status", Value: getString(item, "status")},
					{Key: "Actor", Value: getString(item, "actor")},
					{Key: "Created", Value: getString(item, "created_at")},
				})
				return nil
			}

			var rollbackItem map[string]any
			if err := json.Unmarshal(data, &rollbackItem); err != nil {
				return fmt.Errorf("failed to rollback deployment: %w", err)
			}

			deployID := getString(rollbackItem, "id")
			if deployID == "" {
				return fmt.Errorf("failed to rollback deployment: response missing deployment ID")
			}

			cfg := &waitConfig{
				ResourceID:       deployID,
				PollPath:         deploysBasePath + "/" + deployID,
				Interval:         interval,
				Timeout:          timeout,
				TerminalStatuses: map[string]bool{"deployed": true},
				FailedStatuses:   map[string]bool{"failed": true, "cancelled": true},
				Noun:             "Deployment",
				FormatDisplay:    deployFormatDisplay,
			}

			result, err := waitForResource(cmd.Context(), app, cfg)
			if err != nil {
				return err
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(result.FinalData)
			}

			var finalItem map[string]any
			if err := json.Unmarshal(result.FinalData, &finalItem); err != nil {
				return fmt.Errorf("failed to rollback deployment: %w", err)
			}

			app.Output.Message(fmt.Sprintf("Deployment %s %s in %s", deployID, result.Status, result.Elapsed.Truncate(time.Second)))
			app.Output.KeyValue([]output.KeyValue{
				{Key: "ID", Value: getString(finalItem, "id")},
				{Key: "Environment ID", Value: getString(finalItem, "vector_environment_id")},
				{Key: "Status", Value: getString(finalItem, "status")},
				{Key: "Actor", Value: getString(finalItem, "actor")},
				{Key: "Created", Value: getString(finalItem, "created_at")},
				{Key: "Updated", Value: getString(finalItem, "updated_at")},
			})
			return nil
		},
	}

	cmd.Flags().String("target", "", "Target deployment ID to rollback to")
	addWaitFlags(cmd)

	return cmd
}
