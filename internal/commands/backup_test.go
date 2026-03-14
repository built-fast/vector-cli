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

var backupListResponse = map[string]any{
	"data": []map[string]any{
		{
			"id":                   "bk-001",
			"archivable_type":      "vector_site",
			"archivable_id":        "site-001",
			"type":                 "manual",
			"scope":                "full",
			"status":               "completed",
			"description":          "Pre-deployment backup",
			"file_snapshot_id":     "abc123",
			"database_snapshot_id": "def456",
			"started_at":          "2025-01-15T12:00:00+00:00",
			"completed_at":        "2025-01-15T12:05:00+00:00",
			"created_at":          "2025-01-15T12:00:00+00:00",
			"updated_at":          "2025-01-15T12:05:00+00:00",
		},
		{
			"id":                   "bk-002",
			"archivable_type":      "vector_environment",
			"archivable_id":        "env-001",
			"type":                 "scheduled",
			"scope":                "database",
			"status":               "pending",
			"description":          nil,
			"file_snapshot_id":     nil,
			"database_snapshot_id": nil,
			"started_at":          nil,
			"completed_at":        nil,
			"created_at":          "2025-01-16T12:00:00+00:00",
			"updated_at":          "2025-01-16T12:00:00+00:00",
		},
	},
	"meta": map[string]any{
		"current_page": 1,
		"last_page":    1,
		"total":        2,
	},
	"message":     "Backups retrieved successfully",
	"http_status": 200,
}

var backupShowResponse = map[string]any{
	"data": map[string]any{
		"id":                   "bk-001",
		"archivable_type":      "vector_site",
		"archivable_id":        "site-001",
		"type":                 "manual",
		"scope":                "full",
		"status":               "completed",
		"description":          "Pre-deployment backup",
		"file_snapshot_id":     "abc123",
		"database_snapshot_id": "def456",
		"started_at":          "2025-01-15T12:00:00+00:00",
		"completed_at":        "2025-01-15T12:05:00+00:00",
		"created_at":          "2025-01-15T12:00:00+00:00",
		"updated_at":          "2025-01-15T12:05:00+00:00",
	},
	"message":     "Backup retrieved successfully",
	"http_status": 200,
}

var backupCreateResponse = map[string]any{
	"data": map[string]any{
		"id":                   "bk-003",
		"archivable_type":      "vector_site",
		"archivable_id":        "site-001",
		"type":                 "manual",
		"scope":                "full",
		"status":               "pending",
		"description":          "Manual backup",
		"file_snapshot_id":     nil,
		"database_snapshot_id": nil,
		"started_at":          nil,
		"completed_at":        nil,
		"created_at":          "2025-01-20T12:00:00+00:00",
		"updated_at":          "2025-01-20T12:00:00+00:00",
	},
	"message":     "Backup initiated successfully",
	"http_status": 202,
}

func newBackupTestServer(validToken string) *httptest.Server {
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
		case method == "GET" && path == "/api/v1/vector/backups":
			_ = json.NewEncoder(w).Encode(backupListResponse)

		case method == "GET" && path == "/api/v1/vector/backups/bk-001":
			_ = json.NewEncoder(w).Encode(backupShowResponse)

		case method == "POST" && path == "/api/v1/vector/backups":
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(backupCreateResponse)

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Not Found",
				"http_status": 404,
			})
		}
	}))
}

func buildBackupCmd(baseURL, token string, format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	root := &cobra.Command{
		Use: "vector",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			client := api.NewClient(baseURL, token, "test-agent")
			app := appctx.NewApp(
				config.DefaultConfig(),
				&config.Credentials{ApiKey: token},
				client,
				format,
				"",
			)
			cmd.SetContext(appctx.WithApp(cmd.Context(), app))
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(NewBackupCmd())

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

func buildBackupCmdNoAuth(format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	root := &cobra.Command{
		Use: "vector",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			client := api.NewClient("http://localhost", "", "test-agent")
			app := appctx.NewApp(
				config.DefaultConfig(),
				&config.Credentials{},
				client,
				format,
				"",
			)
			cmd.SetContext(appctx.WithApp(cmd.Context(), app))
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(NewBackupCmd())

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

// --- Backup List Tests ---

func TestBackupListCmd_TableOutput(t *testing.T) {
	ts := newBackupTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildBackupCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"backup", "list"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "bk-001")
	assert.Contains(t, out, "Site")
	assert.Contains(t, out, "manual")
	assert.Contains(t, out, "full")
	assert.Contains(t, out, "completed")
	assert.Contains(t, out, "Pre-deployment backup")
	assert.Contains(t, out, "bk-002")
	assert.Contains(t, out, "Environment")
	assert.Contains(t, out, "scheduled")
	assert.Contains(t, out, "pending")
}

func TestBackupListCmd_JSONOutput(t *testing.T) {
	ts := newBackupTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildBackupCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"backup", "list"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Len(t, result, 2)
	assert.Equal(t, "bk-001", result[0]["id"])
}

func TestBackupListCmd_Pagination(t *testing.T) {
	var receivedPage, receivedPerPage string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPage = r.URL.Query().Get("page")
		receivedPerPage = r.URL.Query().Get("per_page")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(backupListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildBackupCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"backup", "list", "--page", "3", "--per-page", "25"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "3", receivedPage)
	assert.Equal(t, "25", receivedPerPage)
}

func TestBackupListCmd_FilterFlags(t *testing.T) {
	var receivedSiteID, receivedEnvID, receivedType string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSiteID = r.URL.Query().Get("site_id")
		receivedEnvID = r.URL.Query().Get("environment_id")
		receivedType = r.URL.Query().Get("type")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(backupListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildBackupCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"backup", "list", "--site-id", "site-001", "--environment-id", "env-001", "--type", "site"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "site-001", receivedSiteID)
	assert.Equal(t, "env-001", receivedEnvID)
	assert.Equal(t, "site", receivedType)
}

func TestBackupListCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(backupListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildBackupCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"backup", "list"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "GET", receivedMethod)
	assert.Equal(t, "/api/v1/vector/backups", receivedPath)
}

func TestBackupListCmd_AuthError(t *testing.T) {
	ts := newBackupTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildBackupCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"backup", "list"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestBackupListCmd_NoAuth(t *testing.T) {
	cmd, _, _ := buildBackupCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"backup", "list"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

// --- Backup Show Tests ---

func TestBackupShowCmd_TableOutput(t *testing.T) {
	ts := newBackupTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildBackupCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"backup", "show", "bk-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "bk-001")
	assert.Contains(t, out, "Site")
	assert.Contains(t, out, "site-001")
	assert.Contains(t, out, "manual")
	assert.Contains(t, out, "full")
	assert.Contains(t, out, "completed")
	assert.Contains(t, out, "Pre-deployment backup")
	assert.Contains(t, out, "abc123")
	assert.Contains(t, out, "def456")
	assert.Contains(t, out, "2025-01-15T12:00:00+00:00")
	assert.Contains(t, out, "2025-01-15T12:05:00+00:00")
}

func TestBackupShowCmd_JSONOutput(t *testing.T) {
	ts := newBackupTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildBackupCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"backup", "show", "bk-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "bk-001", result["id"])
	assert.Equal(t, "vector_site", result["archivable_type"])
}

func TestBackupShowCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(backupShowResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildBackupCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"backup", "show", "bk-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "GET", receivedMethod)
	assert.Equal(t, "/api/v1/vector/backups/bk-001", receivedPath)
}

func TestBackupShowCmd_MissingArg(t *testing.T) {
	ts := newBackupTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildBackupCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"backup", "show"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

// --- Backup Create Tests ---

func TestBackupCreateCmd_TableOutput(t *testing.T) {
	ts := newBackupTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildBackupCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"backup", "create", "--site-id", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "Backup created: bk-003 (pending)")
	assert.Contains(t, out, "bk-003")
	assert.Contains(t, out, "Site")
	assert.Contains(t, out, "manual")
	assert.Contains(t, out, "full")
	assert.Contains(t, out, "pending")
}

func TestBackupCreateCmd_JSONOutput(t *testing.T) {
	ts := newBackupTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildBackupCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"backup", "create", "--site-id", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "bk-003", result["id"])
	assert.Equal(t, "pending", result["status"])
}

func TestBackupCreateCmd_RequestBody(t *testing.T) {
	var receivedBody map[string]any
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(backupCreateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildBackupCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"backup", "create", "--site-id", "site-001", "--scope", "database", "--description", "My backup"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/api/v1/vector/backups", receivedPath)
	assert.Equal(t, "site-001", receivedBody["site_id"])
	assert.Equal(t, "manual", receivedBody["type"])
	assert.Equal(t, "database", receivedBody["scope"])
	assert.Equal(t, "My backup", receivedBody["description"])
}

func TestBackupCreateCmd_EnvironmentID(t *testing.T) {
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(backupCreateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildBackupCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"backup", "create", "--environment-id", "env-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "env-001", receivedBody["environment_id"])
	assert.Equal(t, "manual", receivedBody["type"])
	assert.Equal(t, "full", receivedBody["scope"])
	_, hasSiteID := receivedBody["site_id"]
	assert.False(t, hasSiteID)
}

func TestBackupCreateCmd_MissingSiteAndEnv(t *testing.T) {
	ts := newBackupTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildBackupCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"backup", "create"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "either --site-id or --environment-id is required")
}

func TestBackupCreateCmd_DefaultScope(t *testing.T) {
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(backupCreateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildBackupCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"backup", "create", "--site-id", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "full", receivedBody["scope"])
}

// --- Help Tests ---

func TestBackupCmd_Help(t *testing.T) {
	cmd := NewBackupCmd()

	stdout := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "list")
	assert.Contains(t, out, "show")
	assert.Contains(t, out, "create")
	assert.Contains(t, out, "backups")
}

// --- formatArchivableType Tests ---

func TestFormatArchivableType(t *testing.T) {
	assert.Equal(t, "Site", formatArchivableType("vector_site"))
	assert.Equal(t, "Environment", formatArchivableType("vector_environment"))
	assert.Equal(t, "Site", formatArchivableType("site"))
	assert.Equal(t, "-", formatArchivableType(""))
}
