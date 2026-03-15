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

var sslStatusResponse = map[string]any{
	"data": map[string]any{
		"id":                "env-001",
		"vector_site_id":    "site-001",
		"name":              "production",
		"is_production":     true,
		"status":            "active",
		"provisioning_step": "complete",
		"failure_reason":    nil,
		"php_version":       "8.3",
		"platform_domain":   "wispy-dust.vectorpages.com",
		"custom_domain":     "example.com",
		"created_at":        "2025-01-15T12:00:00+00:00",
		"updated_at":        "2025-01-15T12:00:00+00:00",
	},
	"message":     "SSL status retrieved",
	"http_status": 200,
}

var sslNudgeProgressedResponse = map[string]any{
	"data": map[string]any{
		"id":                "env-001",
		"vector_site_id":    "site-001",
		"name":              "production",
		"is_production":     true,
		"status":            "provisioning",
		"provisioning_step": "deploying",
		"failure_reason":    nil,
		"php_version":       "8.3",
		"platform_domain":   "wispy-dust.vectorpages.com",
		"custom_domain":     "example.com",
		"created_at":        "2025-01-15T12:00:00+00:00",
		"updated_at":        "2025-01-15T12:00:00+00:00",
	},
	"message":     "SSL provisioning advanced from waiting_cert to deploying",
	"http_status": 200,
}

var sslNudgeWaitingResponse = map[string]any{
	"data": map[string]any{
		"id":                "env-001",
		"vector_site_id":    "site-001",
		"name":              "production",
		"is_production":     true,
		"status":            "provisioning",
		"provisioning_step": "waiting_custom_dns",
		"failure_reason":    nil,
		"php_version":       "8.3",
		"platform_domain":   "wispy-dust.vectorpages.com",
		"custom_domain":     "example.com",
		"created_at":        "2025-01-15T12:00:00+00:00",
		"updated_at":        "2025-01-15T12:00:00+00:00",
	},
	"message":     "SSL provisioning is waiting and cannot advance yet. Current step: waiting_custom_dns",
	"http_status": 200,
}

func newSSLTestServer(validToken string) *httptest.Server {
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
		case method == "GET" && path == "/api/v1/vector/environments/env-001/ssl":
			_ = json.NewEncoder(w).Encode(sslStatusResponse)

		case method == "POST" && path == "/api/v1/vector/environments/env-001/ssl/nudge":
			body, _ := io.ReadAll(r.Body)
			var reqBody map[string]any
			_ = json.Unmarshal(body, &reqBody)

			if retry, ok := reqBody["retry"].(bool); ok && retry {
				_ = json.NewEncoder(w).Encode(sslNudgeProgressedResponse)
			} else {
				_ = json.NewEncoder(w).Encode(sslNudgeWaitingResponse)
			}

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Not Found",
				"http_status": 404,
			})
		}
	}))
}

func buildSSLCmd(baseURL, token string, format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	stdout := new(bytes.Buffer)

	root := &cobra.Command{
		Use: "vector",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			client := api.NewClient(baseURL, token, "test-agent")
			app := appctx.NewApp(
				config.DefaultConfig(),
				&config.Credentials{ApiKey: token},
				client,
				format,
				"",
			)
			app.Output = output.NewWriter(stdout, format)
			cmd.SetContext(appctx.WithApp(cmd.Context(), app))
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	sslCmd := NewSSLCmd()
	root.AddCommand(sslCmd)

	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

func buildSSLCmdNoAuth(format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	stdout := new(bytes.Buffer)

	root := &cobra.Command{
		Use: "vector",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			client := api.NewClient("http://localhost", "", "test-agent")
			app := appctx.NewApp(
				config.DefaultConfig(),
				&config.Credentials{},
				client,
				format,
				"",
			)
			app.Output = output.NewWriter(stdout, format)
			cmd.SetContext(appctx.WithApp(cmd.Context(), app))
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	sslCmd := NewSSLCmd()
	root.AddCommand(sslCmd)

	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

// --- SSL Status Tests ---

func TestSSLStatusCmd_TableOutput(t *testing.T) {
	ts := newSSLTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSSLCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"ssl", "status", "env-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "active")
	assert.Contains(t, out, "complete")
	assert.Contains(t, out, "Yes")
	assert.Contains(t, out, "example.com")
	assert.Contains(t, out, "wispy-dust.vectorpages.com")
}

func TestSSLStatusCmd_JSONOutput(t *testing.T) {
	ts := newSSLTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSSLCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"ssl", "status", "env-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "env-001", result["id"])
	assert.Equal(t, "active", result["status"])
	assert.Equal(t, "complete", result["provisioning_step"])
}

func TestSSLStatusCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sslStatusResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildSSLCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"ssl", "status", "env-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "GET", receivedMethod)
	assert.Equal(t, "/api/v1/vector/environments/env-001/ssl", receivedPath)
}

func TestSSLStatusCmd_AuthError(t *testing.T) {
	ts := newSSLTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildSSLCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"ssl", "status", "env-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestSSLStatusCmd_NoAuthToken(t *testing.T) {
	cmd, _, _ := buildSSLCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"ssl", "status", "env-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestSSLStatusCmd_MissingArgs(t *testing.T) {
	ts := newSSLTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildSSLCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"ssl", "status"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

// --- SSL Nudge Tests ---

func TestSSLNudgeCmd_TableOutput(t *testing.T) {
	ts := newSSLTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSSLCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"ssl", "nudge", "env-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "SSL provisioning is waiting")
	assert.Contains(t, out, "waiting_custom_dns")
	assert.Contains(t, out, "provisioning")
}

func TestSSLNudgeCmd_WithRetryFlag(t *testing.T) {
	ts := newSSLTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSSLCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"ssl", "nudge", "env-001", "--retry"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "SSL provisioning advanced")
	assert.Contains(t, out, "deploying")
}

func TestSSLNudgeCmd_JSONOutput(t *testing.T) {
	ts := newSSLTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSSLCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"ssl", "nudge", "env-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "env-001", result["id"])
	assert.Equal(t, "provisioning", result["status"])
}

func TestSSLNudgeCmd_RequestBodyWithRetry(t *testing.T) {
	var receivedBody map[string]any
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sslNudgeProgressedResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildSSLCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"ssl", "nudge", "env-001", "--retry"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/api/v1/vector/environments/env-001/ssl/nudge", receivedPath)
	assert.Equal(t, true, receivedBody["retry"])
}

func TestSSLNudgeCmd_RequestBodyWithoutRetry(t *testing.T) {
	var receivedBody map[string]any
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sslNudgeWaitingResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildSSLCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"ssl", "nudge", "env-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/api/v1/vector/environments/env-001/ssl/nudge", receivedPath)
	assert.Nil(t, receivedBody["retry"])
}

func TestSSLNudgeCmd_AuthError(t *testing.T) {
	ts := newSSLTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildSSLCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"ssl", "nudge", "env-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestSSLNudgeCmd_NoAuthToken(t *testing.T) {
	cmd, _, _ := buildSSLCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"ssl", "nudge", "env-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestSSLNudgeCmd_MissingArgs(t *testing.T) {
	ts := newSSLTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildSSLCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"ssl", "nudge"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

// --- Help Text Tests ---

func TestSSLCmd_HelpText(t *testing.T) {
	ts := newSSLTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSSLCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"ssl", "--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "status")
	assert.Contains(t, out, "nudge")
}

func TestSSLStatusCmd_HelpText(t *testing.T) {
	ts := newSSLTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSSLCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"ssl", "status", "--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "env-id")
}

func TestSSLNudgeCmd_HelpText(t *testing.T) {
	ts := newSSLTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSSLCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"ssl", "nudge", "--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "--retry")
	assert.Contains(t, out, "env-id")
}
