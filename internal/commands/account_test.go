package commands

import (
	"bytes"
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

var accountShowResponse = map[string]any{
	"data": map[string]any{
		"owner": map[string]any{
			"name":  "John Doe",
			"email": "user@example.com",
		},
		"account": map[string]any{
			"name":    "My Account",
			"company": "Acme Corp",
		},
		"cluster": map[string]any{
			"alb_dns_name":             "alb-abc123.us-west-2.elb.amazonaws.com",
			"aurora_cluster_endpoint":  "cluster.abc123.us-east-1.rds.amazonaws.com",
			"ssh_nlb_dns":              "nlb-abc123.us-west-2.elb.amazonaws.com",
		},
		"domains": []any{"example.com", "example.org"},
		"sites": map[string]any{
			"total": float64(5),
			"by_status": map[string]any{
				"pending":                float64(0),
				"activation_requested":   float64(0),
				"active":                 float64(3),
				"suspension_requested":   float64(0),
				"suspended":              float64(1),
				"unsuspension_requested": float64(0),
				"termination_requested":  float64(0),
				"terminated":             float64(1),
				"canceled":               float64(0),
			},
		},
		"environments": map[string]any{
			"total": float64(8),
			"by_status": map[string]any{
				"pending":      float64(1),
				"provisioning": float64(0),
				"active":       float64(5),
				"suspending":   float64(0),
				"suspended":    float64(1),
				"unsuspending": float64(0),
				"terminating":  float64(0),
				"terminated":   float64(1),
				"failed":       float64(0),
			},
		},
	},
	"message":     "Account summary retrieved successfully",
	"http_status": 200,
}

func newAccountTestServer(validToken string) *httptest.Server {
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

		if r.Method == "GET" && r.URL.Path == "/api/v1/vector/account" {
			_ = json.NewEncoder(w).Encode(accountShowResponse)
		} else {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Not Found",
				"http_status": 404,
			})
		}
	}))
}

func buildAccountCmd(baseURL, token string, format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
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

	accountCmd := NewAccountCmd()
	root.AddCommand(accountCmd)

	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

func buildAccountCmdNoAuth(format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
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

	accountCmd := NewAccountCmd()
	root.AddCommand(accountCmd)

	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

// --- Account Show Tests ---

func TestAccountShowCmd_TableOutput(t *testing.T) {
	ts := newAccountTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "show"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "John Doe")
	assert.Contains(t, out, "user@example.com")
	assert.Contains(t, out, "My Account")
	assert.Contains(t, out, "Acme Corp")
	assert.Contains(t, out, "5")
	assert.Contains(t, out, "3")
	assert.Contains(t, out, "8")
}

func TestAccountShowCmd_JSONOutput(t *testing.T) {
	ts := newAccountTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"account", "show"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	owner := result["owner"].(map[string]any)
	assert.Equal(t, "John Doe", owner["name"])
	assert.Equal(t, "user@example.com", owner["email"])
	account := result["account"].(map[string]any)
	assert.Equal(t, "My Account", account["name"])
}

func TestAccountShowCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(accountShowResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "show"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "GET", receivedMethod)
	assert.Equal(t, "/api/v1/vector/account", receivedPath)
}

func TestAccountShowCmd_AuthError(t *testing.T) {
	ts := newAccountTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildAccountCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"account", "show"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestAccountShowCmd_NoAuthToken(t *testing.T) {
	cmd, _, _ := buildAccountCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"account", "show"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

// --- Help Text Tests ---

func TestAccountCmd_HelpText(t *testing.T) {
	ts := newAccountTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "show")
	assert.Contains(t, out, "account-level resources")
}

func TestAccountShowCmd_HelpText(t *testing.T) {
	ts := newAccountTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAccountCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"account", "show", "--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "account details")
}
