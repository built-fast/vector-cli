package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/output"
)

var dbExportCreateResponse = map[string]any{
	"data": map[string]any{
		"id":                  "exp-001",
		"vector_site_id":      "site-001",
		"status":              "pending",
		"format":              "sql",
		"size_bytes":          nil,
		"duration_ms":         nil,
		"error_message":       nil,
		"download_url":        nil,
		"download_expires_at": nil,
		"created_at":          "2025-01-15T12:00:00+00:00",
		"completed_at":        nil,
	},
	"message":     "Database export started",
	"http_status": 202,
}

var dbExportStatusCompletedResponse = map[string]any{
	"data": map[string]any{
		"id":                  "exp-001",
		"vector_site_id":      "site-001",
		"status":              "completed",
		"format":              "sql",
		"size_bytes":          float64(10485760),
		"duration_ms":         float64(5000),
		"error_message":       nil,
		"download_url":        "https://s3.amazonaws.com/bucket/exports/exp-001.sql.gz?presigned=abc",
		"download_expires_at": "2025-01-15T18:00:00+00:00",
		"created_at":          "2025-01-15T12:00:00+00:00",
		"completed_at":        "2025-01-15T12:00:05+00:00",
	},
	"message":     "Database export retrieved successfully",
	"http_status": 200,
}

var dbExportStatusPendingResponse = map[string]any{
	"data": map[string]any{
		"id":                  "exp-001",
		"vector_site_id":      "site-001",
		"status":              "processing",
		"format":              "sql",
		"size_bytes":          nil,
		"duration_ms":         nil,
		"error_message":       nil,
		"download_url":        nil,
		"download_expires_at": nil,
		"created_at":          "2025-01-15T12:00:00+00:00",
		"completed_at":        nil,
	},
	"message":     "Database export retrieved successfully",
	"http_status": 200,
}

func newDBExportTestServer(validToken string) *httptest.Server {
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
		case method == "POST" && path == "/api/v1/vector/sites/site-001/db/export":
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(dbExportCreateResponse)

		case method == "GET" && path == "/api/v1/vector/sites/site-001/db/exports/exp-001":
			_ = json.NewEncoder(w).Encode(dbExportStatusCompletedResponse)

		case method == "GET" && path == "/api/v1/vector/sites/site-001/db/exports/exp-pending":
			_ = json.NewEncoder(w).Encode(dbExportStatusPendingResponse)

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Not Found",
				"http_status": 404,
			})
		}
	}))
}

// --- Export Create Tests ---

func TestDbExportCreateCmd_TableOutput(t *testing.T) {
	ts := newDBExportTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildDBCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"db", "export", "create", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "Export started: exp-001 (pending)")
	assert.Contains(t, out, "Check status with: vector db export status site-001 exp-001")
}

func TestDbExportCreateCmd_JSONOutput(t *testing.T) {
	ts := newDBExportTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildDBCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"db", "export", "create", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "exp-001", result["id"])
	assert.Equal(t, "pending", result["status"])
}

func TestDbExportCreateCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(dbExportCreateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildDBCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"db", "export", "create", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/db/export", receivedPath)
	assert.Equal(t, "sql", receivedBody["format"])
}

func TestDbExportCreateCmd_WithFormat(t *testing.T) {
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(dbExportCreateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildDBCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"db", "export", "create", "site-001", "--format", "csv"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "csv", receivedBody["format"])
}

func TestDbExportCreateCmd_MissingArg(t *testing.T) {
	ts := newDBExportTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildDBCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"db", "export", "create"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

func TestDbExportCreateCmd_AuthError(t *testing.T) {
	ts := newDBExportTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildDBCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"db", "export", "create", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestDbExportCreateCmd_NoAuth(t *testing.T) {
	cmd, _, _ := buildDBCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"db", "export", "create", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

// --- Export Status Tests ---

func TestDbExportStatusCmd_CompletedOutput(t *testing.T) {
	ts := newDBExportTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildDBCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"db", "export", "status", "site-001", "exp-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "exp-001")
	assert.Contains(t, out, "completed")
	assert.Contains(t, out, "sql")
	assert.Contains(t, out, "10485760")
	assert.Contains(t, out, "5000")
	assert.Contains(t, out, "https://s3.amazonaws.com/bucket/exports/exp-001.sql.gz?presigned=abc")
	assert.Contains(t, out, "2025-01-15T18:00:00+00:00")
	assert.Contains(t, out, "2025-01-15T12:00:00+00:00")
	assert.Contains(t, out, "2025-01-15T12:00:05+00:00")
}

func TestDbExportStatusCmd_PendingOutput(t *testing.T) {
	ts := newDBExportTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildDBCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"db", "export", "status", "site-001", "exp-pending"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "exp-001")
	assert.Contains(t, out, "processing")
	assert.NotContains(t, out, "Download URL")
	assert.NotContains(t, out, "Download Expires")
}

func TestDbExportStatusCmd_JSONOutput(t *testing.T) {
	ts := newDBExportTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildDBCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"db", "export", "status", "site-001", "exp-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "exp-001", result["id"])
	assert.Equal(t, "completed", result["status"])
	assert.Equal(t, float64(10485760), result["size_bytes"])
}

func TestDbExportStatusCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(dbExportStatusCompletedResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildDBCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"db", "export", "status", "site-001", "exp-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "GET", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/db/exports/exp-001", receivedPath)
}

func TestDbExportStatusCmd_MissingArgs(t *testing.T) {
	ts := newDBExportTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildDBCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"db", "export", "status", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 2 arg(s)")
}

func TestDbExportStatusCmd_AuthError(t *testing.T) {
	ts := newDBExportTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildDBCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"db", "export", "status", "site-001", "exp-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

// --- Help Tests ---

func TestDbExportCmd_Help(t *testing.T) {
	cmd := NewDBExportCmd()

	stdout := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "create")
	assert.Contains(t, out, "status")
	assert.Contains(t, out, "database export")
}
