package commands

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/output"
)

const eventsBasePath = "/api/v1/vector/events"

// NewEventCmd creates the event command group.
func NewEventCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "event",
		Short: "Manage events",
		Long:  "View account event logs for auditing activity.",
	}

	cmd.AddCommand(newEventListCmd())

	return cmd
}

func newEventListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List events",
		Long:  "Retrieve a paginated list of account event logs in reverse chronological order.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			page, perPage := getPagination(cmd)
			query := buildPaginationQuery(page, perPage)

			// Add optional filter flags
			if cmd.Flags().Changed("from") {
				v, _ := cmd.Flags().GetString("from")
				if v != "" {
					query.Set("from", v)
				}
			}
			if cmd.Flags().Changed("to") {
				v, _ := cmd.Flags().GetString("to")
				if v != "" {
					query.Set("to", v)
				}
			}
			if cmd.Flags().Changed("event") {
				v, _ := cmd.Flags().GetString("event")
				if v != "" {
					query.Set("event", v)
				}
			}

			resp, err := app.Client.Get(cmd.Context(), eventsBasePath, query)
			if err != nil {
				return fmt.Errorf("failed to list events: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to list events: %w", err)
			}

			if app.Output.Format() == output.JSON {
				data, err := parseResponseData(body)
				if err != nil {
					return fmt.Errorf("failed to list events: %w", err)
				}
				return app.Output.JSON(json.RawMessage(data))
			}

			data, meta, err := parseResponseWithMeta(body)
			if err != nil {
				return fmt.Errorf("failed to list events: %w", err)
			}

			var items []map[string]any
			if err := json.Unmarshal(data, &items); err != nil {
				return fmt.Errorf("failed to list events: %w", err)
			}

			headers := []string{"ID", "EVENT", "ACTOR", "RESOURCE", "CREATED"}
			var rows [][]string
			for _, item := range items {
				rows = append(rows, []string{
					getString(item, "id"),
					getString(item, "event"),
					formatActor(item),
					formatResource(item),
					getString(item, "created_at"),
				})
			}

			app.Output.Table(headers, rows)
			if meta != nil && meta.LastPage > 1 {
				app.Output.Pagination(meta.CurrentPage, meta.LastPage, meta.Total)
			}
			return nil
		},
	}

	addPaginationFlags(cmd)
	cmd.Flags().String("from", "", "Filter events from this ISO 8601 timestamp")
	cmd.Flags().String("to", "", "Filter events to this ISO 8601 timestamp")
	cmd.Flags().String("event", "", "Filter by event type (comma-separated)")

	return cmd
}

// formatActor formats the actor column: token name > IP > "-".
func formatActor(item map[string]any) string {
	actor := getMap(item, "actor")
	if actor == nil {
		return "-"
	}

	tokenName := getString(actor, "token_name")
	if tokenName != "" {
		return tokenName
	}

	ip := getString(actor, "ip")
	if ip != "" {
		return ip
	}

	return "-"
}

// formatResource formats the resource column as model_type:model_id or just model_type.
func formatResource(item map[string]any) string {
	modelType := getString(item, "model_type")
	if modelType == "" {
		return "-"
	}

	modelID := getString(item, "model_id")
	if modelID != "" {
		return modelType + ":" + modelID
	}

	return modelType
}
