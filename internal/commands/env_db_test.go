package commands

import (
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

var dbPromoteResponse = map[string]any{
	"data": map[string]any{
		"id":                      "prm-001",
		"vector_environment_id":   "env-001",
		"vector_db_export_id":     nil,
		"status":                  "pending",
		"options": map[string]any{
			"drop_tables":         true,
			"disable_foreign_keys": true,
			"search_replace":      nil,
		},
		"duration_ms":    nil,
		"error_message":  nil,
		"created_at":     "2025-01-15T12:00:00+00:00",
		"started_at":     nil,
		"completed_at":   nil,
	},
	"message":     "Database promote initiated",
	"http_status": 202,
}

var dbPromoteStatusResponse = map[string]any{
	"data": map[string]any{
		"id":                    "prm-001",
		"vector_environment_id": "env-001",
		"status":                "completed",
		"duration_ms":           1500,
		"error_message":         nil,
		"created_at":            "2025-01-15T12:00:00+00:00",
		"started_at":            "2025-01-15T12:00:01+00:00",
		"completed_at":          "2025-01-15T12:00:02+00:00",
	},
	"message":     "Database promote retrieved successfully",
	"http_status": 200,
}

func newDBTestServer(validToken string) *httptest.Server {
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
		case method == "POST" && path == "/api/v1/vector/environments/env-001/db/promote":
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(dbPromoteResponse)

		case method == "GET" && path == "/api/v1/vector/environments/env-001/db/promotes/prm-001":
			_ = json.NewEncoder(w).Encode(dbPromoteStatusResponse)

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Not Found",
				"http_status": 404,
			})
		}
	}))
}

// --- DB Promote Tests ---

func TestEnvDBPromoteCmd_TableOutput(t *testing.T) {
	ts := newDBTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "db", "promote", "env-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "prm-001")
	assert.Contains(t, out, "env-001")
	assert.Contains(t, out, "pending")
}

func TestEnvDBPromoteCmd_JSONOutput(t *testing.T) {
	ts := newDBTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"env", "db", "promote", "env-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "prm-001", result["id"])
	assert.Equal(t, "pending", result["status"])
}

func TestEnvDBPromoteCmd_RequestBody(t *testing.T) {
	var receivedBody map[string]any
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(dbPromoteResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "db", "promote", "env-001", "--drop-tables=false", "--disable-foreign-keys=false"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/api/v1/vector/environments/env-001/db/promote", receivedPath)
	assert.Equal(t, false, receivedBody["drop_tables"])
	assert.Equal(t, false, receivedBody["disable_foreign_keys"])
}

func TestEnvDBPromoteCmd_AuthError(t *testing.T) {
	ts := newDBTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildEnvCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"env", "db", "promote", "env-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

// --- DB Promote Status Tests ---

func TestEnvDBPromoteStatusCmd_TableOutput(t *testing.T) {
	ts := newDBTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "db", "promote-status", "env-001", "prm-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "prm-001")
	assert.Contains(t, out, "completed")
	assert.Contains(t, out, "1500ms")
}

func TestEnvDBPromoteStatusCmd_JSONOutput(t *testing.T) {
	ts := newDBTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"env", "db", "promote-status", "env-001", "prm-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "prm-001", result["id"])
	assert.Equal(t, "completed", result["status"])
}

func TestEnvDBPromoteStatusCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(dbPromoteStatusResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "db", "promote-status", "env-001", "prm-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "GET", receivedMethod)
	assert.Equal(t, "/api/v1/vector/environments/env-001/db/promotes/prm-001", receivedPath)
}

func TestEnvDBPromoteStatusCmd_MissingArgs(t *testing.T) {
	ts := newDBTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "db", "promote-status", "env-001"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 2 arg(s)")
}
