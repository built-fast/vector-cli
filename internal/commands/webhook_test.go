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

var webhookListResponse = map[string]any{
	"data": []map[string]any{
		{
			"id":         "wh-001",
			"type":       "http",
			"url":        "https://example.com/webhook",
			"events":     []any{"site.created", "deployment.completed"},
			"enabled":    true,
			"created_at": "2025-01-01T00:00:00+00:00",
			"updated_at": "2025-01-01T00:00:00+00:00",
		},
		{
			"id":         "wh-002",
			"type":       "slack",
			"url":        "https://hooks.slack.com/services/T00/B00/XXX",
			"events":     []any{"deployment.completed"},
			"enabled":    false,
			"created_at": "2025-01-02T00:00:00+00:00",
			"updated_at": "2025-01-02T00:00:00+00:00",
		},
	},
	"meta": map[string]any{
		"current_page": 1,
		"last_page":    1,
		"total":        2,
	},
	"message":     "Webhooks retrieved successfully",
	"http_status": 200,
}

var webhookShowResponse = map[string]any{
	"data": map[string]any{
		"id":         "wh-001",
		"type":       "http",
		"url":        "https://example.com/webhook",
		"events":     []any{"site.created", "deployment.completed"},
		"enabled":    true,
		"created_at": "2025-01-01T00:00:00+00:00",
		"updated_at": "2025-01-05T00:00:00+00:00",
	},
	"message":     "Webhook retrieved successfully",
	"http_status": 200,
}

var webhookCreateHTTPResponse = map[string]any{
	"data": map[string]any{
		"id":         "wh-003",
		"type":       "http",
		"url":        "https://example.com/new-webhook",
		"events":     []any{"site.created", "deployment.completed"},
		"secret":     "a1b2c3d4e5f6789012345678901234567890",
		"enabled":    true,
		"created_at": "2025-01-15T00:00:00+00:00",
		"updated_at": "2025-01-15T00:00:00+00:00",
	},
	"message":     "Webhook created successfully.",
	"http_status": 201,
}

var webhookCreateSlackResponse = map[string]any{
	"data": map[string]any{
		"id":         "wh-004",
		"type":       "slack",
		"url":        "https://hooks.slack.com/services/T00/B00/XXX",
		"events":     []any{"deployment.completed"},
		"enabled":    true,
		"created_at": "2025-01-15T00:00:00+00:00",
		"updated_at": "2025-01-15T00:00:00+00:00",
	},
	"message":     "Slack webhook created successfully.",
	"http_status": 201,
}

var webhookUpdateResponse = map[string]any{
	"data": map[string]any{
		"id":         "wh-001",
		"type":       "http",
		"url":        "https://example.com/new-webhook",
		"events":     []any{"site.created", "site.updated"},
		"enabled":    false,
		"created_at": "2025-01-01T00:00:00+00:00",
		"updated_at": "2025-01-20T00:00:00+00:00",
	},
	"message":     "Webhook updated successfully",
	"http_status": 200,
}

var webhookDeleteResponse = map[string]any{
	"data": map[string]any{
		"id":         "wh-001",
		"type":       "http",
		"url":        "https://example.com/webhook",
		"events":     []any{"site.created", "deployment.completed"},
		"enabled":    true,
		"created_at": "2025-01-01T00:00:00+00:00",
		"updated_at": "2025-01-05T00:00:00+00:00",
	},
	"message":     "Webhook deleted successfully",
	"http_status": 200,
}

func newWebhookTestServer(validToken string) *httptest.Server {
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
		case method == "GET" && path == "/api/v1/vector/webhooks":
			_ = json.NewEncoder(w).Encode(webhookListResponse)

		case method == "GET" && path == "/api/v1/vector/webhooks/wh-001":
			_ = json.NewEncoder(w).Encode(webhookShowResponse)

		case method == "POST" && path == "/api/v1/vector/webhooks":
			var reqBody map[string]any
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &reqBody)
			webhookType, _ := reqBody["type"].(string)
			if webhookType == "slack" {
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(webhookCreateSlackResponse)
			} else {
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(webhookCreateHTTPResponse)
			}

		case method == "PUT" && path == "/api/v1/vector/webhooks/wh-001":
			_ = json.NewEncoder(w).Encode(webhookUpdateResponse)

		case method == "DELETE" && path == "/api/v1/vector/webhooks/wh-001":
			_ = json.NewEncoder(w).Encode(webhookDeleteResponse)

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Not Found",
				"http_status": 404,
			})
		}
	}))
}

func buildWebhookCmd(baseURL, token string, format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	root := &cobra.Command{
		Use: "vector",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			client := api.NewClient(baseURL, token, "test-agent")
			app := appctx.NewApp(
				config.DefaultConfig(),
				&config.Credentials{ApiKey: token},
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

	root.AddCommand(NewWebhookCmd())

	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

func buildWebhookCmdNoAuth(format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	root := &cobra.Command{
		Use: "vector",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			client := api.NewClient("http://localhost", "", "test-agent")
			app := appctx.NewApp(
				config.DefaultConfig(),
				&config.Credentials{},
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

	root.AddCommand(NewWebhookCmd())

	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

// --- Webhook List Tests ---

func TestWebhookListCmd_TableOutput(t *testing.T) {
	ts := newWebhookTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWebhookCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"webhook", "list"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "wh-001")
	assert.Contains(t, out, "http")
	assert.Contains(t, out, "https://example.com/webhook")
	assert.Contains(t, out, "Yes")
	assert.Contains(t, out, "wh-002")
	assert.Contains(t, out, "slack")
	assert.Contains(t, out, "No")
}

func TestWebhookListCmd_JSONOutput(t *testing.T) {
	ts := newWebhookTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWebhookCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"webhook", "list"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Len(t, result, 2)
	assert.Equal(t, "wh-001", result[0]["id"])
}

func TestWebhookListCmd_Pagination(t *testing.T) {
	var receivedPage, receivedPerPage string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPage = r.URL.Query().Get("page")
		receivedPerPage = r.URL.Query().Get("per_page")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(webhookListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildWebhookCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"webhook", "list", "--page", "3", "--per-page", "25"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "3", receivedPage)
	assert.Equal(t, "25", receivedPerPage)
}

func TestWebhookListCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(webhookListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildWebhookCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"webhook", "list"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "GET", receivedMethod)
	assert.Equal(t, "/api/v1/vector/webhooks", receivedPath)
}

func TestWebhookListCmd_AuthError(t *testing.T) {
	ts := newWebhookTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildWebhookCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"webhook", "list"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestWebhookListCmd_NoAuth(t *testing.T) {
	cmd, _, _ := buildWebhookCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"webhook", "list"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

// --- Webhook Show Tests ---

func TestWebhookShowCmd_TableOutput(t *testing.T) {
	ts := newWebhookTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWebhookCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"webhook", "show", "wh-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "wh-001")
	assert.Contains(t, out, "http")
	assert.Contains(t, out, "https://example.com/webhook")
	assert.Contains(t, out, "Yes")
	assert.Contains(t, out, "site.created, deployment.completed")
	assert.Contains(t, out, "2025-01-05T00:00:00+00:00")
}

func TestWebhookShowCmd_JSONOutput(t *testing.T) {
	ts := newWebhookTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWebhookCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"webhook", "show", "wh-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "wh-001", result["id"])
	assert.Equal(t, "http", result["type"])
}

func TestWebhookShowCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(webhookShowResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildWebhookCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"webhook", "show", "wh-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "GET", receivedMethod)
	assert.Equal(t, "/api/v1/vector/webhooks/wh-001", receivedPath)
}

func TestWebhookShowCmd_MissingArg(t *testing.T) {
	ts := newWebhookTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildWebhookCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"webhook", "show"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

// --- Webhook Create Tests ---

func TestWebhookCreateCmd_HTTPTableOutput(t *testing.T) {
	ts := newWebhookTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWebhookCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"webhook", "create", "--url", "https://example.com/new-webhook", "--events", "site.created,deployment.completed"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "wh-003")
	assert.Contains(t, out, "http")
	assert.Contains(t, out, "https://example.com/new-webhook")
	assert.Contains(t, out, "a1b2c3d4e5f6789012345678901234567890")
	assert.Contains(t, out, "Save this secret")
}

func TestWebhookCreateCmd_SlackTableOutput(t *testing.T) {
	ts := newWebhookTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWebhookCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"webhook", "create", "--url", "https://hooks.slack.com/services/T00/B00/XXX", "--events", "deployment.completed", "--type", "slack"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "wh-004")
	assert.Contains(t, out, "slack")
	assert.NotContains(t, out, "Save this secret")
}

func TestWebhookCreateCmd_JSONOutput(t *testing.T) {
	ts := newWebhookTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWebhookCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"webhook", "create", "--url", "https://example.com/new-webhook", "--events", "site.created,deployment.completed"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "wh-003", result["id"])
}

func TestWebhookCreateCmd_RequestBody(t *testing.T) {
	var receivedBody map[string]any
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(webhookCreateHTTPResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildWebhookCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"webhook", "create", "--url", "https://example.com/new-webhook", "--events", "site.created,deployment.completed"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/api/v1/vector/webhooks", receivedPath)
	assert.Equal(t, "https://example.com/new-webhook", receivedBody["url"])
	assert.Equal(t, "http", receivedBody["type"])
	events, ok := receivedBody["events"].([]any)
	require.True(t, ok)
	assert.Equal(t, []any{"site.created", "deployment.completed"}, events)
}

func TestWebhookCreateCmd_MissingRequiredFlags(t *testing.T) {
	ts := newWebhookTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildWebhookCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"webhook", "create"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestWebhookCreateCmd_MissingEventsFlag(t *testing.T) {
	ts := newWebhookTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildWebhookCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"webhook", "create", "--url", "https://example.com/webhook"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

// --- Webhook Update Tests ---

func TestWebhookUpdateCmd_TableOutput(t *testing.T) {
	ts := newWebhookTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWebhookCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"webhook", "update", "wh-001", "--url", "https://example.com/new-webhook"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "wh-001")
	assert.Contains(t, out, "https://example.com/new-webhook")
	assert.Contains(t, out, "2025-01-20T00:00:00+00:00")
}

func TestWebhookUpdateCmd_JSONOutput(t *testing.T) {
	ts := newWebhookTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWebhookCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"webhook", "update", "wh-001", "--url", "https://example.com/new-webhook"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "wh-001", result["id"])
}

func TestWebhookUpdateCmd_RequestBody(t *testing.T) {
	var receivedBody map[string]any
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(webhookUpdateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildWebhookCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"webhook", "update", "wh-001", "--url", "https://example.com/new-webhook"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "PUT", receivedMethod)
	assert.Equal(t, "/api/v1/vector/webhooks/wh-001", receivedPath)
	assert.Equal(t, "https://example.com/new-webhook", receivedBody["url"])
	// events and enabled should not be sent when not provided
	_, hasEvents := receivedBody["events"]
	assert.False(t, hasEvents)
	_, hasEnabled := receivedBody["enabled"]
	assert.False(t, hasEnabled)
}

func TestWebhookUpdateCmd_EventsFlag(t *testing.T) {
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(webhookUpdateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildWebhookCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"webhook", "update", "wh-001", "--events", "site.created,site.updated"})

	err := cmd.Execute()
	require.NoError(t, err)

	events, ok := receivedBody["events"].([]any)
	require.True(t, ok)
	assert.Equal(t, []any{"site.created", "site.updated"}, events)
}

func TestWebhookUpdateCmd_EnabledFlag(t *testing.T) {
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(webhookUpdateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildWebhookCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"webhook", "update", "wh-001", "--enabled=false"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, false, receivedBody["enabled"])
}

func TestWebhookUpdateCmd_MissingArg(t *testing.T) {
	ts := newWebhookTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildWebhookCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"webhook", "update"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

// --- Webhook Delete Tests ---

func TestWebhookDeleteCmd_TableOutput(t *testing.T) {
	ts := newWebhookTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWebhookCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"webhook", "delete", "wh-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Webhook deleted successfully")
}

func TestWebhookDeleteCmd_JSONOutput(t *testing.T) {
	ts := newWebhookTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWebhookCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"webhook", "delete", "wh-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "wh-001", result["id"])
}

func TestWebhookDeleteCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(webhookDeleteResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildWebhookCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"webhook", "delete", "wh-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "DELETE", receivedMethod)
	assert.Equal(t, "/api/v1/vector/webhooks/wh-001", receivedPath)
}

func TestWebhookDeleteCmd_MissingArg(t *testing.T) {
	ts := newWebhookTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildWebhookCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"webhook", "delete"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

// --- Help Tests ---

func TestWebhookCmd_Help(t *testing.T) {
	cmd := NewWebhookCmd()

	stdout := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "list")
	assert.Contains(t, out, "show")
	assert.Contains(t, out, "create")
	assert.Contains(t, out, "update")
	assert.Contains(t, out, "delete")
	assert.Contains(t, out, "webhooks")
}
