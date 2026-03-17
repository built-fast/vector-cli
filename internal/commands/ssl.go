package commands

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/output"
)

// NewSSLCmd creates the ssl command group.
func NewSSLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssl",
		Short: "Manage SSL certificates",
		Long:  "Manage SSL certificate provisioning for environments, including checking status and nudging stuck provisioning.",
	}

	cmd.AddCommand(newSSLStatusCmd())
	cmd.AddCommand(newSSLNudgeCmd())

	return cmd
}

func newSSLStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <env-id>",
		Short: "Check SSL provisioning status",
		Long:  "Get the current SSL provisioning status for an environment.",
		Example: `  # Check SSL status
  vector ssl status env-abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			path := envsBasePath + "/" + args[0] + "/ssl"
			resp, err := app.Client.Get(cmd.Context(), path, nil)
			if err != nil {
				return fmt.Errorf("failed to get SSL status: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to get SSL status: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to get SSL status: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to get SSL status: %w", err)
			}

			app.Output.KeyValue([]output.KeyValue{
				{Key: "Status", Value: getString(item, "status")},
				{Key: "Provisioning Step", Value: formatString(getString(item, "provisioning_step"))},
				{Key: "Failure Reason", Value: formatString(getString(item, "failure_reason"))},
				{Key: "Production", Value: formatBool(getBool(item, "is_production"))},
				{Key: "Custom Domain", Value: formatString(getString(item, "custom_domain"))},
				{Key: "Platform Domain", Value: formatString(getString(item, "platform_domain"))},
			})
			return nil
		},
	}
}

func newSSLNudgeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nudge <env-id>",
		Short: "Nudge SSL provisioning",
		Long:  "Manually nudge SSL provisioning for an environment. Use this when SSL provisioning appears to be stuck or to retry after a failure.",
		Example: `  # Nudge SSL provisioning
  vector ssl nudge env-abc123

  # Retry from a failed state
  vector ssl nudge env-abc123 --retry`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			reqBody := map[string]any{}

			if cmd.Flags().Changed("retry") {
				v, _ := cmd.Flags().GetBool("retry")
				reqBody["retry"] = v
			}

			path := envsBasePath + "/" + args[0] + "/ssl/nudge"
			resp, err := app.Client.Post(cmd.Context(), path, reqBody)
			if err != nil {
				return fmt.Errorf("failed to nudge SSL: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to nudge SSL: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to nudge SSL: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			// Extract message from response
			var envelope struct {
				Message string `json:"message"`
			}
			if err := json.Unmarshal(body, &envelope); err == nil && envelope.Message != "" {
				app.Output.Message(envelope.Message)
			}

			var item map[string]any
			if err := json.Unmarshal(data, &item); err != nil {
				return fmt.Errorf("failed to nudge SSL: %w", err)
			}

			app.Output.KeyValue([]output.KeyValue{
				{Key: "Status", Value: getString(item, "status")},
				{Key: "Provisioning Step", Value: formatString(getString(item, "provisioning_step"))},
				{Key: "Failure Reason", Value: formatString(getString(item, "failure_reason"))},
				{Key: "Production", Value: formatBool(getBool(item, "is_production"))},
				{Key: "Custom Domain", Value: formatString(getString(item, "custom_domain"))},
				{Key: "Platform Domain", Value: formatString(getString(item, "platform_domain"))},
			})
			return nil
		},
	}

	cmd.Flags().Bool("retry", false, "Retry from a failed state")

	return cmd
}
