package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/output"
)

const apiBasePath = "/api/v1/vector/"

// NewAPICmd creates the api passthrough command.
func NewAPICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "api <endpoint>",
		Short: "Make an authenticated request to the Vector Pro API",
		Long: "Send an authenticated HTTP request to any Vector Pro API endpoint and " +
			"print the raw response.\n\n" +
			"An <endpoint> beginning with \"/\" is sent verbatim against the base URL. " +
			"Any other value has \"/api/v1/vector/\" prepended, so \"sites\" resolves to " +
			"\"/api/v1/vector/sites\".",
		Example: `  # GET a resource that has no dedicated subcommand
  vector api php-versions

  # Equivalent to the line above, with an absolute path
  vector api /api/v1/vector/php-versions

  # Filter the response with built-in jq
  vector api sites --jq '.data[].id'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(cmd)
			if err != nil {
				return err
			}

			path := resolveAPIPath(args[0])

			resp, err := app.Client.Do(cmd.Context(), http.MethodGet, path, nil, nil)
			if err != nil {
				return fmt.Errorf("failed to make API request: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to make API request: %w", err)
			}

			return writeAPIResponse(app.Output, body)
		},
	}
}

// resolveAPIPath maps an endpoint argument to a request path. A value starting
// with "/" is returned verbatim; any other value is appended to the Vector Pro
// base path "/api/v1/vector/".
func resolveAPIPath(endpoint string) string {
	if strings.HasPrefix(endpoint, "/") {
		return endpoint
	}
	return apiBasePath + endpoint
}

// writeAPIResponse prints the response body. When the body parses as JSON it is
// pretty-printed (and the --jq filter, if set, is applied); otherwise the raw
// bytes are written verbatim.
func writeAPIResponse(w *output.Writer, body []byte) error {
	var parsed any
	if json.Unmarshal(body, &parsed) == nil {
		return w.JSON(parsed)
	}

	_, err := w.Underlying().Write(body)
	if err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}
	return nil
}
