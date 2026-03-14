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

var importSessionCreateResponse = map[string]any{
	"data": map[string]any{
		"id":                "imp-001",
		"vector_site_id":    "site-001",
		"status":            "pending",
		"scope":             nil,
		"filename":          "backup.sql.gz",
		"content_length":    float64(52428800),
		"duration_ms":       nil,
		"error_message":     nil,
		"created_at":        "2025-01-15T12:00:00+00:00",
		"uploaded_at":       nil,
		"started_at":        nil,
		"completed_at":      nil,
		"upload_url":        "https://s3.amazonaws.com/bucket/imports/imp-001.sql.gz?X-Amz-Expires=3600",
		"upload_expires_at": "2025-01-15T13:00:00+00:00",
	},
	"message":     "Import session created successfully",
	"http_status": 201,
}

var importSessionRunResponse = map[string]any{
	"data": map[string]any{
		"id":             "imp-001",
		"vector_site_id": "site-001",
		"status":         "uploaded",
		"filename":       "backup.sql.gz",
		"duration_ms":    nil,
		"error_message":  nil,
		"created_at":     "2025-01-15T12:00:00+00:00",
		"uploaded_at":    "2025-01-15T12:00:01+00:00",
		"started_at":     nil,
		"completed_at":   nil,
	},
	"message":     "Archive import started",
	"http_status": 202,
}

var importSessionStatusResponse = map[string]any{
	"data": map[string]any{
		"id":             "imp-001",
		"vector_site_id": "site-001",
		"status":         "completed",
		"filename":       "backup.sql.gz",
		"duration_ms":    float64(30000),
		"error_message":  nil,
		"created_at":     "2025-01-15T12:00:00+00:00",
		"uploaded_at":    "2025-01-15T12:00:01+00:00",
		"started_at":     "2025-01-15T12:00:02+00:00",
		"completed_at":   "2025-01-15T12:00:32+00:00",
	},
	"message":     "Import retrieved successfully",
	"http_status": 200,
}

func newImportSessionTestServer(validToken string) *httptest.Server {
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
		case method == "POST" && path == "/api/v1/vector/sites/site-001/imports":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(importSessionCreateResponse)

		case method == "POST" && path == "/api/v1/vector/sites/site-001/imports/imp-001/run":
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(importSessionRunResponse)

		case method == "GET" && path == "/api/v1/vector/sites/site-001/imports/imp-001":
			_ = json.NewEncoder(w).Encode(importSessionStatusResponse)

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Not Found",
				"http_status": 404,
			})
		}
	}))
}

func buildDbCmd(baseURL, token string, format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	root := &cobra.Command{Use: "vector"}
	root.AddCommand(NewDbCmd())

	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		client := api.NewClient(baseURL, token, "")
		app := appctx.NewApp(&config.Config{}, &config.Credentials{}, client, format, "test")
		cmd.SetContext(appctx.WithApp(cmd.Context(), app))
		return nil
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

func buildDbCmdNoAuth(format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	root := &cobra.Command{Use: "vector"}
	root.AddCommand(NewDbCmd())

	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		client := api.NewClient("", "", "")
		app := appctx.NewApp(&config.Config{}, &config.Credentials{}, client, format, "")
		cmd.SetContext(appctx.WithApp(cmd.Context(), app))
		return nil
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

// --- Import Session Create Tests ---

func TestDbImportSessionCreateCmd_TableOutput(t *testing.T) {
	ts := newImportSessionTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildDbCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"db", "import-session", "create", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "imp-001")
	assert.Contains(t, out, "pending")
	assert.Contains(t, out, "https://s3.amazonaws.com/bucket/imports/imp-001.sql.gz?X-Amz-Expires=3600")
	assert.Contains(t, out, "2025-01-15T13:00:00+00:00")
	assert.Contains(t, out, "Upload your SQL file to the URL above, then run: vector db import-session run site-001 imp-001")
}

func TestDbImportSessionCreateCmd_JSONOutput(t *testing.T) {
	ts := newImportSessionTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildDbCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"db", "import-session", "create", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "imp-001", result["id"])
	assert.Equal(t, "pending", result["status"])
}

func TestDbImportSessionCreateCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(importSessionCreateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildDbCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"db", "import-session", "create", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/imports", receivedPath)
	assert.Equal(t, "database", receivedBody["scope"])
}

func TestDbImportSessionCreateCmd_WithOptions(t *testing.T) {
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(importSessionCreateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildDbCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{
		"db", "import-session", "create", "site-001",
		"--filename", "dump.sql.gz",
		"--content-length", "12345",
		"--drop-tables",
		"--disable-foreign-keys",
		"--search-replace-from", "example.org",
		"--search-replace-to", "example.com",
	})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "database", receivedBody["scope"])
	assert.Equal(t, "dump.sql.gz", receivedBody["filename"])
	assert.Equal(t, float64(12345), receivedBody["content_length"])

	options, ok := receivedBody["options"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, options["drop_tables"])
	assert.Equal(t, true, options["disable_foreign_keys"])

	sr, ok := options["search_replace"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "example.org", sr["from"])
	assert.Equal(t, "example.com", sr["to"])
}

func TestDbImportSessionCreateCmd_MissingArg(t *testing.T) {
	ts := newImportSessionTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildDbCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"db", "import-session", "create"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

func TestDbImportSessionCreateCmd_AuthError(t *testing.T) {
	ts := newImportSessionTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildDbCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"db", "import-session", "create", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestDbImportSessionCreateCmd_NoAuth(t *testing.T) {
	cmd, _, _ := buildDbCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"db", "import-session", "create", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

// --- Import Session Run Tests ---

func TestDbImportSessionRunCmd_TableOutput(t *testing.T) {
	ts := newImportSessionTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildDbCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"db", "import-session", "run", "site-001", "imp-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "imp-001")
	assert.Contains(t, out, "uploaded")
}

func TestDbImportSessionRunCmd_JSONOutput(t *testing.T) {
	ts := newImportSessionTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildDbCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"db", "import-session", "run", "site-001", "imp-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "imp-001", result["id"])
	assert.Equal(t, "uploaded", result["status"])
}

func TestDbImportSessionRunCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(importSessionRunResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildDbCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"db", "import-session", "run", "site-001", "imp-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/imports/imp-001/run", receivedPath)
}

func TestDbImportSessionRunCmd_MissingArgs(t *testing.T) {
	ts := newImportSessionTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildDbCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"db", "import-session", "run", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 2 arg(s)")
}

func TestDbImportSessionRunCmd_AuthError(t *testing.T) {
	ts := newImportSessionTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildDbCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"db", "import-session", "run", "site-001", "imp-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestDbImportSessionRunCmd_NoAuth(t *testing.T) {
	cmd, _, _ := buildDbCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"db", "import-session", "run", "site-001", "imp-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

// --- Import Session Status Tests ---

func TestDbImportSessionStatusCmd_TableOutput(t *testing.T) {
	ts := newImportSessionTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildDbCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"db", "import-session", "status", "site-001", "imp-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "imp-001")
	assert.Contains(t, out, "completed")
	assert.Contains(t, out, "backup.sql.gz")
	assert.Contains(t, out, "30000")
	assert.Contains(t, out, "2025-01-15T12:00:00+00:00")
	assert.Contains(t, out, "2025-01-15T12:00:32+00:00")
}

func TestDbImportSessionStatusCmd_JSONOutput(t *testing.T) {
	ts := newImportSessionTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildDbCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"db", "import-session", "status", "site-001", "imp-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "imp-001", result["id"])
	assert.Equal(t, "completed", result["status"])
	assert.Equal(t, float64(30000), result["duration_ms"])
}

func TestDbImportSessionStatusCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(importSessionStatusResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildDbCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"db", "import-session", "status", "site-001", "imp-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "GET", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/imports/imp-001", receivedPath)
}

func TestDbImportSessionStatusCmd_MissingArgs(t *testing.T) {
	ts := newImportSessionTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildDbCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"db", "import-session", "status", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 2 arg(s)")
}

func TestDbImportSessionStatusCmd_AuthError(t *testing.T) {
	ts := newImportSessionTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildDbCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"db", "import-session", "status", "site-001", "imp-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

// --- Help Tests ---

func TestDbImportSessionCmd_Help(t *testing.T) {
	cmd := NewDbImportSessionCmd()

	stdout := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "create")
	assert.Contains(t, out, "run")
	assert.Contains(t, out, "status")
	assert.Contains(t, out, "Manage database import sessions")
}
