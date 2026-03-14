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

var accountAPIKeyListResponse = map[string]any{
	"data": []map[string]any{
		{
			"id":           "token-001",
			"name":         "Production API Key",
			"abilities":    []string{"site:read", "site:write"},
			"last_used_at": "2025-01-15T12:00:00+00:00",
			"expires_at":   "2025-12-31T23:59:59+00:00",
			"created_at":   "2025-01-01T00:00:00+00:00",
		},
	},
	"meta": map[string]any{
		"current_page": 1,
		"last_page":    1,
		"total":        1,
	},
	"message":     "API keys retrieved successfully",
	"http_status": 200,
}

var accountAPIKeyCreateResponse = map[string]any{
	"data": map[string]any{
		"id":         "token-002",
		"name":       "New API Key",
		"token":      "1|abc123def456789",
		"abilities":  []string{"*"},
		"expires_at": nil,
		"created_at": "2025-01-15T12:00:00+00:00",
	},
	"message":     "API key created successfully",
	"http_status": 201,
}

var accountAPIKeyDeleteResponse = map[string]any{
	"data": map[string]any{
		"id":           "token-001",
		"name":         "Production API Key",
		"abilities":    []string{"site:read", "site:write"},
		"last_used_at": "2025-01-15T12:00:00+00:00",
		"expires_at":   nil,
		"created_at":   "2025-01-01T00:00:00+00:00",
	},
	"message":     "API key deleted successfully",
	"http_status": 200,
}

func newAccountAPIKeyTestServer(validToken string) *httptest.Server {
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
		case method == "GET" && path == "/api/v1/vector/api-keys":
			_ = json.NewEncoder(w).Encode(accountAPIKeyListResponse)

		case method == "POST" && path == "/api/v1/vector/api-keys":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(accountAPIKeyCreateResponse)

		case method == "DELETE" && path == "/api/v1/vector/api-keys/token-001":
			_ = json.NewEncoder(w).Encode(accountAPIKeyDeleteResponse)

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Not Found",
				"http_status": 404,
			})
		}
	}))
}

// --- API Key List Tests ---

func TestAccountAPIKeyListCmd_TableOutput(t *testing.T) {
	ts := newAccountAPIKeyTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "api-key", "list"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "token-001")
	assert.Contains(t, out, "Production API Key")
	assert.Contains(t, out, "site:read, site:write")
	assert.Contains(t, out, "2025-01-15T12:00:00+00:00")
	assert.Contains(t, out, "2025-12-31T23:59:59+00:00")
}

func TestAccountAPIKeyListCmd_JSONOutput(t *testing.T) {
	ts := newAccountAPIKeyTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"account", "api-key", "list"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Len(t, result, 1)
	assert.Equal(t, "token-001", result[0]["id"])
}

func TestAccountAPIKeyListCmd_Pagination(t *testing.T) {
	var receivedPage, receivedPerPage string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPage = r.URL.Query().Get("page")
		receivedPerPage = r.URL.Query().Get("per_page")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(accountAPIKeyListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "api-key", "list", "--page", "2", "--per-page", "10"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "2", receivedPage)
	assert.Equal(t, "10", receivedPerPage)
}

func TestAccountAPIKeyListCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(accountAPIKeyListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "api-key", "list"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "GET", receivedMethod)
	assert.Equal(t, "/api/v1/vector/api-keys", receivedPath)
}

func TestAccountAPIKeyListCmd_AuthError(t *testing.T) {
	ts := newAccountAPIKeyTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"account", "api-key", "list"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestAccountAPIKeyListCmd_NoAuth(t *testing.T) {
	cmd, _, _ := buildAccountCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"account", "api-key", "list"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

// --- API Key Create Tests ---

func TestAccountAPIKeyCreateCmd_TableOutput(t *testing.T) {
	ts := newAccountAPIKeyTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "api-key", "create", "--name", "New API Key"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "token-002")
	assert.Contains(t, out, "New API Key")
	assert.Contains(t, out, "1|abc123def456789")
	assert.Contains(t, out, "Save this token")
}

func TestAccountAPIKeyCreateCmd_JSONOutput(t *testing.T) {
	ts := newAccountAPIKeyTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"account", "api-key", "create", "--name", "New API Key"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "token-002", result["id"])
	assert.Equal(t, "1|abc123def456789", result["token"])
}

func TestAccountAPIKeyCreateCmd_RequestBody(t *testing.T) {
	var receivedBody map[string]any
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(accountAPIKeyCreateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "api-key", "create", "--name", "My Key"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/api/v1/vector/api-keys", receivedPath)
	assert.Equal(t, "My Key", receivedBody["name"])
}

func TestAccountAPIKeyCreateCmd_WithAbilities(t *testing.T) {
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(accountAPIKeyCreateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "api-key", "create",
		"--name", "My Key",
		"--abilities", "site:read,site:write",
	})

	err := cmd.Execute()
	require.NoError(t, err)

	abilities, ok := receivedBody["abilities"].([]any)
	require.True(t, ok)
	assert.Equal(t, []any{"site:read", "site:write"}, abilities)
}

func TestAccountAPIKeyCreateCmd_WithExpiresAt(t *testing.T) {
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(accountAPIKeyCreateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "api-key", "create",
		"--name", "My Key",
		"--expires-at", "2025-12-31T23:59:59+00:00",
	})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "2025-12-31T23:59:59+00:00", receivedBody["expires_at"])
}

func TestAccountAPIKeyCreateCmd_MissingRequiredFlags(t *testing.T) {
	ts := newAccountAPIKeyTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "api-key", "create"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

// --- API Key Delete Tests ---

func TestAccountAPIKeyDeleteCmd_TableOutput(t *testing.T) {
	ts := newAccountAPIKeyTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "api-key", "delete", "token-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "API key deleted successfully")
}

func TestAccountAPIKeyDeleteCmd_JSONOutput(t *testing.T) {
	ts := newAccountAPIKeyTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"account", "api-key", "delete", "token-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "token-001", result["id"])
}

func TestAccountAPIKeyDeleteCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(accountAPIKeyDeleteResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "api-key", "delete", "token-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "DELETE", receivedMethod)
	assert.Equal(t, "/api/v1/vector/api-keys/token-001", receivedPath)
}

func TestAccountAPIKeyDeleteCmd_MissingArg(t *testing.T) {
	ts := newAccountAPIKeyTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "api-key", "delete"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

// --- Help Tests ---

func TestAccountAPIKeyCmd_Help(t *testing.T) {
	cmd := NewAccountCmd()

	stdout := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetArgs([]string{"api-key", "--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "list")
	assert.Contains(t, out, "create")
	assert.Contains(t, out, "delete")
	assert.Contains(t, out, "programmatic access")
}
