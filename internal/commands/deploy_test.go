package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/appctx"
	"github.com/built-fast/vector-cli/internal/config"
	"github.com/built-fast/vector-cli/internal/output"
)

var deployListResponse = map[string]any{
	"data": []map[string]any{
		{
			"id":                     "dep-001",
			"vector_environment_id":  "env-001",
			"status":                 "deployed",
			"stdout":                 "Deployment successful",
			"stderr":                 nil,
			"actor":                  "user@example.com",
			"created_at":             "2025-01-15T12:00:00+00:00",
			"updated_at":             "2025-01-15T12:05:00+00:00",
		},
		{
			"id":                     "dep-002",
			"vector_environment_id":  "env-001",
			"status":                 "deployed",
			"stdout":                 "Deployment successful",
			"stderr":                 nil,
			"actor":                  "admin@example.com",
			"created_at":             "2025-01-14T10:00:00+00:00",
			"updated_at":             "2025-01-14T10:03:00+00:00",
		},
	},
	"meta": map[string]any{
		"current_page": 1,
		"last_page":    1,
		"total":        2,
	},
	"message":     "Deployments retrieved successfully",
	"http_status": 200,
}

var deployShowResponse = map[string]any{
	"data": map[string]any{
		"id":                     "dep-001",
		"vector_environment_id":  "env-001",
		"status":                 "deployed",
		"stdout":                 "Deploying files...\nDone.",
		"stderr":                 nil,
		"actor":                  "user@example.com",
		"created_at":             "2025-01-15T12:00:00+00:00",
		"updated_at":             "2025-01-15T12:05:00+00:00",
	},
	"message":     "Deployment retrieved successfully",
	"http_status": 200,
}

var deployShowWithStderrResponse = map[string]any{
	"data": map[string]any{
		"id":                     "dep-003",
		"vector_environment_id":  "env-001",
		"status":                 "failed",
		"stdout":                 "Deploying files...",
		"stderr":                 "Error: permission denied",
		"actor":                  "user@example.com",
		"created_at":             "2025-01-15T12:00:00+00:00",
		"updated_at":             "2025-01-15T12:05:00+00:00",
	},
	"message":     "Deployment retrieved successfully",
	"http_status": 200,
}

var deployShowNoOutputResponse = map[string]any{
	"data": map[string]any{
		"id":                     "dep-004",
		"vector_environment_id":  "env-001",
		"status":                 "pending",
		"stdout":                 nil,
		"stderr":                 nil,
		"actor":                  "user@example.com",
		"created_at":             "2025-01-15T12:00:00+00:00",
		"updated_at":             "2025-01-15T12:05:00+00:00",
	},
	"message":     "Deployment retrieved successfully",
	"http_status": 200,
}

var deployTriggerResponse = map[string]any{
	"data": map[string]any{
		"id":                     "dep-005",
		"vector_environment_id":  "env-001",
		"status":                 "pending",
		"stdout":                 nil,
		"stderr":                 nil,
		"actor":                  "user@example.com",
		"created_at":             "2025-01-15T12:00:00+00:00",
		"updated_at":             "2025-01-15T12:00:00+00:00",
	},
	"message":     "Deployment initiated",
	"http_status": 201,
}

var deployRollbackResponse = map[string]any{
	"data": map[string]any{
		"id":                     "dep-006",
		"vector_environment_id":  "env-001",
		"status":                 "pending",
		"stdout":                 nil,
		"stderr":                 nil,
		"actor":                  "user@example.com",
		"created_at":             "2025-01-15T12:00:00+00:00",
		"updated_at":             "2025-01-15T12:00:00+00:00",
	},
	"message":     "Rollback initiated",
	"http_status": 201,
}

func newDeployTestServer(validToken string) *httptest.Server {
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
		case method == "GET" && path == "/api/v1/vector/environments/env-001/deployments":
			_ = json.NewEncoder(w).Encode(deployListResponse)

		case method == "GET" && path == "/api/v1/vector/deployments/dep-001":
			_ = json.NewEncoder(w).Encode(deployShowResponse)

		case method == "GET" && path == "/api/v1/vector/deployments/dep-003":
			_ = json.NewEncoder(w).Encode(deployShowWithStderrResponse)

		case method == "GET" && path == "/api/v1/vector/deployments/dep-004":
			_ = json.NewEncoder(w).Encode(deployShowNoOutputResponse)

		case method == "POST" && path == "/api/v1/vector/environments/env-001/deployments":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(deployTriggerResponse)

		case method == "POST" && path == "/api/v1/vector/environments/env-001/rollback":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(deployRollbackResponse)

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Not Found",
				"http_status": 404,
			})
		}
	}))
}

func buildDeployCmd(baseURL, token string, format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	root := &cobra.Command{
		Use: "vector",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			client := api.NewClient(baseURL, token, "test-agent")
			app := appctx.NewApp(
				config.DefaultConfig(),
				client,
				"",
			)
			app.Output = output.NewWriter(stdout, format)
			cmd.SetContext(appctx.WithApp(cmd.Context(), app))
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	deployCmd := NewDeployCmd()
	root.AddCommand(deployCmd)

	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

func buildDeployCmdNoAuth(format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	root := &cobra.Command{
		Use: "vector",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			client := api.NewClient("http://localhost", "", "test-agent")
			app := appctx.NewApp(
				config.DefaultConfig(),
				client,
				"",
			)
			app.Output = output.NewWriter(stdout, format)
			cmd.SetContext(appctx.WithApp(cmd.Context(), app))
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	deployCmd := NewDeployCmd()
	root.AddCommand(deployCmd)

	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

// --- Deploy List Tests ---

func TestDeployListCmd_TableOutput(t *testing.T) {
	ts := newDeployTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildDeployCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"deploy", "list", "env-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "dep-001")
	assert.Contains(t, out, "dep-002")
	assert.Contains(t, out, "deployed")
	assert.Contains(t, out, "user@example.com")
	assert.Contains(t, out, "admin@example.com")
}

func TestDeployListCmd_JSONOutput(t *testing.T) {
	ts := newDeployTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildDeployCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"deploy", "list", "env-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Len(t, result, 2)
	assert.Equal(t, "dep-001", result[0]["id"])
	assert.Equal(t, "dep-002", result[1]["id"])
}

func TestDeployListCmd_PaginationQueryParams(t *testing.T) {
	var receivedPath, receivedQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(deployListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildDeployCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"deploy", "list", "env-001", "--page", "2", "--per-page", "10"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "/api/v1/vector/environments/env-001/deployments", receivedPath)
	assert.Contains(t, receivedQuery, "page=2")
	assert.Contains(t, receivedQuery, "per_page=10")
}

func TestDeployListCmd_AuthError(t *testing.T) {
	ts := newDeployTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildDeployCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"deploy", "list", "env-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestDeployListCmd_NoAuthToken(t *testing.T) {
	cmd, _, _ := buildDeployCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"deploy", "list", "env-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestDeployListCmd_MissingArgs(t *testing.T) {
	ts := newDeployTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildDeployCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"deploy", "list"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

// --- Deploy Show Tests ---

func TestDeployShowCmd_TableOutput(t *testing.T) {
	ts := newDeployTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildDeployCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"deploy", "show", "dep-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "dep-001")
	assert.Contains(t, out, "env-001")
	assert.Contains(t, out, "deployed")
	assert.Contains(t, out, "user@example.com")
	assert.Contains(t, out, "Stdout:")
	assert.Contains(t, out, "Deploying files...")
	assert.NotContains(t, out, "Stderr:")
}

func TestDeployShowCmd_WithStderr(t *testing.T) {
	ts := newDeployTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildDeployCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"deploy", "show", "dep-003"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "dep-003")
	assert.Contains(t, out, "failed")
	assert.Contains(t, out, "Stdout:")
	assert.Contains(t, out, "Deploying files...")
	assert.Contains(t, out, "Stderr:")
	assert.Contains(t, out, "Error: permission denied")
}

func TestDeployShowCmd_NoOutput(t *testing.T) {
	ts := newDeployTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildDeployCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"deploy", "show", "dep-004"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "dep-004")
	assert.Contains(t, out, "pending")
	assert.NotContains(t, out, "Stdout:")
	assert.NotContains(t, out, "Stderr:")
}

func TestDeployShowCmd_JSONOutput(t *testing.T) {
	ts := newDeployTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildDeployCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"deploy", "show", "dep-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "dep-001", result["id"])
	assert.Equal(t, "deployed", result["status"])
}

func TestDeployShowCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(deployShowResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildDeployCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"deploy", "show", "dep-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "GET", receivedMethod)
	assert.Equal(t, "/api/v1/vector/deployments/dep-001", receivedPath)
}

func TestDeployShowCmd_AuthError(t *testing.T) {
	ts := newDeployTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildDeployCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"deploy", "show", "dep-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

// --- Deploy Trigger Tests ---

func TestDeployTriggerCmd_TableOutput(t *testing.T) {
	ts := newDeployTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildDeployCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"deploy", "trigger", "env-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "dep-005")
	assert.Contains(t, out, "env-001")
	assert.Contains(t, out, "pending")
}

func TestDeployTriggerCmd_JSONOutput(t *testing.T) {
	ts := newDeployTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildDeployCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"deploy", "trigger", "env-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "dep-005", result["id"])
	assert.Equal(t, "pending", result["status"])
}

func TestDeployTriggerCmd_RequestBodyWithFlags(t *testing.T) {
	var receivedBody map[string]any
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(deployTriggerResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildDeployCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"deploy", "trigger", "env-001", "--include-uploads", "--include-database=false"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/api/v1/vector/environments/env-001/deployments", receivedPath)
	assert.Equal(t, true, receivedBody["include_uploads"])
	assert.Equal(t, false, receivedBody["include_database"])
}

func TestDeployTriggerCmd_RequestBodyNoFlags(t *testing.T) {
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(deployTriggerResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildDeployCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"deploy", "trigger", "env-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	// When no flags are set, the body should be empty (no include_uploads or include_database)
	assert.Nil(t, receivedBody["include_uploads"])
	assert.Nil(t, receivedBody["include_database"])
}

func TestDeployTriggerCmd_AuthError(t *testing.T) {
	ts := newDeployTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildDeployCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"deploy", "trigger", "env-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestDeployTriggerCmd_MissingArgs(t *testing.T) {
	ts := newDeployTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildDeployCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"deploy", "trigger"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

// --- Deploy Rollback Tests ---

func TestDeployRollbackCmd_TableOutput(t *testing.T) {
	ts := newDeployTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildDeployCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"deploy", "rollback", "env-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "dep-006")
	assert.Contains(t, out, "env-001")
	assert.Contains(t, out, "pending")
}

func TestDeployRollbackCmd_JSONOutput(t *testing.T) {
	ts := newDeployTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildDeployCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"deploy", "rollback", "env-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "dep-006", result["id"])
	assert.Equal(t, "pending", result["status"])
}

func TestDeployRollbackCmd_WithTargetFlag(t *testing.T) {
	var receivedBody map[string]any
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(deployRollbackResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildDeployCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"deploy", "rollback", "env-001", "--target", "dep-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/api/v1/vector/environments/env-001/rollback", receivedPath)
	assert.Equal(t, "dep-001", receivedBody["target_deployment_id"])
}

func TestDeployRollbackCmd_WithoutTargetFlag(t *testing.T) {
	var receivedBody map[string]any
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(deployRollbackResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildDeployCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"deploy", "rollback", "env-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/api/v1/vector/environments/env-001/rollback", receivedPath)
	assert.Nil(t, receivedBody["target_deployment_id"])
}

func TestDeployRollbackCmd_AuthError(t *testing.T) {
	ts := newDeployTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildDeployCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"deploy", "rollback", "env-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestDeployRollbackCmd_MissingArgs(t *testing.T) {
	ts := newDeployTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildDeployCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"deploy", "rollback"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

// --- Help Text Tests ---

func TestDeployCmd_HelpText(t *testing.T) {
	ts := newDeployTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildDeployCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"deploy", "--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "list")
	assert.Contains(t, out, "show")
	assert.Contains(t, out, "trigger")
	assert.Contains(t, out, "rollback")
}

func TestDeployTriggerCmd_HelpText(t *testing.T) {
	ts := newDeployTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildDeployCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"deploy", "trigger", "--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "--include-uploads")
	assert.Contains(t, out, "--include-database")
}

func TestDeployRollbackCmd_HelpText(t *testing.T) {
	ts := newDeployTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildDeployCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"deploy", "rollback", "--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "--target")
}

// --- Server Error Tests ---

func TestDeployShowCmd_NotFound(t *testing.T) {
	ts := newDeployTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildDeployCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"deploy", "show", "dep-nonexistent"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to show deployment")
}

// --- Deploy Trigger --wait Tests ---

// newDeployWaitTestServer creates a test server that handles:
// - POST /environments/{id}/deployments -> returns deployTriggerResponse
// - POST /environments/{id}/rollback -> returns deployRollbackResponse
// - GET /deployments/{id} -> returns successive poll responses
func newDeployWaitTestServer(validToken string, pollResponses []countingResponse) *httptest.Server {
	var pollCount atomic.Int64

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
		case method == "POST" && path == "/api/v1/vector/environments/env-001/deployments":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(deployTriggerResponse)

		case method == "POST" && path == "/api/v1/vector/environments/env-001/rollback":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(deployRollbackResponse)

		case method == "GET" && (path == "/api/v1/vector/deployments/dep-005" || path == "/api/v1/vector/deployments/dep-006"):
			idx := int(pollCount.Add(1)) - 1
			if idx >= len(pollResponses) {
				idx = len(pollResponses) - 1
			}
			resp := pollResponses[idx]
			if resp.httpStatus != 0 {
				w.WriteHeader(resp.httpStatus)
			}
			_ = json.NewEncoder(w).Encode(resp.body)

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Not Found",
				"http_status": 404,
			})
		}
	}))
}

func makeDeployPollResponse(id, status string) countingResponse {
	return countingResponse{
		httpStatus: http.StatusOK,
		body: map[string]any{
			"data": map[string]any{
				"id":                    id,
				"vector_environment_id": "env-001",
				"status":                status,
				"actor":                 "user@example.com",
				"created_at":            "2025-01-15T12:00:00+00:00",
				"updated_at":            "2025-01-15T12:05:00+00:00",
			},
			"message":     "Deployment retrieved successfully",
			"http_status": 200,
		},
	}
}

func TestDeployTriggerCmd_WaitSuccess(t *testing.T) {
	overrideWaitGlobals(t, false)

	ts := newDeployWaitTestServer("valid-token", []countingResponse{
		makeDeployPollResponse("dep-005", "pending"),
		makeDeployPollResponse("dep-005", "deploying"),
		makeDeployPollResponse("dep-005", "deployed"),
	})
	defer ts.Close()

	cmd, stdout, _ := buildDeployCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"deploy", "trigger", "env-001", "--wait", "--poll-interval", "1s", "--timeout", "30s"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "dep-005")
	assert.Contains(t, out, "deployed")
	assert.Contains(t, out, "Deployment dep-005 deployed in")
}

func TestDeployTriggerCmd_WaitFailure(t *testing.T) {
	overrideWaitGlobals(t, false)

	ts := newDeployWaitTestServer("valid-token", []countingResponse{
		makeDeployPollResponse("dep-005", "pending"),
		makeDeployPollResponse("dep-005", "failed"),
	})
	defer ts.Close()

	cmd, _, _ := buildDeployCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"deploy", "trigger", "env-001", "--wait", "--poll-interval", "1s", "--timeout", "30s"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 1, apiErr.ExitCode)
	assert.Contains(t, apiErr.Message, "failed status")
}

func TestDeployTriggerCmd_WaitJSON(t *testing.T) {
	overrideWaitGlobals(t, false)

	ts := newDeployWaitTestServer("valid-token", []countingResponse{
		makeDeployPollResponse("dep-005", "pending"),
		makeDeployPollResponse("dep-005", "deployed"),
	})
	defer ts.Close()

	cmd, stdout, _ := buildDeployCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"deploy", "trigger", "env-001", "--wait", "--poll-interval", "1s", "--timeout", "30s"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "dep-005", result["id"])
	assert.Equal(t, "deployed", result["status"])
}

// --- Deploy Rollback --wait Tests ---

func TestDeployRollbackCmd_WaitSuccess(t *testing.T) {
	overrideWaitGlobals(t, false)

	ts := newDeployWaitTestServer("valid-token", []countingResponse{
		makeDeployPollResponse("dep-006", "pending"),
		makeDeployPollResponse("dep-006", "deployed"),
	})
	defer ts.Close()

	cmd, stdout, _ := buildDeployCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"deploy", "rollback", "env-001", "--wait", "--poll-interval", "1s", "--timeout", "30s"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "dep-006")
	assert.Contains(t, out, "deployed")
	assert.Contains(t, out, "Deployment dep-006 deployed in")
}

func TestDeployRollbackCmd_WaitFailure(t *testing.T) {
	overrideWaitGlobals(t, false)

	ts := newDeployWaitTestServer("valid-token", []countingResponse{
		makeDeployPollResponse("dep-006", "pending"),
		makeDeployPollResponse("dep-006", "cancelled"),
	})
	defer ts.Close()

	cmd, _, _ := buildDeployCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"deploy", "rollback", "env-001", "--wait", "--poll-interval", "1s", "--timeout", "30s"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 1, apiErr.ExitCode)
	assert.Contains(t, apiErr.Message, "failed status")
	assert.Contains(t, apiErr.Message, "cancelled")
}

func TestDeployRollbackCmd_WaitJSON(t *testing.T) {
	overrideWaitGlobals(t, false)

	ts := newDeployWaitTestServer("valid-token", []countingResponse{
		makeDeployPollResponse("dep-006", "pending"),
		makeDeployPollResponse("dep-006", "deployed"),
	})
	defer ts.Close()

	cmd, stdout, _ := buildDeployCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"deploy", "rollback", "env-001", "--wait", "--poll-interval", "1s", "--timeout", "30s"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "dep-006", result["id"])
	assert.Equal(t, "deployed", result["status"])
}
