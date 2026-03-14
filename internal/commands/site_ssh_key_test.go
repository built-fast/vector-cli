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

var sshKeyListResponse = map[string]any{
	"data": []map[string]any{
		{
			"id":                 "key-001",
			"account_id":        1,
			"vector_site_id":    "site-001",
			"name":              "developer key",
			"fingerprint":       "SHA256:abc123def456",
			"public_key_preview": "ssh-rsa AAAAB3...user@host",
			"is_account_default": false,
			"created_at":        "2025-01-15T12:00:00+00:00",
			"updated_at":        "2025-01-15T12:00:00+00:00",
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

var sshKeyAddResponse = map[string]any{
	"data": map[string]any{
		"id":                 "key-002",
		"account_id":        1,
		"vector_site_id":    "site-001",
		"name":              "new key",
		"fingerprint":       "SHA256:xyz789",
		"public_key_preview": "ssh-rsa BBBBB3...user@host",
		"is_account_default": false,
		"created_at":        "2025-01-15T12:00:00+00:00",
		"updated_at":        "2025-01-15T12:00:00+00:00",
	},
	"message":     "SSH key added to site successfully",
	"http_status": 201,
}

var sshKeyRemoveResponse = map[string]any{
	"data": map[string]any{
		"id":                 "key-001",
		"account_id":        1,
		"vector_site_id":    "site-001",
		"name":              "developer key",
		"fingerprint":       "SHA256:abc123def456",
		"public_key_preview": "ssh-rsa AAAAB3...user@host",
		"is_account_default": false,
	},
	"message":     "SSH key removed from site successfully",
	"http_status": 200,
}

func newSSHKeyTestServer(validToken string) *httptest.Server {
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
		case method == "GET" && path == "/api/v1/vector/sites/site-001/ssh-keys":
			_ = json.NewEncoder(w).Encode(sshKeyListResponse)

		case method == "POST" && path == "/api/v1/vector/sites/site-001/ssh-keys":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(sshKeyAddResponse)

		case method == "DELETE" && path == "/api/v1/vector/sites/site-001/ssh-keys/key-001":
			_ = json.NewEncoder(w).Encode(sshKeyRemoveResponse)

		case method == "DELETE" && path == "/api/v1/vector/sites/site-001/ssh-keys/nonexistent":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "SSH key not found on this site",
				"http_status": 404,
			})

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

func TestSSHKeyListCmd_TableOutput(t *testing.T) {
	ts := newSSHKeyTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "ssh-key", "list", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "key-001")
	assert.Contains(t, out, "developer key")
	assert.Contains(t, out, "SHA256:abc123def456")
	assert.Contains(t, out, "No")
}

func TestSSHKeyListCmd_JSONOutput(t *testing.T) {
	ts := newSSHKeyTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"site", "ssh-key", "list", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Len(t, result, 1)
	assert.Equal(t, "key-001", result[0]["id"])
}

func TestSSHKeyListCmd_Pagination(t *testing.T) {
	var receivedPage, receivedPerPage string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPage = r.URL.Query().Get("page")
		receivedPerPage = r.URL.Query().Get("per_page")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sshKeyListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "ssh-key", "list", "site-001", "--page", "3", "--per-page", "25"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "3", receivedPage)
	assert.Equal(t, "25", receivedPerPage)
}

func TestSSHKeyListCmd_AuthError(t *testing.T) {
	ts := newSSHKeyTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildSiteCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"site", "ssh-key", "list", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestSSHKeyListCmd_NoAuth(t *testing.T) {
	cmd, _, _ := buildSiteCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"site", "ssh-key", "list", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestSSHKeyListCmd_HTTPPath(t *testing.T) {
	var receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sshKeyListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "ssh-key", "list", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "/api/v1/vector/sites/site-001/ssh-keys", receivedPath)
}

// --- SSH Key Add Tests ---

func TestSSHKeyAddCmd_TableOutput(t *testing.T) {
	ts := newSSHKeyTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "ssh-key", "add", "site-001",
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

func TestSSHKeyAddCmd_JSONOutput(t *testing.T) {
	ts := newSSHKeyTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"site", "ssh-key", "add", "site-001",
		"--name", "new key",
		"--public-key", "ssh-rsa BBBBB3...",
	})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "key-002", result["id"])
}

func TestSSHKeyAddCmd_RequestBody(t *testing.T) {
	var receivedBody map[string]any
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(sshKeyAddResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "ssh-key", "add", "site-001",
		"--name", "my key",
		"--public-key", "ssh-rsa AAAAB3NzaC1yc2EA...",
	})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/ssh-keys", receivedPath)
	assert.Equal(t, "my key", receivedBody["name"])
	assert.Equal(t, "ssh-rsa AAAAB3NzaC1yc2EA...", receivedBody["public_key"])
}

func TestSSHKeyAddCmd_MissingRequiredFlags(t *testing.T) {
	ts := newSSHKeyTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "ssh-key", "add", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

// --- SSH Key Remove Tests ---

func TestSSHKeyRemoveCmd_TableOutput(t *testing.T) {
	ts := newSSHKeyTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "ssh-key", "remove", "site-001", "key-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "SSH key removed successfully")
}

func TestSSHKeyRemoveCmd_JSONOutput(t *testing.T) {
	ts := newSSHKeyTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"site", "ssh-key", "remove", "site-001", "key-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "key-001", result["id"])
}

func TestSSHKeyRemoveCmd_NotFound(t *testing.T) {
	ts := newSSHKeyTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "ssh-key", "remove", "site-001", "nonexistent"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 4, apiErr.ExitCode)
}

func TestSSHKeyRemoveCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sshKeyRemoveResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "ssh-key", "remove", "site-001", "key-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "DELETE", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/ssh-keys/key-001", receivedPath)
}

func TestSSHKeyRemoveCmd_MissingArgs(t *testing.T) {
	ts := newSSHKeyTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "ssh-key", "remove", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 2 arg(s)")
}

// --- SSH Key Help Tests ---

func TestSSHKeyCmd_Help(t *testing.T) {
	cmd := NewSiteCmd()

	stdout := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetArgs([]string{"ssh-key", "--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "list")
	assert.Contains(t, out, "add")
	assert.Contains(t, out, "remove")
}
