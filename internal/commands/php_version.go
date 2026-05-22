package commands

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/output"
)

const phpVersionsPath = "/api/v1/vector/php-versions"

// NewPHPVersionsCmd creates the php-versions command.
func NewPHPVersionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "php-versions",
		Short: "List available PHP versions",
		Long:  "Retrieve a list of all available PHP versions for Vector environments.",
		Example: `  # List available PHP versions
  vector php-versions`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			resp, err := app.Client.Get(cmd.Context(), phpVersionsPath, nil)
			if err != nil {
				return fmt.Errorf("failed to list PHP versions: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to list PHP versions: %w", err)
			}

			data, err := parseResponseData(body)
			if err != nil {
				return fmt.Errorf("failed to list PHP versions: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(json.RawMessage(data))
			}

			var items []map[string]any
			if err := json.Unmarshal(data, &items); err != nil {
				return fmt.Errorf("failed to list PHP versions: %w", err)
			}

			rows := make([][]string, 0, len(items))
			for _, item := range items {
				rows = append(rows, []string{getString(item, "value")})
			}

			app.Output.Table([]string{"VERSION"}, rows)
			return nil
		},
	}
}
