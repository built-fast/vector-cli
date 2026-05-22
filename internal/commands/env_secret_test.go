package commands

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/output"
)

var secretListResponse = map[string]any{
	"data": []map[string]any{
		{
			"id":         "sec-001",
			"key":        "API_KEY",
			"is_secret":  true,
			"created_at": "2025-01-15T12:00:00+00:00",
			"updated_at": "2025-01-15T12:00:00+00:00",
		},
		{
			"id":         "sec-002",
			"key":        "APP_DEBUG",
			"is_secret":  false,
			"value":      "true",
			"created_at": "2025-01-15T12:00:00+00:00",
			"updated_at": "2025-01-15T12:00:00+00:00",
		},
	},
	"meta": map[string]any{
		"current_page": 1,
		"last_page":    1,
		"total":        2,
	},
	"message":     "Environment secrets retrieved successfully",
	"http_status": 200,
}

var secretShowResponse = map[string]any{
	"data": map[string]any{
		"id":         "sec-002",
		"key":        "APP_DEBUG",
		"is_secret":  false,
		"value":      "true",
		"created_at": "2025-01-15T12:00:00+00:00",
		"updated_at": "2025-01-15T12:00:00+00:00",
	},
	"message":     "Environment secret retrieved successfully",
	"http_status": 200,
}

var secretShowSecretResponse = map[string]any{
	"data": map[string]any{
		"id":         "sec-001",
		"key":        "API_KEY",
		"is_secret":  true,
		"created_at": "2025-01-15T12:00:00+00:00",
		"updated_at": "2025-01-15T12:00:00+00:00",
	},
	"message":     "Environment secret retrieved successfully",
	"http_status": 200,
}

var secretCreateResponse = map[string]any{
	"data": map[string]any{
		"id":         "sec-003",
		"key":        "NEW_SECRET",
		"is_secret":  true,
		"created_at": "2025-01-15T12:00:00+00:00",
		"updated_at": "2025-01-15T12:00:00+00:00",
	},
	"message":     "Environment secret created successfully",
	"http_status": 201,
}

var secretUpdateResponse = map[string]any{
	"data": map[string]any{
		"id":         "sec-001",
		"key":        "UPDATED_KEY",
		"is_secret":  true,
		"created_at": "2025-01-15T12:00:00+00:00",
		"updated_at": "2025-01-16T12:00:00+00:00",
	},
	"message":     "Environment secret updated successfully",
	"http_status": 200,
}

var secretDeleteResponse = map[string]any{
	"data": map[string]any{
		"id":         "sec-001",
		"key":        "API_KEY",
		"is_secret":  true,
		"created_at": "2025-01-15T12:00:00+00:00",
		"updated_at": "2025-01-15T12:00:00+00:00",
	},
	"message":     "Environment secret deleted successfully",
	"http_status": 200,
}

func newSecretTestServer(validToken string) *httptest.Server {
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
		case method == "GET" && path == "/api/v1/vector/environments/env-001/secrets":
			_ = json.NewEncoder(w).Encode(secretListResponse)

		case method == "GET" && path == "/api/v1/vector/secrets/sec-001":
			_ = json.NewEncoder(w).Encode(secretShowSecretResponse)

		case method == "GET" && path == "/api/v1/vector/secrets/sec-002":
			_ = json.NewEncoder(w).Encode(secretShowResponse)

		case method == "POST" && path == "/api/v1/vector/environments/env-001/secrets":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(secretCreateResponse)

		case method == "PUT" && path == "/api/v1/vector/secrets/sec-001":
			_ = json.NewEncoder(w).Encode(secretUpdateResponse)

		case method == "DELETE" && path == "/api/v1/vector/secrets/sec-001":
			_ = json.NewEncoder(w).Encode(secretDeleteResponse)

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

func TestEnvSecretListCmd_TableOutput(t *testing.T) {
	ts := newSecretTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "secret", "list", "env-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "sec-001")
	assert.Contains(t, out, "API_KEY")
	assert.Contains(t, out, "Yes")
	assert.Contains(t, out, "sec-002")
	assert.Contains(t, out, "APP_DEBUG")
}

func TestEnvSecretListCmd_JSONOutput(t *testing.T) {
	ts := newSecretTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"env", "secret", "list", "env-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Len(t, result, 2)
	assert.Equal(t, "sec-001", result[0]["id"])
}

func TestEnvSecretListCmd_Pagination(t *testing.T) {
	var receivedPage, receivedPerPage string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPage = r.URL.Query().Get("page")
		receivedPerPage = r.URL.Query().Get("per_page")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(secretListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "secret", "list", "env-001", "--page", "2", "--per-page", "5"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "2", receivedPage)
	assert.Equal(t, "5", receivedPerPage)
}

// --- Secret Show Tests ---

func TestEnvSecretShowCmd_EnvVar(t *testing.T) {
	ts := newSecretTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "secret", "show", "sec-002"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "sec-002")
	assert.Contains(t, out, "APP_DEBUG")
	assert.Contains(t, out, "No")   // is_secret = false
	assert.Contains(t, out, "true") // value shown for non-secrets
}

func TestEnvSecretShowCmd_Secret(t *testing.T) {
	ts := newSecretTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "secret", "show", "sec-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "sec-001")
	assert.Contains(t, out, "API_KEY")
	assert.Contains(t, out, "Yes")      // is_secret = true
	assert.NotContains(t, out, "Value") // value not shown for secrets
}

func TestEnvSecretShowCmd_JSONOutput(t *testing.T) {
	ts := newSecretTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"env", "secret", "show", "sec-002"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "sec-002", result["id"])
	assert.Equal(t, "APP_DEBUG", result["key"])
}

// --- Secret Create Tests ---

func TestEnvSecretCreateCmd_TableOutput(t *testing.T) {
	ts := newSecretTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "secret", "create", "env-001", "--key", "NEW_SECRET", "--value", "secret123"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "sec-003")
	assert.Contains(t, out, "NEW_SECRET")
}

func TestEnvSecretCreateCmd_JSONOutput(t *testing.T) {
	ts := newSecretTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"env", "secret", "create", "env-001", "--key", "NEW_SECRET", "--value", "secret123"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "sec-003", result["id"])
}

func TestEnvSecretCreateCmd_RequestBody(t *testing.T) {
	var receivedBody map[string]any
	var receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(secretCreateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "secret", "create", "env-001",
		"--key", "MY_VAR",
		"--value", "my_value",
		"--is-secret=false",
	})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "/api/v1/vector/environments/env-001/secrets", receivedPath)
	assert.Equal(t, "MY_VAR", receivedBody["key"])
	assert.Equal(t, "my_value", receivedBody["value"])
	assert.Equal(t, false, receivedBody["is_secret"])
}

func TestEnvSecretCreateCmd_MissingKey(t *testing.T) {
	ts := newSecretTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "secret", "create", "env-001", "--value", "test"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "key")
}

// --- Secret Update Tests ---

func TestEnvSecretUpdateCmd_TableOutput(t *testing.T) {
	ts := newSecretTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "secret", "update", "sec-001", "--key", "UPDATED_KEY"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "sec-001")
	assert.Contains(t, out, "UPDATED_KEY")
}

func TestEnvSecretUpdateCmd_JSONOutput(t *testing.T) {
	ts := newSecretTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"env", "secret", "update", "sec-001", "--key", "UPDATED_KEY"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "sec-001", result["id"])
}

func TestEnvSecretUpdateCmd_RequestBody(t *testing.T) {
	var receivedBody map[string]any
	var receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(secretUpdateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "secret", "update", "sec-001", "--key", "NEW_KEY", "--value", "new_val"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "/api/v1/vector/secrets/sec-001", receivedPath)
	assert.Equal(t, "NEW_KEY", receivedBody["key"])
	assert.Equal(t, "new_val", receivedBody["value"])
}

// --- Secret Delete Tests ---

func TestEnvSecretDeleteCmd_WithForce(t *testing.T) {
	ts := newSecretTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "secret", "delete", "sec-001", "--force"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Secret deleted successfully")
}

func TestEnvSecretDeleteCmd_JSONOutput(t *testing.T) {
	ts := newSecretTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"env", "secret", "delete", "sec-001", "--force"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "sec-001", result["id"])
}

func TestEnvSecretDeleteCmd_ConfirmAbort(t *testing.T) {
	ts := newSecretTestServer("valid-token")
	defer ts.Close()

	origReader := confirmReader
	confirmReader = strings.NewReader("n\n")
	t.Cleanup(func() { confirmReader = origReader })

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "secret", "delete", "sec-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Aborted")
}

func TestEnvSecretDeleteCmd_ConfirmYes(t *testing.T) {
	ts := newSecretTestServer("valid-token")
	defer ts.Close()

	origReader := confirmReader
	confirmReader = strings.NewReader("y\n")
	t.Cleanup(func() { confirmReader = origReader })

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "secret", "delete", "sec-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Secret deleted successfully")
}

func TestEnvSecretDeleteCmd_HTTPMethod(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(secretDeleteResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "secret", "delete", "sec-001", "--force"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "DELETE", receivedMethod)
	assert.Equal(t, "/api/v1/vector/secrets/sec-001", receivedPath)
}

// --- Auth Error Tests ---

func TestEnvSecretListCmd_AuthError(t *testing.T) {
	ts := newSecretTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildEnvCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"env", "secret", "list", "env-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestEnvSecretListCmd_NoAuth(t *testing.T) {
	cmd, _, _ := buildEnvCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"env", "secret", "list", "env-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}
