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

var accountSSHKeyListResponse = map[string]any{
	"data": []map[string]any{
		{
			"id":                  "key-001",
			"account_id":         1,
			"name":               "deploy key",
			"fingerprint":        "SHA256:abc123def456",
			"public_key_preview": "ssh-rsa AAAAB3...user@host",
			"is_account_default": true,
			"created_at":         "2025-01-15T12:00:00+00:00",
			"updated_at":         "2025-01-15T12:00:00+00:00",
		},
	},
	"meta": map[string]any{
		"current_page": 1,
		"last_page":    1,
		"total":        1,
	},
	"message":     "SSH keys retrieved successfully",
	"http_status": 200,
}

var accountSSHKeyShowResponse = map[string]any{
	"data": map[string]any{
		"id":                  "key-001",
		"account_id":         1,
		"name":               "deploy key",
		"fingerprint":        "SHA256:abc123def456",
		"public_key_preview": "ssh-rsa AAAAB3...user@host",
		"is_account_default": true,
		"created_at":         "2025-01-15T12:00:00+00:00",
		"updated_at":         "2025-01-15T12:00:00+00:00",
	},
	"message":     "SSH key retrieved successfully",
	"http_status": 200,
}

var accountSSHKeyCreateResponse = map[string]any{
	"data": map[string]any{
		"id":                  "key-002",
		"account_id":         1,
		"name":               "new key",
		"fingerprint":        "SHA256:xyz789",
		"public_key_preview": "ssh-rsa BBBBB3...user@host",
		"is_account_default": false,
		"created_at":         "2025-01-15T12:00:00+00:00",
		"updated_at":         "2025-01-15T12:00:00+00:00",
	},
	"message":     "SSH key created successfully",
	"http_status": 201,
}

var accountSSHKeyDeleteResponse = map[string]any{
	"data": map[string]any{
		"id":                  "key-001",
		"account_id":         1,
		"name":               "deploy key",
		"fingerprint":        "SHA256:abc123def456",
		"public_key_preview": "ssh-rsa AAAAB3...user@host",
		"is_account_default": true,
	},
	"message":     "SSH key deleted successfully",
	"http_status": 200,
}

func newAccountSSHKeyTestServer(validToken string) *httptest.Server {
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
		case method == "GET" && path == "/api/v1/vector/ssh-keys":
			_ = json.NewEncoder(w).Encode(accountSSHKeyListResponse)

		case method == "GET" && path == "/api/v1/vector/ssh-keys/key-001":
			_ = json.NewEncoder(w).Encode(accountSSHKeyShowResponse)

		case method == "POST" && path == "/api/v1/vector/ssh-keys":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(accountSSHKeyCreateResponse)

		case method == "DELETE" && path == "/api/v1/vector/ssh-keys/key-001":
			_ = json.NewEncoder(w).Encode(accountSSHKeyDeleteResponse)

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Not Found",
				"http_status": 404,
			})
		}
	}))
}

// --- SSH Key List Tests ---

func TestAccountSSHKeyListCmd_TableOutput(t *testing.T) {
	ts := newAccountSSHKeyTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "ssh-key", "list"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "key-001")
	assert.Contains(t, out, "deploy key")
	assert.Contains(t, out, "SHA256:abc123def456")
	assert.Contains(t, out, "2025-01-15T12:00:00+00:00")
}

func TestAccountSSHKeyListCmd_JSONOutput(t *testing.T) {
	ts := newAccountSSHKeyTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"account", "ssh-key", "list"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Len(t, result, 1)
	assert.Equal(t, "key-001", result[0]["id"])
}

func TestAccountSSHKeyListCmd_Pagination(t *testing.T) {
	var receivedPage, receivedPerPage string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPage = r.URL.Query().Get("page")
		receivedPerPage = r.URL.Query().Get("per_page")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(accountSSHKeyListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "ssh-key", "list", "--page", "2", "--per-page", "10"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "2", receivedPage)
	assert.Equal(t, "10", receivedPerPage)
}

func TestAccountSSHKeyListCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(accountSSHKeyListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "ssh-key", "list"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "GET", receivedMethod)
	assert.Equal(t, "/api/v1/vector/ssh-keys", receivedPath)
}

func TestAccountSSHKeyListCmd_AuthError(t *testing.T) {
	ts := newAccountSSHKeyTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"account", "ssh-key", "list"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestAccountSSHKeyListCmd_NoAuth(t *testing.T) {
	cmd, _, _ := buildAccountCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"account", "ssh-key", "list"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

// --- SSH Key Show Tests ---

func TestAccountSSHKeyShowCmd_TableOutput(t *testing.T) {
	ts := newAccountSSHKeyTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "ssh-key", "show", "key-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "key-001")
	assert.Contains(t, out, "deploy key")
	assert.Contains(t, out, "SHA256:abc123def456")
	assert.Contains(t, out, "ssh-rsa AAAAB3...user@host")
	assert.Contains(t, out, "Yes")
	assert.Contains(t, out, "2025-01-15T12:00:00+00:00")
}

func TestAccountSSHKeyShowCmd_JSONOutput(t *testing.T) {
	ts := newAccountSSHKeyTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"account", "ssh-key", "show", "key-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "key-001", result["id"])
	assert.Equal(t, "deploy key", result["name"])
}

func TestAccountSSHKeyShowCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(accountSSHKeyShowResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "ssh-key", "show", "key-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "GET", receivedMethod)
	assert.Equal(t, "/api/v1/vector/ssh-keys/key-001", receivedPath)
}

func TestAccountSSHKeyShowCmd_MissingArg(t *testing.T) {
	ts := newAccountSSHKeyTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "ssh-key", "show"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

// --- SSH Key Create Tests ---

func TestAccountSSHKeyCreateCmd_TableOutput(t *testing.T) {
	ts := newAccountSSHKeyTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "ssh-key", "create",
		"--name", "new key",
		"--public-key", "ssh-rsa BBBBB3...",
	})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "key-002")
	assert.Contains(t, out, "new key")
	assert.Contains(t, out, "SHA256:xyz789")
}

func TestAccountSSHKeyCreateCmd_JSONOutput(t *testing.T) {
	ts := newAccountSSHKeyTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"account", "ssh-key", "create",
		"--name", "new key",
		"--public-key", "ssh-rsa BBBBB3...",
	})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "key-002", result["id"])
}

func TestAccountSSHKeyCreateCmd_RequestBody(t *testing.T) {
	var receivedBody map[string]any
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(accountSSHKeyCreateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "ssh-key", "create",
		"--name", "my key",
		"--public-key", "ssh-rsa AAAAB3NzaC1yc2EA...",
	})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/api/v1/vector/ssh-keys", receivedPath)
	assert.Equal(t, "my key", receivedBody["name"])
	assert.Equal(t, "ssh-rsa AAAAB3NzaC1yc2EA...", receivedBody["public_key"])
}

func TestAccountSSHKeyCreateCmd_MissingRequiredFlags(t *testing.T) {
	ts := newAccountSSHKeyTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "ssh-key", "create"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

// --- SSH Key Delete Tests ---

func TestAccountSSHKeyDeleteCmd_TableOutput(t *testing.T) {
	ts := newAccountSSHKeyTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "ssh-key", "delete", "key-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "SSH key deleted successfully")
}

func TestAccountSSHKeyDeleteCmd_JSONOutput(t *testing.T) {
	ts := newAccountSSHKeyTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"account", "ssh-key", "delete", "key-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "key-001", result["id"])
}

func TestAccountSSHKeyDeleteCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(accountSSHKeyDeleteResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "ssh-key", "delete", "key-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "DELETE", receivedMethod)
	assert.Equal(t, "/api/v1/vector/ssh-keys/key-001", receivedPath)
}

func TestAccountSSHKeyDeleteCmd_MissingArg(t *testing.T) {
	ts := newAccountSSHKeyTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "ssh-key", "delete"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

// --- SSH Key Help Tests ---

func TestAccountSSHKeyCmd_Help(t *testing.T) {
	cmd := NewAccountCmd()

	stdout := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetArgs([]string{"ssh-key", "--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "list")
	assert.Contains(t, out, "show")
	assert.Contains(t, out, "create")
	assert.Contains(t, out, "delete")
	assert.Contains(t, out, "account-level SSH keys")
}
