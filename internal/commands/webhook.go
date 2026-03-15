package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/output"
)

const webhooksBasePath = "/api/v1/vector/webhooks"

// NewWebhookCmd creates the webhook command group.
func NewWebhookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhook",
		Short: "Manage webhooks",
		Long:  "Manage webhooks for receiving notifications about account events.",
	}

	cmd.AddCommand(newWebhookListCmd())
	cmd.AddCommand(newWebhookShowCmd())
	cmd.AddCommand(newWebhookCreateCmd())
	cmd.AddCommand(newWebhookUpdateCmd())
	cmd.AddCommand(newWebhookDeleteCmd())

	return cmd
}

func newWebhookListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List webhooks",
		Long: "Retrieve a paginated list of webhooks for your account.",
		Example: `  # List webhooks
  vector webhook list`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			page, perPage := getPagination(cmd)
			query := buildPaginationQuery(page, perPage)

			resp, err := app.Client.Get(cmd.Context(), webhooksBasePath, query)
			if err != nil {
				return fmt.Errorf("failed to list webhooks: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to list webhooks: %w", err)
			}

			if app.Output.Format() == output.JSON {
				data, err := parseResponseData(body)
				if err != nil {
					return fmt.Errorf("failed to list webhooks: %w", err)
				}
				return app.Output.JSON(json.RawMessage(data))
			}

			data, meta, err := parseResponseWithMeta(body)
			if err != nil {
				return fmt.Errorf("failed to list webhooks: %w", err)
			}

			var items []map[string]any
			if err := json.Unmarshal(data, &items); err != nil {
				return fmt.Errorf("failed to list webhooks: %w", err)
			}

			headers := []string{"ID", "TYPE", "URL", "ENABLED"}
			var rows [][]string
			for _, item := range items {
				rows = append(rows, []string{
					getString(item, "id"),
					getString(item, "type"),
					getString(item, "url"),
					formatBool(getBool(item, "enabled")),
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

func newWebhookShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a webhook",
		Long: "Display details of a specific webhook.",
		Example: `  # Show webhook details
  vector webhook show webhook-456`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Get(cmd.Context(), webhooksBasePath+"/"+args[0], nil)
			if err != nil {
				return fmt.Errorf("failed to get webhook: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to get webhook: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to get webhook: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to get webhook: %w", err)
			}

			app.Output.KeyValue([]output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Type", Value: getString(item, "type")},
				{Key: "URL", Value: getString(item, "url")},
				{Key: "Enabled", Value: formatBool(getBool(item, "enabled"))},
				{Key: "Events", Value: formatEvents(item)},
				{Key: "Created", Value: getString(item, "created_at")},
				{Key: "Updated", Value: getString(item, "updated_at")},
			})
			return nil
		},
	}
}

func newWebhookCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a webhook",
		Long: "Create a new webhook for receiving notifications about account events.",
		Example: `  # Create a webhook
  vector webhook create --url https://example.com/hooks/vector --events "site.created,deploy.completed"

  # Create a Slack webhook
  vector webhook create --url https://hooks.slack.com/services/xxx --events "deploy.completed" --type slack`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			urlFlag, _ := cmd.Flags().GetString("url")
			eventsStr, _ := cmd.Flags().GetString("events")
			webhookType, _ := cmd.Flags().GetString("type")

			events := strings.Split(eventsStr, ",")

			reqBody := map[string]any{
				"url":    urlFlag,
				"events": events,
				"type":   webhookType,
			}

			resp, err := app.Client.Post(cmd.Context(), webhooksBasePath, reqBody)
			if err != nil {
				return fmt.Errorf("failed to create webhook: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to create webhook: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to create webhook: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to create webhook: %w", err)
			}

			w := cmd.OutOrStdout()
			kvs := []output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Type", Value: getString(item, "type")},
				{Key: "URL", Value: getString(item, "url")},
				{Key: "Enabled", Value: formatBool(getBool(item, "enabled"))},
				{Key: "Events", Value: formatEvents(item)},
				{Key: "Created", Value: getString(item, "created_at")},
			}

			secret := getString(item, "secret")
			if secret != "" {
				kvs = append(kvs, output.KeyValue{Key: "Secret", Value: secret})
			}

			app.Output.KeyValue(kvs)

			if secret != "" {
				output.PrintMessage(w, "")
				output.PrintMessage(w, "Save this secret — it won't be shown again!")
			}
			return nil
		},
	}

	cmd.Flags().String("url", "", "Webhook URL (required)")
	cmd.Flags().String("events", "", "Comma-separated event types (required)")
	cmd.Flags().String("type", "http", "Webhook type (http or slack)")
	_ = cmd.MarkFlagRequired("url")
	_ = cmd.MarkFlagRequired("events")

	return cmd
}

func newWebhookUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a webhook",
		Long: "Update an existing webhook configuration.",
		Example: `  # Update webhook URL
  vector webhook update webhook-456 --url https://example.com/hooks/new

  # Disable a webhook
  vector webhook update webhook-456 --enabled`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			reqBody := map[string]any{}

			if cmd.Flags().Changed("url") {
				urlFlag, _ := cmd.Flags().GetString("url")
				reqBody["url"] = urlFlag
			}

			if cmd.Flags().Changed("events") {
				eventsStr, _ := cmd.Flags().GetString("events")
				events := strings.Split(eventsStr, ",")
				reqBody["events"] = events
			}

			if cmd.Flags().Changed("enabled") {
				enabled, _ := cmd.Flags().GetBool("enabled")
				reqBody["enabled"] = enabled
			}

			resp, err := app.Client.Put(cmd.Context(), webhooksBasePath+"/"+args[0], reqBody)
			if err != nil {
				return fmt.Errorf("failed to update webhook: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to update webhook: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to update webhook: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to update webhook: %w", err)
			}

			app.Output.KeyValue([]output.KeyValue{
				{Key: "ID", Value: getString(item, "id")},
				{Key: "Type", Value: getString(item, "type")},
				{Key: "URL", Value: getString(item, "url")},
				{Key: "Enabled", Value: formatBool(getBool(item, "enabled"))},
				{Key: "Events", Value: formatEvents(item)},
				{Key: "Created", Value: getString(item, "created_at")},
				{Key: "Updated", Value: getString(item, "updated_at")},
			})
			return nil
		},
	}

	cmd.Flags().String("url", "", "Webhook URL")
	cmd.Flags().String("events", "", "Comma-separated event types")
	cmd.Flags().Bool("enabled", false, "Whether the webhook is enabled")

	return cmd
}

func newWebhookDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a webhook",
		Long: "Delete a webhook. All associated delivery logs will also be deleted.",
		Example: `  # Delete a webhook
  vector webhook delete webhook-456`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Delete(cmd.Context(), webhooksBasePath+"/"+args[0])
			if err != nil {
				return fmt.Errorf("failed to delete webhook: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to delete webhook: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to delete webhook: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			output.PrintMessage(cmd.OutOrStdout(), "Webhook deleted successfully.")
			return nil
		},
	}
}

// formatEvents joins the events array into a comma-separated string.
func formatEvents(item map[string]any) string {
	events := getSlice(item, "events")
	if len(events) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(events))
	for _, e := range events {
		if s, ok := e.(string); ok {
			parts = append(parts, s)
		}
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ", ")
}
