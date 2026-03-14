package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/appctx"
	"github.com/built-fast/vector-cli/internal/output"
)

// confirmReader is the reader used for confirmation prompts. Override in tests.
var confirmReader io.Reader = os.Stdin

// PaginationMeta holds pagination metadata from API responses.
type PaginationMeta struct {
	CurrentPage int `json:"current_page"`
	LastPage    int `json:"last_page"`
	Total       int `json:"total"`
}

// requireApp extracts *appctx.App from the command context and returns an error
// if no API token is set.
func requireApp(cmd *cobra.Command) (*appctx.App, error) {
	app := appctx.FromContext(cmd.Context())
	if app == nil {
		return nil, fmt.Errorf("app not initialized")
	}
	if app.Client.Token == "" {
		return nil, &api.APIError{
			Message:  "Authentication required. Run 'vector auth login' first.",
			ExitCode: 2,
		}
	}
	return app, nil
}

// addPaginationFlags adds --page and --per-page flags to a command.
func addPaginationFlags(cmd *cobra.Command) {
	cmd.Flags().Int("page", 0, "Page number")
	cmd.Flags().Int("per-page", 0, "Items per page")
}

// getPagination reads --page and --per-page flag values from the command.
func getPagination(cmd *cobra.Command) (page, perPage int) {
	page, _ = cmd.Flags().GetInt("page")
	perPage, _ = cmd.Flags().GetInt("per-page")
	return page, perPage
}

// buildPaginationQuery creates url.Values with page and per_page parameters.
// Defaults to page=1 and per_page=15 when values are <= 0.
func buildPaginationQuery(page, perPage int) url.Values {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 15
	}
	q := url.Values{}
	q.Set("page", strconv.Itoa(page))
	q.Set("per_page", strconv.Itoa(perPage))
	return q
}

// parseResponseData parses a JSON response with a "data" key and returns the
// raw JSON for the data value (works for both objects and arrays).
func parseResponseData(resp []byte) (json.RawMessage, error) {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(resp, &envelope); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil, fmt.Errorf("response missing \"data\" key")
	}
	return envelope.Data, nil
}

// parseResponseWithMeta parses a JSON response with "data" and "meta" keys,
// returning the raw data and pagination metadata.
func parseResponseWithMeta(resp []byte) (json.RawMessage, *PaginationMeta, error) {
	var envelope struct {
		Data json.RawMessage `json:"data"`
		Meta struct {
			CurrentPage int `json:"current_page"`
			LastPage    int `json:"last_page"`
			Total       int `json:"total"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(resp, &envelope); err != nil {
		return nil, nil, fmt.Errorf("parsing response: %w", err)
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil, nil, fmt.Errorf("response missing \"data\" key")
	}
	meta := &PaginationMeta{
		CurrentPage: envelope.Meta.CurrentPage,
		LastPage:    envelope.Meta.LastPage,
		Total:       envelope.Meta.Total,
	}
	return envelope.Data, meta, nil
}

// formatBool returns "Yes" for true, "No" for false.
func formatBool(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}

// formatTags joins a tag slice with ", " or returns "-" if empty.
func formatTags(tags []string) string {
	if len(tags) == 0 {
		return "-"
	}
	return strings.Join(tags, ", ")
}

// formatString returns the string or "-" if empty.
func formatString(v string) string {
	if v == "" {
		return "-"
	}
	return v
}

// getString safely gets a string value from a map, returning "" if missing or wrong type.
func getString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// getFloat safely gets a float64 value from a map, returning 0 if missing or wrong type.
func getFloat(m map[string]any, key string) float64 {
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	f, ok := v.(float64)
	if !ok {
		return 0
	}
	return f
}

// getBool safely gets a bool value from a map, returning false if missing or wrong type.
func getBool(m map[string]any, key string) bool {
	v, ok := m[key]
	if !ok || v == nil {
		return false
	}
	b, ok := v.(bool)
	if !ok {
		return false
	}
	return b
}

// getSlice safely gets a []any value from a map, returning nil if missing or wrong type.
func getSlice(m map[string]any, key string) []any {
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	s, ok := v.([]any)
	if !ok {
		return nil
	}
	return s
}

// getMap safely gets a map[string]any value from a map, returning nil if missing or wrong type.
func getMap(m map[string]any, key string) map[string]any {
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	m2, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	return m2
}

// printPaginationIfNeeded prints a pagination line only when there are multiple pages.
func printPaginationIfNeeded(w io.Writer, meta *PaginationMeta) {
	if meta != nil && meta.LastPage > 1 {
		output.PrintPagination(w, meta.CurrentPage, meta.LastPage, meta.Total)
	}
}

// confirmAction prompts the user on stderr and returns true only for "y" or "yes" input.
func confirmAction(cmd *cobra.Command, prompt string) bool {
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%s [y/N]: ", prompt)
	scanner := bufio.NewScanner(confirmReader)
	if !scanner.Scan() {
		return false
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return answer == "y" || answer == "yes"
}
