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

var phpVersionsResponse = map[string]any{
	"data": []map[string]any{
		{"value": "7.4", "label": "PHP 7.4"},
		{"value": "8.0", "label": "PHP 8.0"},
		{"value": "8.1", "label": "PHP 8.1"},
		{"value": "8.2", "label": "PHP 8.2"},
		{"value": "8.3", "label": "PHP 8.3"},
	},
	"http_status": 200,
}

func newPHPVersionsTestServer(validToken string) *httptest.Server {
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

		if r.Method == "GET" && r.URL.Path == "/api/v1/vector/php-versions" {
			_ = json.NewEncoder(w).Encode(phpVersionsResponse)
		} else {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Not Found",
				"http_status": 404,
			})
		}
	}))
}

func buildPHPVersionsCmd(baseURL, token string, format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
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
			cmd.SetContext(appctx.WithApp(cmd.Context(), app))
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(NewPHPVersionsCmd())

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

func buildPHPVersionsCmdNoAuth(format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
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
			cmd.SetContext(appctx.WithApp(cmd.Context(), app))
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(NewPHPVersionsCmd())

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

// --- PHP Versions Tests ---

func TestPHPVersionsCmd_TableOutput(t *testing.T) {
	ts := newPHPVersionsTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildPHPVersionsCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"php-versions"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "VERSION")
	assert.Contains(t, out, "7.4")
	assert.Contains(t, out, "8.0")
	assert.Contains(t, out, "8.1")
	assert.Contains(t, out, "8.2")
	assert.Contains(t, out, "8.3")
}

func TestPHPVersionsCmd_JSONOutput(t *testing.T) {
	ts := newPHPVersionsTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildPHPVersionsCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"php-versions"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Len(t, result, 5)
	assert.Equal(t, "7.4", result[0]["value"])
	assert.Equal(t, "8.3", result[4]["value"])
}

func TestPHPVersionsCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(phpVersionsResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildPHPVersionsCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"php-versions"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "GET", receivedMethod)
	assert.Equal(t, "/api/v1/vector/php-versions", receivedPath)
}

func TestPHPVersionsCmd_AuthError(t *testing.T) {
	ts := newPHPVersionsTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildPHPVersionsCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"php-versions"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestPHPVersionsCmd_NoAuthToken(t *testing.T) {
	cmd, _, _ := buildPHPVersionsCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"php-versions"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestPHPVersionsCmd_HelpText(t *testing.T) {
	ts := newPHPVersionsTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildPHPVersionsCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"php-versions", "--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "php-versions")
	assert.Contains(t, out, "available PHP versions")
}
