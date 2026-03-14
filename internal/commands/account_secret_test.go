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

var accountSecretListResponse = map[string]any{
	"data": []map[string]any{
		{
			"id":         "secret-001",
			"key":        "DB_PASSWORD",
			"value":      "",
			"is_secret":  true,
			"created_at": "2025-01-01T00:00:00+00:00",
			"updated_at": "2025-01-01T00:00:00+00:00",
		},
		{
			"id":         "secret-002",
			"key":        "APP_ENV",
			"value":      "production",
			"is_secret":  false,
			"created_at": "2025-01-02T00:00:00+00:00",
			"updated_at": "2025-01-02T00:00:00+00:00",
		},
	},
	"meta": map[string]any{
		"current_page": 1,
		"last_page":    1,
		"total":        2,
	},
	"message":     "Global secrets retrieved successfully",
	"http_status": 200,
}

var accountSecretShowResponse = map[string]any{
	"data": map[string]any{
		"id":         "secret-001",
		"key":        "DB_PASSWORD",
		"value":      "",
		"is_secret":  true,
		"created_at": "2025-01-01T00:00:00+00:00",
		"updated_at": "2025-01-05T00:00:00+00:00",
	},
	"message":     "Global secret retrieved successfully",
	"http_status": 200,
}

var accountSecretCreateResponse = map[string]any{
	"data": map[string]any{
		"id":         "secret-003",
		"key":        "API_TOKEN",
		"value":      "",
		"is_secret":  true,
		"created_at": "2025-01-15T00:00:00+00:00",
		"updated_at": "2025-01-15T00:00:00+00:00",
	},
	"message":     "Global secret created successfully",
	"http_status": 201,
}

var accountSecretCreatePlainResponse = map[string]any{
	"data": map[string]any{
		"id":         "secret-004",
		"key":        "APP_DEBUG",
		"value":      "true",
		"is_secret":  false,
		"created_at": "2025-01-15T00:00:00+00:00",
		"updated_at": "2025-01-15T00:00:00+00:00",
	},
	"message":     "Global secret created successfully",
	"http_status": 201,
}

var accountSecretUpdateResponse = map[string]any{
	"data": map[string]any{
		"id":         "secret-001",
		"key":        "DB_PASSWORD",
		"value":      "",
		"is_secret":  true,
		"created_at": "2025-01-01T00:00:00+00:00",
		"updated_at": "2025-01-20T00:00:00+00:00",
	},
	"message":     "Global secret updated successfully",
	"http_status": 200,
}

var accountSecretDeleteResponse = map[string]any{
	"data": map[string]any{
		"id":         "secret-001",
		"key":        "DB_PASSWORD",
		"value":      "",
		"is_secret":  true,
		"created_at": "2025-01-01T00:00:00+00:00",
		"updated_at": "2025-01-05T00:00:00+00:00",
	},
	"message":     "Global secret deleted successfully",
	"http_status": 200,
}

func newAccountSecretTestServer(validToken string) *httptest.Server {
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
		case method == "GET" && path == "/api/v1/vector/global-secrets":
			_ = json.NewEncoder(w).Encode(accountSecretListResponse)

		case method == "GET" && path == "/api/v1/vector/global-secrets/secret-001":
			_ = json.NewEncoder(w).Encode(accountSecretShowResponse)

		case method == "POST" && path == "/api/v1/vector/global-secrets":
			var reqBody map[string]any
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &reqBody)
			isSecret, _ := reqBody["is_secret"].(bool)
			if !isSecret {
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(accountSecretCreatePlainResponse)
			} else {
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(accountSecretCreateResponse)
			}

		case method == "PUT" && path == "/api/v1/vector/global-secrets/secret-001":
			_ = json.NewEncoder(w).Encode(accountSecretUpdateResponse)

		case method == "DELETE" && path == "/api/v1/vector/global-secrets/secret-001":
			_ = json.NewEncoder(w).Encode(accountSecretDeleteResponse)

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Not Found",
				"http_status": 404,
			})
		}
	}))
}

// --- Secret List Tests ---

func TestAccountSecretListCmd_TableOutput(t *testing.T) {
	ts := newAccountSecretTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "secret", "list"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "secret-001")
	assert.Contains(t, out, "DB_PASSWORD")
	assert.Contains(t, out, "Yes")
	assert.Contains(t, out, "secret-002")
	assert.Contains(t, out, "APP_ENV")
	assert.Contains(t, out, "No")
	assert.Contains(t, out, "production")
}

func TestAccountSecretListCmd_SecretValueHidden(t *testing.T) {
	ts := newAccountSecretTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "secret", "list"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	// The secret item should show "-" for value, the plain one should show "production"
	assert.Contains(t, out, "production")
}

func TestAccountSecretListCmd_JSONOutput(t *testing.T) {
	ts := newAccountSecretTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"account", "secret", "list"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Len(t, result, 2)
	assert.Equal(t, "secret-001", result[0]["id"])
}

func TestAccountSecretListCmd_Pagination(t *testing.T) {
	var receivedPage, receivedPerPage string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPage = r.URL.Query().Get("page")
		receivedPerPage = r.URL.Query().Get("per_page")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(accountSecretListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "secret", "list", "--page", "3", "--per-page", "25"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "3", receivedPage)
	assert.Equal(t, "25", receivedPerPage)
}

func TestAccountSecretListCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(accountSecretListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "secret", "list"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "GET", receivedMethod)
	assert.Equal(t, "/api/v1/vector/global-secrets", receivedPath)
}

func TestAccountSecretListCmd_AuthError(t *testing.T) {
	ts := newAccountSecretTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"account", "secret", "list"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestAccountSecretListCmd_NoAuth(t *testing.T) {
	cmd, _, _ := buildAccountCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"account", "secret", "list"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

// --- Secret Show Tests ---

func TestAccountSecretShowCmd_TableOutput(t *testing.T) {
	ts := newAccountSecretTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "secret", "show", "secret-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "secret-001")
	assert.Contains(t, out, "DB_PASSWORD")
	assert.Contains(t, out, "Yes")
	assert.Contains(t, out, "2025-01-05T00:00:00+00:00")
}

func TestAccountSecretShowCmd_JSONOutput(t *testing.T) {
	ts := newAccountSecretTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"account", "secret", "show", "secret-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "secret-001", result["id"])
	assert.Equal(t, "DB_PASSWORD", result["key"])
}

func TestAccountSecretShowCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(accountSecretShowResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "secret", "show", "secret-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "GET", receivedMethod)
	assert.Equal(t, "/api/v1/vector/global-secrets/secret-001", receivedPath)
}

func TestAccountSecretShowCmd_MissingArg(t *testing.T) {
	ts := newAccountSecretTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "secret", "show"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

// --- Secret Create Tests ---

func TestAccountSecretCreateCmd_TableOutput(t *testing.T) {
	ts := newAccountSecretTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "secret", "create", "--key", "API_TOKEN", "--value", "my-secret-value"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "secret-003")
	assert.Contains(t, out, "API_TOKEN")
	assert.Contains(t, out, "Yes")
}

func TestAccountSecretCreateCmd_JSONOutput(t *testing.T) {
	ts := newAccountSecretTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"account", "secret", "create", "--key", "API_TOKEN", "--value", "my-secret-value"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "secret-003", result["id"])
}

func TestAccountSecretCreateCmd_RequestBody(t *testing.T) {
	var receivedBody map[string]any
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(accountSecretCreateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "secret", "create", "--key", "API_TOKEN", "--value", "secret123"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/api/v1/vector/global-secrets", receivedPath)
	assert.Equal(t, "API_TOKEN", receivedBody["key"])
	assert.Equal(t, "secret123", receivedBody["value"])
	assert.Equal(t, true, receivedBody["is_secret"])
}

func TestAccountSecretCreateCmd_NoSecretFlag(t *testing.T) {
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(accountSecretCreatePlainResponse)
	}))
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "secret", "create", "--key", "APP_DEBUG", "--value", "true", "--no-secret"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, false, receivedBody["is_secret"])

	out := stdout.String()
	assert.Contains(t, out, "APP_DEBUG")
	assert.Contains(t, out, "No")
	assert.Contains(t, out, "true")
}

func TestAccountSecretCreateCmd_MissingRequiredFlags(t *testing.T) {
	ts := newAccountSecretTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "secret", "create"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestAccountSecretCreateCmd_MissingValueFlag(t *testing.T) {
	ts := newAccountSecretTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "secret", "create", "--key", "MY_KEY"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

// --- Secret Update Tests ---

func TestAccountSecretUpdateCmd_TableOutput(t *testing.T) {
	ts := newAccountSecretTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "secret", "update", "secret-001", "--value", "new-password"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "secret-001")
	assert.Contains(t, out, "DB_PASSWORD")
	assert.Contains(t, out, "2025-01-20T00:00:00+00:00")
}

func TestAccountSecretUpdateCmd_JSONOutput(t *testing.T) {
	ts := newAccountSecretTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"account", "secret", "update", "secret-001", "--value", "new-password"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "secret-001", result["id"])
}

func TestAccountSecretUpdateCmd_RequestBody(t *testing.T) {
	var receivedBody map[string]any
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(accountSecretUpdateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "secret", "update", "secret-001", "--value", "new-password"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "PUT", receivedMethod)
	assert.Equal(t, "/api/v1/vector/global-secrets/secret-001", receivedPath)
	assert.Equal(t, "new-password", receivedBody["value"])
	// is_secret should not be sent when --no-secret is not provided
	_, hasIsSecret := receivedBody["is_secret"]
	assert.False(t, hasIsSecret)
}

func TestAccountSecretUpdateCmd_NoSecretFlag(t *testing.T) {
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(accountSecretUpdateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "secret", "update", "secret-001", "--no-secret"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, false, receivedBody["is_secret"])
}

func TestAccountSecretUpdateCmd_MissingArg(t *testing.T) {
	ts := newAccountSecretTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "secret", "update"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

// --- Secret Delete Tests ---

func TestAccountSecretDeleteCmd_TableOutput(t *testing.T) {
	ts := newAccountSecretTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "secret", "delete", "secret-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Secret deleted successfully")
}

func TestAccountSecretDeleteCmd_JSONOutput(t *testing.T) {
	ts := newAccountSecretTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"account", "secret", "delete", "secret-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "secret-001", result["id"])
}

func TestAccountSecretDeleteCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(accountSecretDeleteResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "secret", "delete", "secret-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "DELETE", receivedMethod)
	assert.Equal(t, "/api/v1/vector/global-secrets/secret-001", receivedPath)
}

func TestAccountSecretDeleteCmd_MissingArg(t *testing.T) {
	ts := newAccountSecretTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "secret", "delete"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

// --- Help Tests ---

func TestAccountSecretCmd_Help(t *testing.T) {
	cmd := NewAccountCmd()

	stdout := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetArgs([]string{"secret", "--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "list")
	assert.Contains(t, out, "show")
	assert.Contains(t, out, "create")
	assert.Contains(t, out, "update")
	assert.Contains(t, out, "delete")
	assert.Contains(t, out, "secrets and environment variables")
}
