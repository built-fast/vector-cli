package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/appctx"
	"github.com/built-fast/vector-cli/internal/config"
)

// newTestCmd creates a cobra.Command with a background context set.
func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())
	return cmd
}

// --- requireApp ---

func TestRequireApp_WithToken(t *testing.T) {
	cmd := newTestCmd()
	client := api.NewClient("http://localhost", "valid-token", "test")
	app := appctx.NewApp(config.DefaultConfig(), client, "")
	cmd.SetContext(appctx.WithApp(cmd.Context(), app))

	got, err := requireApp(cmd)
	require.NoError(t, err)
	assert.Equal(t, app, got)
}

func TestRequireApp_NoToken(t *testing.T) {
	cmd := newTestCmd()
	client := api.NewClient("http://localhost", "", "test")
	app := appctx.NewApp(config.DefaultConfig(), client, "")
	cmd.SetContext(appctx.WithApp(cmd.Context(), app))

	_, err := requireApp(cmd)
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestRequireApp_NilContext(t *testing.T) {
	cmd := newTestCmd()

	_, err := requireApp(cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "app not initialized")
}

// --- addPaginationFlags / getPagination ---

func TestPaginationFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	addPaginationFlags(cmd)

	cmd.SetArgs([]string{"--page", "3", "--per-page", "25"})
	require.NoError(t, cmd.Execute())

	page, perPage := getPagination(cmd)
	assert.Equal(t, 3, page)
	assert.Equal(t, 25, perPage)
}

func TestPaginationFlags_Defaults(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	addPaginationFlags(cmd)

	require.NoError(t, cmd.Execute())

	page, perPage := getPagination(cmd)
	assert.Equal(t, 0, page)
	assert.Equal(t, 0, perPage)
}

// --- buildPaginationQuery ---

func TestBuildPaginationQuery(t *testing.T) {
	tests := []struct {
		name          string
		page, perPage int
		wantPage      string
		wantPerPage   string
	}{
		{"explicit values", 2, 30, "2", "30"},
		{"defaults for zero", 0, 0, "1", "15"},
		{"defaults for negative", -1, -5, "1", "15"},
		{"page default only", 0, 10, "1", "10"},
		{"perPage default only", 5, 0, "5", "15"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := buildPaginationQuery(tt.page, tt.perPage)
			assert.Equal(t, tt.wantPage, q.Get("page"))
			assert.Equal(t, tt.wantPerPage, q.Get("per_page"))
		})
	}
}

// --- parseResponseData ---

func TestParseResponseData(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "object data",
			input: `{"data": {"id": 1, "name": "test"}}`,
			want:  `{"id": 1, "name": "test"}`,
		},
		{
			name:  "array data",
			input: `{"data": [{"id": 1}, {"id": 2}]}`,
			want:  `[{"id": 1}, {"id": 2}]`,
		},
		{
			name:    "missing data key",
			input:   `{"message": "ok"}`,
			wantErr: true,
		},
		{
			name:    "invalid json",
			input:   `not json`,
			wantErr: true,
		},
		{
			name:    "null data",
			input:   `{"data": null}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseResponseData([]byte(tt.input))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			// Compare as compact JSON
			assert.JSONEq(t, tt.want, string(got))
		})
	}
}

// --- parseResponseWithMeta ---

func TestParseResponseWithMeta(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantData string
		wantMeta *PaginationMeta
		wantErr  bool
	}{
		{
			name:     "full response",
			input:    `{"data": [{"id": 1}], "meta": {"current_page": 2, "last_page": 5, "total": 50}}`,
			wantData: `[{"id": 1}]`,
			wantMeta: &PaginationMeta{CurrentPage: 2, LastPage: 5, Total: 50},
		},
		{
			name:     "no meta",
			input:    `{"data": [{"id": 1}]}`,
			wantData: `[{"id": 1}]`,
			wantMeta: &PaginationMeta{},
		},
		{
			name:    "missing data",
			input:   `{"meta": {"current_page": 1}}`,
			wantErr: true,
		},
		{
			name:    "invalid json",
			input:   `{broken`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, meta, err := parseResponseWithMeta([]byte(tt.input))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.JSONEq(t, tt.wantData, string(data))
			assert.Equal(t, tt.wantMeta, meta)
		})
	}
}

// --- formatBool ---

func TestFormatBool(t *testing.T) {
	assert.Equal(t, "Yes", formatBool(true))
	assert.Equal(t, "No", formatBool(false))
}

// --- formatTags ---

func TestFormatTags(t *testing.T) {
	tests := []struct {
		name string
		tags []string
		want string
	}{
		{"multiple tags", []string{"go", "cli", "api"}, "go, cli, api"},
		{"single tag", []string{"go"}, "go"},
		{"empty slice", []string{}, "-"},
		{"nil slice", nil, "-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatTags(tt.tags))
		})
	}
}

// --- formatString ---

func TestFormatString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"non-empty", "hello", "hello"},
		{"empty", "", "-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatString(tt.input))
		})
	}
}

// --- safe map getters ---

func TestGetString(t *testing.T) {
	m := map[string]any{"name": "test", "count": 42, "nil": nil}

	assert.Equal(t, "test", getString(m, "name"))
	assert.Empty(t, getString(m, "missing"))
	assert.Empty(t, getString(m, "count"))
	assert.Empty(t, getString(m, "nil"))
}

func TestGetFloat(t *testing.T) {
	m := map[string]any{"count": 42.5, "name": "test", "nil": nil}

	assert.Equal(t, 42.5, getFloat(m, "count"))
	assert.Equal(t, 0.0, getFloat(m, "missing"))
	assert.Equal(t, 0.0, getFloat(m, "name"))
	assert.Equal(t, 0.0, getFloat(m, "nil"))
}

func TestGetBool(t *testing.T) {
	m := map[string]any{"active": true, "name": "test", "nil": nil}

	assert.True(t, getBool(m, "active"))
	assert.False(t, getBool(m, "missing"))
	assert.False(t, getBool(m, "name"))
	assert.False(t, getBool(m, "nil"))
}

func TestGetSlice(t *testing.T) {
	m := map[string]any{
		"tags": []any{"a", "b"},
		"name": "test",
		"nil":  nil,
	}

	assert.Equal(t, []any{"a", "b"}, getSlice(m, "tags"))
	assert.Nil(t, getSlice(m, "missing"))
	assert.Nil(t, getSlice(m, "name"))
	assert.Nil(t, getSlice(m, "nil"))
}

func TestGetMap(t *testing.T) {
	inner := map[string]any{"key": "val"}
	m := map[string]any{
		"nested": inner,
		"name":   "test",
		"nil":    nil,
	}

	assert.Equal(t, inner, getMap(m, "nested"))
	assert.Nil(t, getMap(m, "missing"))
	assert.Nil(t, getMap(m, "name"))
	assert.Nil(t, getMap(m, "nil"))
}

// --- printPaginationIfNeeded ---

func TestPrintPaginationIfNeeded(t *testing.T) {
	tests := []struct {
		name string
		meta *PaginationMeta
		want string
	}{
		{
			name: "multiple pages",
			meta: &PaginationMeta{CurrentPage: 1, LastPage: 3, Total: 45},
			want: "Page 1 of 3 (45 total)\n",
		},
		{
			name: "single page",
			meta: &PaginationMeta{CurrentPage: 1, LastPage: 1, Total: 5},
			want: "",
		},
		{
			name: "nil meta",
			meta: nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			printPaginationIfNeeded(&buf, tt.meta)
			assert.Equal(t, tt.want, buf.String())
		})
	}
}

// --- confirmAction ---

func TestConfirmAction(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"yes lowercase", "yes\n", true},
		{"y lowercase", "y\n", true},
		{"YES uppercase", "YES\n", true},
		{"Y uppercase", "Y\n", true},
		{"no", "no\n", false},
		{"empty", "\n", false},
		{"random text", "maybe\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origReader := confirmReader
			t.Cleanup(func() { confirmReader = origReader })

			confirmReader = strings.NewReader(tt.input)

			cmd := &cobra.Command{Use: "test"}
			stderr := new(bytes.Buffer)
			cmd.SetErr(stderr)

			got := confirmAction(cmd, "Delete this?")
			assert.Equal(t, tt.want, got)
			assert.Contains(t, stderr.String(), "Delete this? [y/N]: ")
		})
	}
}

func TestConfirmAction_EOF(t *testing.T) {
	origReader := confirmReader
	t.Cleanup(func() { confirmReader = origReader })

	confirmReader = strings.NewReader("")

	cmd := &cobra.Command{Use: "test"}
	stderr := new(bytes.Buffer)
	cmd.SetErr(stderr)

	assert.False(t, confirmAction(cmd, "Delete?"))
}

// --- integration: parseResponseData with JSON unmarshal ---

func TestParseResponseData_UnmarshalResult(t *testing.T) {
	input := `{"data": {"id": 42, "name": "site1", "active": true}}`
	data, err := parseResponseData([]byte(input))
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(data, &result))
	assert.Equal(t, 42.0, result["id"])
	assert.Equal(t, "site1", result["name"])
	assert.Equal(t, true, result["active"])
}
