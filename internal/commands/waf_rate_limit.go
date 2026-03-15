package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/output"
)

func wafRateLimitsPath(siteID string) string {
	return sitesBasePath + "/" + siteID + "/waf/rate-limits"
}

// NewWafRateLimitCmd creates the waf rate-limit command group.
func NewWafRateLimitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rate-limit",
		Short: "Manage WAF rate limiting rules",
		Long:  "Manage WAF rate limiting rules to protect your sites from abuse.",
	}

	cmd.AddCommand(newWafRateLimitListCmd())
	cmd.AddCommand(newWafRateLimitShowCmd())
	cmd.AddCommand(newWafRateLimitCreateCmd())
	cmd.AddCommand(newWafRateLimitUpdateCmd())
	cmd.AddCommand(newWafRateLimitDeleteCmd())

	return cmd
}

func newWafRateLimitListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <site-id>",
		Short: "List WAF rate limiting rules",
		Long: "Retrieve all rate limit rules configured for a site.",
		Example: `  # List rate limit rules
  vector waf rate-limit list site-abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Get(cmd.Context(), wafRateLimitsPath(args[0]), nil)
			if err != nil {
				return fmt.Errorf("failed to list rate limits: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to list rate limits: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to list rate limits: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var items []map[string]any
			if err := json.Unmarshal(data, &items); err != nil {
				return fmt.Errorf("failed to list rate limits: %w", err)
			}

			headers := []string{"ID", "NAME", "REQUESTS/TIME", "BLOCK TIME"}
			var rows [][]string
			for _, item := range items {
				config := getMap(item, "configuration")
				reqCount := getFloat(config, "request_count")
				timeframe := getFloat(config, "timeframe")
				blockTime := getFloat(config, "block_time")

				rows = append(rows, []string{
					fmt.Sprintf("%.0f", getFloat(item, "id")),
					getString(item, "name"),
					fmt.Sprintf("%.0f/%.0fs", reqCount, timeframe),
					fmt.Sprintf("%.0fs", blockTime),
				})
			}

			app.Output.Table(headers, rows)
			return nil
		},
	}
}

func newWafRateLimitShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <site-id> <rule-id>",
		Short: "Show a WAF rate limiting rule",
		Long: "Display details of a specific rate limit rule.",
		Example: `  # Show rule details
  vector waf rate-limit show site-abc123 rule-42`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Get(cmd.Context(), wafRateLimitsPath(args[0])+"/"+args[1], nil)
			if err != nil {
				return fmt.Errorf("failed to get rate limit: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to get rate limit: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to get rate limit: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to get rate limit: %w", err)
			}

			config := getMap(item, "configuration")

			app.Output.KeyValue([]output.KeyValue{
				{Key: "ID", Value: fmt.Sprintf("%.0f", getFloat(item, "id"))},
				{Key: "Name", Value: getString(item, "name")},
				{Key: "Description", Value: formatString(getString(item, "description"))},
				{Key: "Request Count", Value: fmt.Sprintf("%.0f", getFloat(config, "request_count"))},
				{Key: "Timeframe", Value: fmt.Sprintf("%.0f", getFloat(config, "timeframe"))},
				{Key: "Block Time", Value: fmt.Sprintf("%.0f", getFloat(config, "block_time"))},
				{Key: "Value", Value: formatString(getString(config, "value"))},
				{Key: "Operator", Value: formatString(getString(config, "operator"))},
				{Key: "Variables", Value: formatSliceField(config, "variables")},
				{Key: "Transformations", Value: formatSliceField(config, "transformations")},
			})
			return nil
		},
	}
}

func newWafRateLimitCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <site-id>",
		Short: "Create a WAF rate limiting rule",
		Long: "Create a new rate limit rule for a site.",
		Example: `  # Create a rate limit rule
  vector waf rate-limit create site-abc123 --name "login-limit" --request-count 100 --timeframe 10 --block-time 60`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			name, _ := cmd.Flags().GetString("name")
			requestCount, _ := cmd.Flags().GetInt("request-count")
			timeframe, _ := cmd.Flags().GetInt("timeframe")
			blockTime, _ := cmd.Flags().GetInt("block-time")

			reqBody := map[string]any{
				"name":          name,
				"request_count": requestCount,
				"timeframe":     timeframe,
				"block_time":    blockTime,
			}

			if cmd.Flags().Changed("description") {
				desc, _ := cmd.Flags().GetString("description")
				reqBody["description"] = desc
			}

			if cmd.Flags().Changed("value") {
				value, _ := cmd.Flags().GetString("value")
				reqBody["value"] = value
			}

			if cmd.Flags().Changed("operator") {
				operator, _ := cmd.Flags().GetString("operator")
				reqBody["operator"] = operator
			}

			if cmd.Flags().Changed("variables") {
				vars, _ := cmd.Flags().GetString("variables")
				reqBody["variables"] = strings.Split(vars, ",")
			}

			if cmd.Flags().Changed("transformations") {
				trans, _ := cmd.Flags().GetString("transformations")
				reqBody["transformations"] = strings.Split(trans, ",")
			}

			resp, err := app.Client.Post(cmd.Context(), wafRateLimitsPath(args[0]), reqBody)
			if err != nil {
				return fmt.Errorf("failed to create rate limit: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to create rate limit: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to create rate limit: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to create rate limit: %w", err)
			}

			config := getMap(item, "configuration")

			app.Output.KeyValue([]output.KeyValue{
				{Key: "ID", Value: fmt.Sprintf("%.0f", getFloat(item, "id"))},
				{Key: "Name", Value: getString(item, "name")},
				{Key: "Description", Value: formatString(getString(item, "description"))},
				{Key: "Request Count", Value: fmt.Sprintf("%.0f", getFloat(config, "request_count"))},
				{Key: "Timeframe", Value: fmt.Sprintf("%.0f", getFloat(config, "timeframe"))},
				{Key: "Block Time", Value: fmt.Sprintf("%.0f", getFloat(config, "block_time"))},
				{Key: "Value", Value: formatString(getString(config, "value"))},
				{Key: "Operator", Value: formatString(getString(config, "operator"))},
				{Key: "Variables", Value: formatSliceField(config, "variables")},
				{Key: "Transformations", Value: formatSliceField(config, "transformations")},
			})
			return nil
		},
	}

	cmd.Flags().String("name", "", "Rule name (required)")
	cmd.Flags().Int("request-count", 0, "Number of requests allowed within the timeframe (required)")
	cmd.Flags().Int("timeframe", 0, "Time window in seconds (required)")
	cmd.Flags().Int("block-time", 0, "Duration to block in seconds (required)")
	cmd.Flags().String("description", "", "Rule description")
	cmd.Flags().String("value", "", "URL path or pattern to match")
	cmd.Flags().String("operator", "", "Match operator")
	cmd.Flags().String("variables", "", "Comma-separated request variables to inspect")
	cmd.Flags().String("transformations", "", "Comma-separated transformations to apply")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("request-count")
	_ = cmd.MarkFlagRequired("timeframe")
	_ = cmd.MarkFlagRequired("block-time")

	return cmd
}

func newWafRateLimitUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <site-id> <rule-id>",
		Short: "Update a WAF rate limiting rule",
		Long: "Update an existing rate limit rule. Only sends changed fields.",
		Example: `  # Update block time
  vector waf rate-limit update site-abc123 rule-42 --block-time 300`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			reqBody := map[string]any{}

			if cmd.Flags().Changed("name") {
				name, _ := cmd.Flags().GetString("name")
				reqBody["name"] = name
			}

			if cmd.Flags().Changed("description") {
				desc, _ := cmd.Flags().GetString("description")
				reqBody["description"] = desc
			}

			if cmd.Flags().Changed("request-count") {
				rc, _ := cmd.Flags().GetInt("request-count")
				reqBody["request_count"] = rc
			}

			if cmd.Flags().Changed("timeframe") {
				tf, _ := cmd.Flags().GetInt("timeframe")
				reqBody["timeframe"] = tf
			}

			if cmd.Flags().Changed("block-time") {
				bt, _ := cmd.Flags().GetInt("block-time")
				reqBody["block_time"] = bt
			}

			if cmd.Flags().Changed("value") {
				value, _ := cmd.Flags().GetString("value")
				reqBody["value"] = value
			}

			if cmd.Flags().Changed("operator") {
				operator, _ := cmd.Flags().GetString("operator")
				reqBody["operator"] = operator
			}

			if cmd.Flags().Changed("variables") {
				vars, _ := cmd.Flags().GetString("variables")
				reqBody["variables"] = strings.Split(vars, ",")
			}

			if cmd.Flags().Changed("transformations") {
				trans, _ := cmd.Flags().GetString("transformations")
				reqBody["transformations"] = strings.Split(trans, ",")
			}

			resp, err := app.Client.Put(cmd.Context(), wafRateLimitsPath(args[0])+"/"+args[1], reqBody)
			if err != nil {
				return fmt.Errorf("failed to update rate limit: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to update rate limit: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to update rate limit: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to update rate limit: %w", err)
			}

			config := getMap(item, "configuration")

			app.Output.KeyValue([]output.KeyValue{
				{Key: "ID", Value: fmt.Sprintf("%.0f", getFloat(item, "id"))},
				{Key: "Name", Value: getString(item, "name")},
				{Key: "Description", Value: formatString(getString(item, "description"))},
				{Key: "Request Count", Value: fmt.Sprintf("%.0f", getFloat(config, "request_count"))},
				{Key: "Timeframe", Value: fmt.Sprintf("%.0f", getFloat(config, "timeframe"))},
				{Key: "Block Time", Value: fmt.Sprintf("%.0f", getFloat(config, "block_time"))},
				{Key: "Value", Value: formatString(getString(config, "value"))},
				{Key: "Operator", Value: formatString(getString(config, "operator"))},
				{Key: "Variables", Value: formatSliceField(config, "variables")},
				{Key: "Transformations", Value: formatSliceField(config, "transformations")},
			})
			return nil
		},
	}

	cmd.Flags().String("name", "", "Rule name")
	cmd.Flags().String("description", "", "Rule description")
	cmd.Flags().Int("request-count", 0, "Number of requests allowed within the timeframe")
	cmd.Flags().Int("timeframe", 0, "Time window in seconds")
	cmd.Flags().Int("block-time", 0, "Duration to block in seconds")
	cmd.Flags().String("value", "", "URL path or pattern to match")
	cmd.Flags().String("operator", "", "Match operator")
	cmd.Flags().String("variables", "", "Comma-separated request variables to inspect")
	cmd.Flags().String("transformations", "", "Comma-separated transformations to apply")

	return cmd
}

func newWafRateLimitDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <site-id> <rule-id>",
		Short: "Delete a WAF rate limiting rule",
		Long: "Permanently delete a rate limit rule. This action cannot be undone.",
		Example: `  # Delete a rule
  vector waf rate-limit delete site-abc123 rule-42`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Delete(cmd.Context(), wafRateLimitsPath(args[0])+"/"+args[1])
			if err != nil {
				return fmt.Errorf("failed to delete rate limit: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to delete rate limit: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to delete rate limit: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			app.Output.Message("Rate limit rule deleted successfully.")
			return nil
		},
	}
}

// formatSliceField joins string elements of a slice field into a comma-separated string.
func formatSliceField(m map[string]any, key string) string {
	items := getSlice(m, key)
	if len(items) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			parts = append(parts, s)
		}
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ", ")
}
