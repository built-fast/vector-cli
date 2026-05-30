package commands

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/itchyny/gojq"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/appctx"
	"github.com/built-fast/vector-cli/internal/config"
	"github.com/built-fast/vector-cli/internal/output"
)

// mustCompileJQ parses and compiles a jq expression for use in tests.
func mustCompileJQ(t *testing.T, expr string) *gojq.Code {
	t.Helper()
	query, err := gojq.Parse(expr)
	require.NoError(t, err)
	code, err := gojq.Compile(query)
	require.NoError(t, err)
	return code
}

var apiSitesResponse = map[string]any{
	"data": []map[string]any{
		{"id": 1, "name": "example.com"},
		{"id": 2, "name": "example.org"},
	},
	"meta":        map[string]any{"current_page": 1, "last_page": 1, "total": 2},
	"http_status": 200,
}

func newAPITestServer(validToken string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+validToken {
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
		case r.Method == "GET" && r.URL.Path == "/api/v1/vector/sites":
			_ = json.NewEncoder(w).Encode(apiSitesResponse)
		case r.Method == "GET" && r.URL.Path == "/api/v1/vector/raw":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("plain text body, not json"))
		case r.Method == "GET" && r.URL.Path == "/api/v1/vector/missing":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Site not found",
				"http_status": 404,
			})
		case r.Method == "GET" && r.URL.Path == "/api/v1/vector/boom":
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Internal Server Error",
				"http_status": 500,
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

func buildAPICmd(baseURL, token string, format output.Format, opts ...output.WriterOption) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
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
			app.Output = output.NewWriter(stdout, format, opts...)
			cmd.SetContext(appctx.WithApp(cmd.Context(), app))
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(NewAPICmd())

	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

func buildAPICmdNoAuth(format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
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

	root.AddCommand(NewAPICmd())

	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

// --- Path resolution ---

func TestResolveAPIPath(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     string
	}{
		{"bare resource", "sites", "/api/v1/vector/sites"},
		{"bare nested resource", "sites/123/environments", "/api/v1/vector/sites/123/environments"},
		{"leading slash verbatim", "/api/v1/vector/sites", "/api/v1/vector/sites"},
		{"leading slash non-vector path", "/account", "/account"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, resolveAPIPath(tt.endpoint))
		})
	}
}

func TestAPICmd_BarePathPrependsBase(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(apiSitesResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildAPICmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"api", "sites"})

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "GET", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites", receivedPath)
}

func TestAPICmd_LeadingSlashPathSentVerbatim(t *testing.T) {
	var receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(apiSitesResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildAPICmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"api", "/account"})

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "/account", receivedPath)
}

// --- Output ---

func TestAPICmd_PrettyPrintsJSON(t *testing.T) {
	ts := newAPITestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAPICmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"api", "sites"})

	require.NoError(t, cmd.Execute())

	out := stdout.String()
	// Pretty-printed JSON is indented and preserves the full envelope.
	assert.Contains(t, out, "\n  ")
	assert.Contains(t, out, "\"data\"")
	assert.Contains(t, out, "\"meta\"")

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Contains(t, result, "data")
	assert.Contains(t, result, "meta")
}

func TestAPICmd_RawBodyWhenNotJSON(t *testing.T) {
	ts := newAPITestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAPICmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"api", "raw"})

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "plain text body, not json", stdout.String())
}

func TestAPICmd_JQFiltersFullEnvelope(t *testing.T) {
	ts := newAPITestServer("valid-token")
	defer ts.Close()

	code := mustCompileJQ(t, ".data[].id")
	cmd, stdout, _ := buildAPICmd(ts.URL, "valid-token", output.JSON, output.WithJQ(".data[].id", code))
	cmd.SetArgs([]string{"api", "sites"})

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "1\n2\n", stdout.String())
}

// --- Errors ---

func TestAPICmd_NotFoundExitCode(t *testing.T) {
	ts := newAPITestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildAPICmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"api", "missing"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 4, apiErr.ExitCode)
	assert.Contains(t, apiErr.Error(), "Site not found")
}

func TestAPICmd_ServerErrorExitCode(t *testing.T) {
	ts := newAPITestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildAPICmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"api", "boom"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 5, apiErr.ExitCode)
}

func TestAPICmd_AuthError(t *testing.T) {
	ts := newAPITestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildAPICmd(ts.URL, "bad-token", output.JSON)
	cmd.SetArgs([]string{"api", "sites"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestAPICmd_NoAuthToken(t *testing.T) {
	cmd, _, _ := buildAPICmdNoAuth(output.JSON)
	cmd.SetArgs([]string{"api", "sites"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestAPICmd_RequiresEndpointArg(t *testing.T) {
	ts := newAPITestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildAPICmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"api"})

	require.Error(t, cmd.Execute())
}

func TestAPICmd_HelpText(t *testing.T) {
	ts := newAPITestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAPICmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"api", "--help"})

	require.NoError(t, cmd.Execute())

	out := stdout.String()
	assert.Contains(t, out, "api <endpoint>")
	assert.Contains(t, out, "Vector Pro API")
}
