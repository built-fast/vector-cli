package commands

import (
	"bytes"
	"context"
	"encoding/json"
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

var eventListResponse = map[string]any{
	"data": []map[string]any{
		{
			"id":         "evt-001",
			"event":      "site.provisioning.completed",
			"model_type": "VectorSite",
			"model_id":   "site-001",
			"context":    nil,
			"actor": map[string]any{
				"id":         1,
				"ip":         "192.0.2.1",
				"token_id":   1,
				"token_name": "Production API Key",
			},
			"occurred_at": "2025-01-15T12:00:00+00:00",
			"created_at":  "2025-01-15T12:00:00+00:00",
		},
		{
			"id":         "evt-002",
			"event":      "deployment.completed",
			"model_type": "VectorEnvironment",
			"model_id":   "",
			"context":    nil,
			"actor": map[string]any{
				"id":         2,
				"ip":         "10.0.0.1",
				"token_id":   nil,
				"token_name": "",
			},
			"occurred_at": "2025-01-15T13:00:00+00:00",
			"created_at":  "2025-01-15T13:00:00+00:00",
		},
		{
			"id":         "evt-003",
			"event":      "site.deleted",
			"model_type": "",
			"model_id":   "",
			"context":    nil,
			"actor":      nil,
			"occurred_at": "2025-01-15T14:00:00+00:00",
			"created_at":  "2025-01-15T14:00:00+00:00",
		},
	},
	"meta": map[string]any{
		"current_page": 1,
		"last_page":    1,
		"total":        3,
	},
	"message":     "Event logs retrieved successfully",
	"http_status": 200,
}

func newEventTestServer(validToken string) *httptest.Server {
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

		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v1/vector/events":
			_ = json.NewEncoder(w).Encode(eventListResponse)
		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Not Found",
				"http_status": 404,
			})
		}
	}))
}

func buildEventCmd(baseURL, token string, format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	stdout := new(bytes.Buffer)

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

	root.AddCommand(NewEventCmd())

	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

func buildEventCmdNoAuth(format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	stdout := new(bytes.Buffer)

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

	root.AddCommand(NewEventCmd())

	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

// --- Event List Tests ---

func TestEventListCmd_TableOutput(t *testing.T) {
	ts := newEventTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEventCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"event", "list"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	// First event: has token name
	assert.Contains(t, out, "evt-001")
	assert.Contains(t, out, "site.provisioning.completed")
	assert.Contains(t, out, "Production API Key")
	assert.Contains(t, out, "VectorSite:site-001")

	// Second event: falls back to IP
	assert.Contains(t, out, "evt-002")
	assert.Contains(t, out, "10.0.0.1")
	assert.Contains(t, out, "VectorEnvironment")

	// Third event: no actor, no resource
	assert.Contains(t, out, "evt-003")
}

func TestEventListCmd_JSONOutput(t *testing.T) {
	ts := newEventTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEventCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"event", "list"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Len(t, result, 3)
	assert.Equal(t, "evt-001", result[0]["id"])
}

func TestEventListCmd_Pagination(t *testing.T) {
	var receivedPage, receivedPerPage string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPage = r.URL.Query().Get("page")
		receivedPerPage = r.URL.Query().Get("per_page")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(eventListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildEventCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"event", "list", "--page", "2", "--per-page", "10"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "2", receivedPage)
	assert.Equal(t, "10", receivedPerPage)
}

func TestEventListCmd_FilterFlags(t *testing.T) {
	var receivedFrom, receivedTo, receivedEvent string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedFrom = r.URL.Query().Get("from")
		receivedTo = r.URL.Query().Get("to")
		receivedEvent = r.URL.Query().Get("event")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(eventListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildEventCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"event", "list",
		"--from", "2025-01-01T00:00:00+00:00",
		"--to", "2025-01-31T23:59:59+00:00",
		"--event", "site.provisioning.completed,deployment.completed",
	})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "2025-01-01T00:00:00+00:00", receivedFrom)
	assert.Equal(t, "2025-01-31T23:59:59+00:00", receivedTo)
	assert.Equal(t, "site.provisioning.completed,deployment.completed", receivedEvent)
}

func TestEventListCmd_AuthError(t *testing.T) {
	ts := newEventTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildEventCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"event", "list"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestEventListCmd_NoAuth(t *testing.T) {
	cmd, _, _ := buildEventCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"event", "list"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

// --- Help Text Tests ---

func TestEventCmd_Help(t *testing.T) {
	cmd := NewEventCmd()
	cmd.SetContext(context.Background())

	stdout := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "list")
	assert.Contains(t, out, "View account event logs")
}

func TestEventListCmd_Help(t *testing.T) {
	cmd := NewEventCmd()
	cmd.SetContext(context.Background())

	stdout := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetArgs([]string{"list", "--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "Retrieve a paginated list of account event logs")
	assert.Contains(t, out, "--from")
	assert.Contains(t, out, "--to")
	assert.Contains(t, out, "--event")
	assert.Contains(t, out, "--page")
	assert.Contains(t, out, "--per-page")
}

// --- Actor Formatting Tests ---

func TestFormatActor_TokenName(t *testing.T) {
	item := map[string]any{
		"actor": map[string]any{
			"token_name": "My Token",
			"ip":         "192.168.1.1",
		},
	}
	assert.Equal(t, "My Token", formatActor(item))
}

func TestFormatActor_FallbackToIP(t *testing.T) {
	item := map[string]any{
		"actor": map[string]any{
			"token_name": "",
			"ip":         "192.168.1.1",
		},
	}
	assert.Equal(t, "192.168.1.1", formatActor(item))
}

func TestFormatActor_NilActor(t *testing.T) {
	item := map[string]any{
		"actor": nil,
	}
	assert.Equal(t, "-", formatActor(item))
}

func TestFormatActor_NoActor(t *testing.T) {
	item := map[string]any{}
	assert.Equal(t, "-", formatActor(item))
}

// --- Resource Formatting Tests ---

func TestFormatResource_WithModelID(t *testing.T) {
	item := map[string]any{
		"model_type": "VectorSite",
		"model_id":   "site-001",
	}
	assert.Equal(t, "VectorSite:site-001", formatResource(item))
}

func TestFormatResource_WithoutModelID(t *testing.T) {
	item := map[string]any{
		"model_type": "VectorEnvironment",
		"model_id":   "",
	}
	assert.Equal(t, "VectorEnvironment", formatResource(item))
}

func TestFormatResource_NoModelType(t *testing.T) {
	item := map[string]any{
		"model_type": "",
		"model_id":   "",
	}
	assert.Equal(t, "-", formatResource(item))
}
