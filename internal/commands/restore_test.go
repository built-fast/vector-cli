package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/appctx"
	"github.com/built-fast/vector-cli/internal/config"
	"github.com/built-fast/vector-cli/internal/output"
)

var restoreListResponse = map[string]any{
	"data": []map[string]any{
		{
			"id":               "rst-001",
			"archivable_type":  "vector_site",
			"archivable_id":    "site-001",
			"scope":            "full",
			"trigger":          "manual",
			"status":           "completed",
			"vector_backup_id": "bk-001",
			"search_replace":   nil,
			"drop_tables":      false,
			"disable_foreign_keys": false,
			"error_message":    nil,
			"duration_ms":      float64(45200),
			"started_at":       "2025-01-15T12:00:00+00:00",
			"completed_at":     "2025-01-15T12:05:00+00:00",
			"created_at":       "2025-01-15T12:00:00+00:00",
			"updated_at":       "2025-01-15T12:05:00+00:00",
		},
		{
			"id":               "rst-002",
			"archivable_type":  "vector_environment",
			"archivable_id":    "env-001",
			"scope":            "database",
			"trigger":          "manual",
			"status":           "pending",
			"vector_backup_id": "bk-002",
			"search_replace":   nil,
			"drop_tables":      false,
			"disable_foreign_keys": false,
			"error_message":    nil,
			"duration_ms":      nil,
			"started_at":       nil,
			"completed_at":     nil,
			"created_at":       "2025-01-16T12:00:00+00:00",
			"updated_at":       "2025-01-16T12:00:00+00:00",
		},
	},
	"meta": map[string]any{
		"current_page": 1,
		"last_page":    1,
		"total":        2,
	},
	"message":     "Restores retrieved successfully",
	"http_status": 200,
}

var restoreShowResponse = map[string]any{
	"data": map[string]any{
		"id":               "rst-001",
		"archivable_type":  "vector_site",
		"archivable_id":    "site-001",
		"scope":            "full",
		"trigger":          "manual",
		"status":           "completed",
		"vector_backup_id": "bk-001",
		"search_replace": []map[string]any{
			{"from": "example.org", "to": "example.com"},
		},
		"drop_tables":      false,
		"disable_foreign_keys": false,
		"error_message":    nil,
		"duration_ms":      float64(45200),
		"started_at":       "2025-01-15T12:00:00+00:00",
		"completed_at":     "2025-01-15T12:05:00+00:00",
		"created_at":       "2025-01-15T12:00:00+00:00",
		"updated_at":       "2025-01-15T12:05:00+00:00",
	},
	"message":     "Restore retrieved successfully",
	"http_status": 200,
}

var restoreCreateResponse = map[string]any{
	"data": map[string]any{
		"id":               "rst-003",
		"archivable_type":  "vector_site",
		"archivable_id":    "site-001",
		"scope":            "full",
		"trigger":          "manual",
		"status":           "pending",
		"vector_backup_id": "bk-005",
		"search_replace":   nil,
		"drop_tables":      false,
		"disable_foreign_keys": false,
		"error_message":    nil,
		"duration_ms":      nil,
		"started_at":       nil,
		"completed_at":     nil,
		"created_at":       "2025-01-20T12:00:00+00:00",
		"updated_at":       "2025-01-20T12:00:00+00:00",
	},
	"message":     "Restore initiated successfully",
	"http_status": 202,
}

func newRestoreTestServer(validToken string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+validToken {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Unauthenticated.",
				"http_status": 401,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")

		path := r.URL.Path
		method := r.Method

		switch {
		case method == "GET" && path == "/api/v1/vector/restores":
			_ = json.NewEncoder(w).Encode(restoreListResponse)

		case method == "GET" && path == "/api/v1/vector/restores/rst-001":
			_ = json.NewEncoder(w).Encode(restoreShowResponse)

		case method == "POST" && path == "/api/v1/vector/restores":
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(restoreCreateResponse)

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Not Found",
				"http_status": 404,
			})
		}
	}))
}

func buildRestoreCmd(baseURL, token string, format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	root := &cobra.Command{
		Use: "vector",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			client := api.NewClient(baseURL, token, "test-agent")
			app := appctx.NewApp(
				config.DefaultConfig(),
				client,
				"",
			)
			app.Output = output.NewWriter(stdout, format)
			cmd.SetContext(appctx.WithApp(cmd.Context(), app))
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(NewRestoreCmd())

	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

func buildRestoreCmdNoAuth(format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	root := &cobra.Command{
		Use: "vector",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			client := api.NewClient("http://localhost", "", "test-agent")
			app := appctx.NewApp(
				config.DefaultConfig(),
				client,
				"",
			)
			app.Output = output.NewWriter(stdout, format)
			cmd.SetContext(appctx.WithApp(cmd.Context(), app))
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(NewRestoreCmd())

	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

// --- Restore List Tests ---

func TestRestoreListCmd_TableOutput(t *testing.T) {
	ts := newRestoreTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildRestoreCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"restore", "list"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "rst-001")
	assert.Contains(t, out, "Site")
	assert.Contains(t, out, "bk-001")
	assert.Contains(t, out, "full")
	assert.Contains(t, out, "completed")
	assert.Contains(t, out, "rst-002")
	assert.Contains(t, out, "Environment")
	assert.Contains(t, out, "bk-002")
	assert.Contains(t, out, "pending")
}

func TestRestoreListCmd_JSONOutput(t *testing.T) {
	ts := newRestoreTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildRestoreCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"restore", "list"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Len(t, result, 2)
	assert.Equal(t, "rst-001", result[0]["id"])
}

func TestRestoreListCmd_Pagination(t *testing.T) {
	var receivedPage, receivedPerPage string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPage = r.URL.Query().Get("page")
		receivedPerPage = r.URL.Query().Get("per_page")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(restoreListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildRestoreCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"restore", "list", "--page", "3", "--per-page", "25"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "3", receivedPage)
	assert.Equal(t, "25", receivedPerPage)
}

func TestRestoreListCmd_FilterFlags(t *testing.T) {
	var receivedSiteID, receivedEnvID, receivedType, receivedBackupID string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSiteID = r.URL.Query().Get("site_id")
		receivedEnvID = r.URL.Query().Get("environment_id")
		receivedType = r.URL.Query().Get("type")
		receivedBackupID = r.URL.Query().Get("backup_id")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(restoreListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildRestoreCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"restore", "list", "--site-id", "site-001", "--environment-id", "env-001", "--type", "site", "--backup-id", "bk-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "site-001", receivedSiteID)
	assert.Equal(t, "env-001", receivedEnvID)
	assert.Equal(t, "site", receivedType)
	assert.Equal(t, "bk-001", receivedBackupID)
}

func TestRestoreListCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(restoreListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildRestoreCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"restore", "list"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "GET", receivedMethod)
	assert.Equal(t, "/api/v1/vector/restores", receivedPath)
}

func TestRestoreListCmd_AuthError(t *testing.T) {
	ts := newRestoreTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildRestoreCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"restore", "list"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestRestoreListCmd_NoAuth(t *testing.T) {
	cmd, _, _ := buildRestoreCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"restore", "list"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

// --- Restore Show Tests ---

func TestRestoreShowCmd_TableOutput(t *testing.T) {
	ts := newRestoreTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildRestoreCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"restore", "show", "rst-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "rst-001")
	assert.Contains(t, out, "Site")
	assert.Contains(t, out, "site-001")
	assert.Contains(t, out, "bk-001")
	assert.Contains(t, out, "full")
	assert.Contains(t, out, "manual")
	assert.Contains(t, out, "completed")
	assert.Contains(t, out, "45200")
	assert.Contains(t, out, "2025-01-15T12:00:00+00:00")
	assert.Contains(t, out, "2025-01-15T12:05:00+00:00")
}

func TestRestoreShowCmd_JSONOutput(t *testing.T) {
	ts := newRestoreTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildRestoreCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"restore", "show", "rst-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "rst-001", result["id"])
	assert.Equal(t, "vector_site", result["archivable_type"])
}

func TestRestoreShowCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(restoreShowResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildRestoreCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"restore", "show", "rst-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "GET", receivedMethod)
	assert.Equal(t, "/api/v1/vector/restores/rst-001", receivedPath)
}

func TestRestoreShowCmd_MissingArg(t *testing.T) {
	ts := newRestoreTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildRestoreCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"restore", "show"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

// --- Restore Create Tests ---

func TestRestoreCreateCmd_TableOutput(t *testing.T) {
	ts := newRestoreTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildRestoreCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"restore", "create", "bk-005"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "Restore initiated. Use 'vector restore show rst-003' to check progress.")
	assert.Contains(t, out, "rst-003")
	assert.Contains(t, out, "Site")
	assert.Contains(t, out, "bk-005")
	assert.Contains(t, out, "pending")
}

func TestRestoreCreateCmd_JSONOutput(t *testing.T) {
	ts := newRestoreTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildRestoreCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"restore", "create", "bk-005"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "rst-003", result["id"])
	assert.Equal(t, "pending", result["status"])
}

func TestRestoreCreateCmd_RequestBody(t *testing.T) {
	var receivedBody map[string]any
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(restoreCreateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildRestoreCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"restore", "create", "bk-005"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/api/v1/vector/restores", receivedPath)
	assert.Equal(t, "bk-005", receivedBody["backup_id"])
}

func TestRestoreCreateCmd_WithFlags(t *testing.T) {
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(restoreCreateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildRestoreCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"restore", "create", "bk-005", "--drop-tables", "--disable-foreign-keys", "--search-replace-from", "example.org", "--search-replace-to", "example.com"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "bk-005", receivedBody["backup_id"])
	assert.Equal(t, true, receivedBody["drop_tables"])
	assert.Equal(t, true, receivedBody["disable_foreign_keys"])

	sr, ok := receivedBody["search_replace"].([]any)
	require.True(t, ok)
	require.Len(t, sr, 1)
	pair, ok := sr[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "example.org", pair["from"])
	assert.Equal(t, "example.com", pair["to"])
}

func TestRestoreCreateCmd_MissingArg(t *testing.T) {
	ts := newRestoreTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildRestoreCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"restore", "create"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

// --- Help Tests ---

func TestRestoreCmd_Help(t *testing.T) {
	cmd := NewRestoreCmd()

	stdout := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "list")
	assert.Contains(t, out, "show")
	assert.Contains(t, out, "create")
	assert.Contains(t, out, "restores")
}
