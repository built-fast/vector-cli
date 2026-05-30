package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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

// --- Method selection & request body (US-003) ---

// captured records what an echo test server received.
type captured struct {
	method      string
	path        string
	rawQuery    string
	contentType string
	body        []byte
}

// newAPIEchoServer returns a server that records the request and echoes a
// minimal JSON envelope, capturing the request into c.
func newAPIEchoServer(t *testing.T, validToken string, c *captured) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+validToken {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "Unauthenticated.", "http_status": 401})
			return
		}

		c.method = r.Method
		c.path = r.URL.Path
		c.rawQuery = r.URL.RawQuery
		c.contentType = r.Header.Get("Content-Type")
		c.body, _ = io.ReadAll(r.Body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"ok": true}, "http_status": 201})
	}))
}

func TestAPICmd_MethodOverride(t *testing.T) {
	var c captured
	ts := newAPIEchoServer(t, "valid-token", &c)
	defer ts.Close()

	cmd, _, _ := buildAPICmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"api", "sites/1", "--method", "delete"})

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "DELETE", c.method)
	assert.Equal(t, "/api/v1/vector/sites/1", c.path)
}

func TestAPICmd_AutoPOSTWhenFieldsGiven(t *testing.T) {
	var c captured
	ts := newAPIEchoServer(t, "valid-token", &c)
	defer ts.Close()

	cmd, _, _ := buildAPICmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"api", "sites", "-f", "customer_id=cust_1"})

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "POST", c.method)
	assert.Equal(t, "application/json", c.contentType)
}

func TestAPICmd_AutoPOSTWhenInputGiven(t *testing.T) {
	var c captured
	ts := newAPIEchoServer(t, "valid-token", &c)
	defer ts.Close()

	dir := t.TempDir()
	bodyFile := filepath.Join(dir, "body.json")
	require.NoError(t, os.WriteFile(bodyFile, []byte(`{"a":1}`), 0o600))

	cmd, _, _ := buildAPICmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"api", "sites", "--input", bodyFile})

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "POST", c.method)
}

func TestAPICmd_RawFieldIsString(t *testing.T) {
	var c captured
	ts := newAPIEchoServer(t, "valid-token", &c)
	defer ts.Close()

	cmd, _, _ := buildAPICmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"api", "sites", "-X", "POST", "-f", "count=42", "-f", "flag=true"})

	require.NoError(t, cmd.Execute())
	assert.JSONEq(t, `{"count":"42","flag":"true"}`, string(c.body))
}

func TestAPICmd_TypedFieldCoercion(t *testing.T) {
	var c captured
	ts := newAPIEchoServer(t, "valid-token", &c)
	defer ts.Close()

	cmd, _, _ := buildAPICmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{
		"api", "sites", "-X", "POST",
		"-F", "count=42",
		"-F", "ratio=1.5",
		"-F", "flag=true",
		"-F", "off=false",
		"-F", "empty=null",
		"-F", "name=hello",
	})

	require.NoError(t, cmd.Execute())
	assert.JSONEq(t, `{"count":42,"ratio":1.5,"flag":true,"off":false,"empty":null,"name":"hello"}`, string(c.body))
}

func TestAPICmd_TypedFieldFromFile(t *testing.T) {
	var c captured
	ts := newAPIEchoServer(t, "valid-token", &c)
	defer ts.Close()

	dir := t.TempDir()
	valueFile := filepath.Join(dir, "value.txt")
	require.NoError(t, os.WriteFile(valueFile, []byte("from-file"), 0o600))

	cmd, _, _ := buildAPICmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"api", "sites", "-X", "POST", "-F", "note=@" + valueFile})

	require.NoError(t, cmd.Execute())
	assert.JSONEq(t, `{"note":"from-file"}`, string(c.body))
}

func TestAPICmd_TypedFieldFromStdin(t *testing.T) {
	var c captured
	ts := newAPIEchoServer(t, "valid-token", &c)
	defer ts.Close()

	orig := apiStdinReader
	apiStdinReader = strings.NewReader("from-stdin")
	t.Cleanup(func() { apiStdinReader = orig })

	cmd, _, _ := buildAPICmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"api", "sites", "-X", "POST", "-F", "note=@-"})

	require.NoError(t, cmd.Execute())
	assert.JSONEq(t, `{"note":"from-stdin"}`, string(c.body))
}

func TestAPICmd_FieldsAsQueryForGET(t *testing.T) {
	var c captured
	ts := newAPIEchoServer(t, "valid-token", &c)
	defer ts.Close()

	cmd, _, _ := buildAPICmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"api", "sites", "-X", "GET", "-f", "status=active", "-F", "page=2"})

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "GET", c.method)
	assert.Empty(t, c.body)

	values, err := url.ParseQuery(c.rawQuery)
	require.NoError(t, err)
	assert.Equal(t, "active", values.Get("status"))
	assert.Equal(t, "2", values.Get("page"))
}

func TestAPICmd_ReusedScalarKeyIsError(t *testing.T) {
	var c captured
	ts := newAPIEchoServer(t, "valid-token", &c)
	defer ts.Close()

	cmd, _, _ := buildAPICmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"api", "sites", "-X", "POST", "-f", "name=a", "-F", "name=b"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 3, apiErr.ExitCode)
	assert.Contains(t, apiErr.Error(), `under "name"`)
	assert.Empty(t, c.method, "request should not have been sent")
}

func TestAPICmd_ArrayKeyAppends(t *testing.T) {
	var c captured
	ts := newAPIEchoServer(t, "valid-token", &c)
	defer ts.Close()

	cmd, _, _ := buildAPICmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"api", "sites", "-X", "POST", "-F", "tag[]=a", "-F", "tag[]=b"})

	require.NoError(t, cmd.Execute())
	assert.JSONEq(t, `{"tag":["a","b"]}`, string(c.body))
}

func TestAPICmd_ArrayThenScalarSameKeyIsError(t *testing.T) {
	ts := newAPITestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildAPICmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"api", "sites", "-X", "POST", "-f", "tag[]=a", "-f", "tag=b"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 3, apiErr.ExitCode)
}

func TestAPICmd_InputFromFile(t *testing.T) {
	var c captured
	ts := newAPIEchoServer(t, "valid-token", &c)
	defer ts.Close()

	dir := t.TempDir()
	bodyFile := filepath.Join(dir, "body.json")
	require.NoError(t, os.WriteFile(bodyFile, []byte(`{"customer_id":"cust_9"}`), 0o600))

	cmd, _, _ := buildAPICmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"api", "sites", "-X", "POST", "--input", bodyFile})

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "application/json", c.contentType)
	assert.JSONEq(t, `{"customer_id":"cust_9"}`, string(c.body))
}

func TestAPICmd_InputFromStdin(t *testing.T) {
	var c captured
	ts := newAPIEchoServer(t, "valid-token", &c)
	defer ts.Close()

	orig := apiStdinReader
	apiStdinReader = strings.NewReader(`{"customer_id":"cust_stdin"}`)
	t.Cleanup(func() { apiStdinReader = orig })

	cmd, _, _ := buildAPICmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"api", "sites", "-X", "POST", "--input", "-"})

	require.NoError(t, cmd.Execute())
	assert.JSONEq(t, `{"customer_id":"cust_stdin"}`, string(c.body))
}

func TestAPICmd_InputMissingFileError(t *testing.T) {
	ts := newAPITestServer("valid-token")
	defer ts.Close()

	missing := filepath.Join(t.TempDir(), "nope.json")

	cmd, _, _ := buildAPICmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"api", "sites", "-X", "POST", "--input", missing})

	err := cmd.Execute()
	require.Error(t, err)

	// A missing input file is a general error (exit code 1), not an *api.APIError.
	var apiErr *api.APIError
	assert.False(t, errors.As(err, &apiErr), "missing file should be a general error, not an APIError")
}

func TestAPICmd_InputAndFieldsMutuallyExclusive(t *testing.T) {
	ts := newAPITestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildAPICmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"api", "sites", "--input", "-", "-f", "name=a"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 3, apiErr.ExitCode)
}

func TestAPICmd_InvalidFieldFormat(t *testing.T) {
	ts := newAPITestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildAPICmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"api", "sites", "-X", "POST", "-f", "noequals"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 3, apiErr.ExitCode)
}

// --- collectFields (unit) ---

func TestCollectFields(t *testing.T) {
	got, err := collectFields(
		[]string{"name=alice"},
		[]string{"age=30", "active=true", "tag[]=x", "tag[]=y"},
	)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"name":   "alice",
		"age":    int64(30),
		"active": true,
		"tag":    []any{"x", "y"},
	}, got)
}

func TestCollectFields_ReusedScalarKey(t *testing.T) {
	_, err := collectFields(nil, []string{"k=1", "k=2"})
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 3, apiErr.ExitCode)
}
