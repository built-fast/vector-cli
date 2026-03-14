package commands

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/output"
)

var backupDownloadCreateResponse = map[string]any{
	"data": map[string]any{
		"id":                  "dl-001",
		"vector_backup_id":    "bk-001",
		"status":              "pending",
		"s3_key":              nil,
		"size_bytes":          nil,
		"duration_ms":         nil,
		"error_message":       nil,
		"download_url":        nil,
		"download_expires_at": nil,
		"started_at":          nil,
		"completed_at":        nil,
		"created_at":          "2025-01-15T12:00:00+00:00",
		"updated_at":          "2025-01-15T12:00:00+00:00",
	},
	"message":     "Backup download initiated",
	"http_status": 202,
}

var backupDownloadStatusCompletedResponse = map[string]any{
	"data": map[string]any{
		"id":                  "dl-001",
		"vector_backup_id":    "bk-001",
		"status":              "completed",
		"s3_key":              "backups/downloads/dl-001.tar.gz",
		"size_bytes":          float64(52428800),
		"duration_ms":         float64(12500),
		"error_message":       nil,
		"download_url":        "https://s3.amazonaws.com/bucket/backups/downloads/dl-001.tar.gz?presigned=abc",
		"download_expires_at": "2025-01-15T18:00:00+00:00",
		"started_at":          "2025-01-15T12:00:00+00:00",
		"completed_at":        "2025-01-15T12:00:12+00:00",
		"created_at":          "2025-01-15T12:00:00+00:00",
		"updated_at":          "2025-01-15T12:00:12+00:00",
	},
	"message":     "Backup download retrieved successfully",
	"http_status": 200,
}

var backupDownloadStatusPendingResponse = map[string]any{
	"data": map[string]any{
		"id":                  "dl-001",
		"vector_backup_id":    "bk-001",
		"status":              "processing",
		"s3_key":              nil,
		"size_bytes":          nil,
		"duration_ms":         nil,
		"error_message":       nil,
		"download_url":        nil,
		"download_expires_at": nil,
		"started_at":          nil,
		"completed_at":        nil,
		"created_at":          "2025-01-15T12:00:00+00:00",
		"updated_at":          "2025-01-15T12:00:00+00:00",
	},
	"message":     "Backup download retrieved successfully",
	"http_status": 200,
}

func newBackupDownloadTestServer(validToken string) *httptest.Server {
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
		case method == "POST" && path == "/api/v1/vector/backups/bk-001/downloads":
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(backupDownloadCreateResponse)

		case method == "GET" && path == "/api/v1/vector/backups/bk-001/downloads/dl-001":
			_ = json.NewEncoder(w).Encode(backupDownloadStatusCompletedResponse)

		case method == "GET" && path == "/api/v1/vector/backups/bk-001/downloads/dl-pending":
			_ = json.NewEncoder(w).Encode(backupDownloadStatusPendingResponse)

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Not Found",
				"http_status": 404,
			})
		}
	}))
}

// --- Download Create Tests ---

func TestBackupDownloadCreateCmd_TableOutput(t *testing.T) {
	ts := newBackupDownloadTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildBackupCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"backup", "download", "create", "bk-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "dl-001")
	assert.Contains(t, out, "pending")
	assert.Contains(t, out, "Check download status with: vector backup download status bk-001 dl-001")
}

func TestBackupDownloadCreateCmd_JSONOutput(t *testing.T) {
	ts := newBackupDownloadTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildBackupCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"backup", "download", "create", "bk-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "dl-001", result["id"])
	assert.Equal(t, "pending", result["status"])
}

func TestBackupDownloadCreateCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(backupDownloadCreateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildBackupCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"backup", "download", "create", "bk-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/api/v1/vector/backups/bk-001/downloads", receivedPath)
}

func TestBackupDownloadCreateCmd_MissingArg(t *testing.T) {
	ts := newBackupDownloadTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildBackupCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"backup", "download", "create"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

func TestBackupDownloadCreateCmd_AuthError(t *testing.T) {
	ts := newBackupDownloadTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildBackupCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"backup", "download", "create", "bk-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestBackupDownloadCreateCmd_NoAuth(t *testing.T) {
	cmd, _, _ := buildBackupCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"backup", "download", "create", "bk-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

// --- Download Status Tests ---

func TestBackupDownloadStatusCmd_CompletedOutput(t *testing.T) {
	ts := newBackupDownloadTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildBackupCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"backup", "download", "status", "bk-001", "dl-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "dl-001")
	assert.Contains(t, out, "completed")
	assert.Contains(t, out, "52428800")
	assert.Contains(t, out, "12500")
	assert.Contains(t, out, "https://s3.amazonaws.com/bucket/backups/downloads/dl-001.tar.gz?presigned=abc")
	assert.Contains(t, out, "2025-01-15T18:00:00+00:00")
	assert.Contains(t, out, "2025-01-15T12:00:00+00:00")
	assert.Contains(t, out, "2025-01-15T12:00:12+00:00")
}

func TestBackupDownloadStatusCmd_PendingOutput(t *testing.T) {
	ts := newBackupDownloadTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildBackupCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"backup", "download", "status", "bk-001", "dl-pending"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "dl-001")
	assert.Contains(t, out, "processing")
	// Download URL should NOT be shown when not completed
	assert.NotContains(t, out, "Download URL")
	assert.NotContains(t, out, "Download Expires")
}

func TestBackupDownloadStatusCmd_JSONOutput(t *testing.T) {
	ts := newBackupDownloadTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildBackupCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"backup", "download", "status", "bk-001", "dl-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "dl-001", result["id"])
	assert.Equal(t, "completed", result["status"])
	assert.Equal(t, float64(52428800), result["size_bytes"])
}

func TestBackupDownloadStatusCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(backupDownloadStatusCompletedResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildBackupCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"backup", "download", "status", "bk-001", "dl-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "GET", receivedMethod)
	assert.Equal(t, "/api/v1/vector/backups/bk-001/downloads/dl-001", receivedPath)
}

func TestBackupDownloadStatusCmd_MissingArgs(t *testing.T) {
	ts := newBackupDownloadTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildBackupCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"backup", "download", "status", "bk-001"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 2 arg(s)")
}

func TestBackupDownloadStatusCmd_AuthError(t *testing.T) {
	ts := newBackupDownloadTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildBackupCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"backup", "download", "status", "bk-001", "dl-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

// --- Help Tests ---

func TestBackupDownloadCmd_Help(t *testing.T) {
	cmd := NewBackupDownloadCmd()

	stdout := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "create")
	assert.Contains(t, out, "status")
	assert.Contains(t, out, "backup download")
}

// --- formatFloat Tests ---

func TestFormatFloat(t *testing.T) {
	assert.Equal(t, "-", formatFloat(0))
	assert.Equal(t, "52428800", formatFloat(52428800))
	assert.Equal(t, "12500", formatFloat(12500))
}
