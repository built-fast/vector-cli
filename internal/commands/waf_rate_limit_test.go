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

var rateLimitListResponse = map[string]any{
	"data": []map[string]any{
		{
			"id":              float64(12345),
			"name":            "API Rate Limit",
			"description":     "Limit API requests to 100/second",
			"shield_zone_id":  float64(67890),
			"configuration": map[string]any{
				"request_count":   float64(100),
				"timeframe":       float64(1),
				"block_time":      float64(60),
				"value":           "/api/*",
				"action":          "rate-limit",
				"operator":        "begins-with",
				"variables":       []any{"request-uri"},
				"transformations": []any{"lowercase"},
			},
		},
		{
			"id":              float64(12346),
			"name":            "Login Rate Limit",
			"description":     "Limit login attempts",
			"shield_zone_id":  float64(67890),
			"configuration": map[string]any{
				"request_count":   float64(10),
				"timeframe":       float64(10),
				"block_time":      float64(300),
				"value":           "/login",
				"action":          "rate-limit",
				"operator":        "eq",
				"variables":       []any{"request-uri"},
				"transformations": []any{"lowercase", "url-decode"},
			},
		},
	},
	"message":     "Rate limits retrieved successfully",
	"http_status": 200,
}

var rateLimitShowResponse = map[string]any{
	"data": map[string]any{
		"id":              float64(12345),
		"name":            "API Rate Limit",
		"description":     "Limit API requests to 100/second",
		"shield_zone_id":  float64(67890),
		"configuration": map[string]any{
			"request_count":   float64(100),
			"timeframe":       float64(1),
			"block_time":      float64(60),
			"value":           "/api/*",
			"action":          "rate-limit",
			"operator":        "begins-with",
			"variables":       []any{"request-uri"},
			"transformations": []any{"lowercase"},
		},
	},
	"message":     "Rate limit retrieved successfully",
	"http_status": 200,
}

var rateLimitCreateResponse = map[string]any{
	"data": map[string]any{
		"id":              float64(12347),
		"name":            "New Rate Limit",
		"description":     "New rule description",
		"shield_zone_id":  float64(67890),
		"configuration": map[string]any{
			"request_count":   float64(50),
			"timeframe":       float64(10),
			"block_time":      float64(300),
			"value":           "/api/*",
			"action":          "rate-limit",
			"operator":        "begins-with",
			"variables":       []any{"request-uri"},
			"transformations": []any{"lowercase", "url-decode"},
		},
	},
	"message":     "Rate limit created successfully",
	"http_status": 201,
}

var rateLimitUpdateResponse = map[string]any{
	"data": map[string]any{
		"id":              float64(12345),
		"name":            "Updated Rate Limit",
		"description":     "Updated description",
		"shield_zone_id":  float64(67890),
		"configuration": map[string]any{
			"request_count":   float64(200),
			"timeframe":       float64(10),
			"block_time":      float64(300),
			"value":           "/api/v2/*",
			"action":          "rate-limit",
			"operator":        "regex",
			"variables":       []any{"request-uri", "query-string"},
			"transformations": []any{"lowercase"},
		},
	},
	"message":     "Rate limit updated successfully",
	"http_status": 200,
}

var rateLimitDeleteResponse = map[string]any{
	"data":        map[string]any{},
	"message":     "Rate limit deleted successfully",
	"http_status": 200,
}

func newWafRateLimitTestServer(validToken string) *httptest.Server {
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
		case method == "GET" && path == "/api/v1/vector/sites/site-001/waf/rate-limits":
			_ = json.NewEncoder(w).Encode(rateLimitListResponse)

		case method == "GET" && path == "/api/v1/vector/sites/site-001/waf/rate-limits/12345":
			_ = json.NewEncoder(w).Encode(rateLimitShowResponse)

		case method == "POST" && path == "/api/v1/vector/sites/site-001/waf/rate-limits":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(rateLimitCreateResponse)

		case method == "PUT" && path == "/api/v1/vector/sites/site-001/waf/rate-limits/12345":
			_ = json.NewEncoder(w).Encode(rateLimitUpdateResponse)

		case method == "DELETE" && path == "/api/v1/vector/sites/site-001/waf/rate-limits/12345":
			_ = json.NewEncoder(w).Encode(rateLimitDeleteResponse)

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Not Found",
				"http_status": 404,
			})
		}
	}))
}

func buildWafCmd(baseURL, token string, format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
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

	root.AddCommand(NewWafCmd())

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

func buildWafCmdNoAuth(format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
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

	root.AddCommand(NewWafCmd())

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

// --- Rate Limit List Tests ---

func TestWafRateLimitListCmd_TableOutput(t *testing.T) {
	ts := newWafRateLimitTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "rate-limit", "list", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "12345")
	assert.Contains(t, out, "API Rate Limit")
	assert.Contains(t, out, "100/1s")
	assert.Contains(t, out, "60s")
	assert.Contains(t, out, "12346")
	assert.Contains(t, out, "Login Rate Limit")
	assert.Contains(t, out, "10/10s")
	assert.Contains(t, out, "300s")
}

func TestWafRateLimitListCmd_JSONOutput(t *testing.T) {
	ts := newWafRateLimitTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"waf", "rate-limit", "list", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Len(t, result, 2)
	assert.Equal(t, float64(12345), result[0]["id"])
}

func TestWafRateLimitListCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rateLimitListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "rate-limit", "list", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "GET", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/waf/rate-limits", receivedPath)
}

func TestWafRateLimitListCmd_AuthError(t *testing.T) {
	ts := newWafRateLimitTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildWafCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"waf", "rate-limit", "list", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestWafRateLimitListCmd_NoAuth(t *testing.T) {
	cmd, _, _ := buildWafCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"waf", "rate-limit", "list", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestWafRateLimitListCmd_MissingArg(t *testing.T) {
	ts := newWafRateLimitTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "rate-limit", "list"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

// --- Rate Limit Show Tests ---

func TestWafRateLimitShowCmd_TableOutput(t *testing.T) {
	ts := newWafRateLimitTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "rate-limit", "show", "site-001", "12345"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "12345")
	assert.Contains(t, out, "API Rate Limit")
	assert.Contains(t, out, "Limit API requests to 100/second")
	assert.Contains(t, out, "100")
	assert.Contains(t, out, "60")
	assert.Contains(t, out, "/api/*")
	assert.Contains(t, out, "begins-with")
	assert.Contains(t, out, "request-uri")
	assert.Contains(t, out, "lowercase")
}

func TestWafRateLimitShowCmd_JSONOutput(t *testing.T) {
	ts := newWafRateLimitTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"waf", "rate-limit", "show", "site-001", "12345"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, float64(12345), result["id"])
	assert.Equal(t, "API Rate Limit", result["name"])
}

func TestWafRateLimitShowCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rateLimitShowResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "rate-limit", "show", "site-001", "12345"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "GET", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/waf/rate-limits/12345", receivedPath)
}

func TestWafRateLimitShowCmd_MissingArg(t *testing.T) {
	ts := newWafRateLimitTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "rate-limit", "show", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 2 arg(s)")
}

// --- Rate Limit Create Tests ---

func TestWafRateLimitCreateCmd_TableOutput(t *testing.T) {
	ts := newWafRateLimitTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "rate-limit", "create", "site-001",
		"--name", "New Rate Limit",
		"--request-count", "50",
		"--timeframe", "10",
		"--block-time", "300",
		"--description", "New rule description",
		"--value", "/api/*",
		"--operator", "begins-with",
		"--variables", "request-uri",
		"--transformations", "lowercase,url-decode",
	})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "12347")
	assert.Contains(t, out, "New Rate Limit")
	assert.Contains(t, out, "New rule description")
	assert.Contains(t, out, "50")
	assert.Contains(t, out, "300")
	assert.Contains(t, out, "/api/*")
	assert.Contains(t, out, "begins-with")
	assert.Contains(t, out, "request-uri")
	assert.Contains(t, out, "lowercase, url-decode")
}

func TestWafRateLimitCreateCmd_JSONOutput(t *testing.T) {
	ts := newWafRateLimitTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"waf", "rate-limit", "create", "site-001",
		"--name", "New Rate Limit",
		"--request-count", "50",
		"--timeframe", "10",
		"--block-time", "300",
	})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, float64(12347), result["id"])
}

func TestWafRateLimitCreateCmd_RequestBody(t *testing.T) {
	var receivedBody map[string]any
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(rateLimitCreateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "rate-limit", "create", "site-001",
		"--name", "New Rate Limit",
		"--request-count", "50",
		"--timeframe", "10",
		"--block-time", "300",
		"--description", "New rule description",
		"--variables", "request-uri",
		"--transformations", "lowercase,url-decode",
	})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/waf/rate-limits", receivedPath)
	assert.Equal(t, "New Rate Limit", receivedBody["name"])
	assert.Equal(t, float64(50), receivedBody["request_count"])
	assert.Equal(t, float64(10), receivedBody["timeframe"])
	assert.Equal(t, float64(300), receivedBody["block_time"])
	assert.Equal(t, "New rule description", receivedBody["description"])
	vars, ok := receivedBody["variables"].([]any)
	require.True(t, ok)
	assert.Equal(t, []any{"request-uri"}, vars)
	trans, ok := receivedBody["transformations"].([]any)
	require.True(t, ok)
	assert.Equal(t, []any{"lowercase", "url-decode"}, trans)
}

func TestWafRateLimitCreateCmd_MissingRequiredFlags(t *testing.T) {
	ts := newWafRateLimitTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "rate-limit", "create", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestWafRateLimitCreateCmd_MissingArg(t *testing.T) {
	ts := newWafRateLimitTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "rate-limit", "create"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

// --- Rate Limit Update Tests ---

func TestWafRateLimitUpdateCmd_TableOutput(t *testing.T) {
	ts := newWafRateLimitTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "rate-limit", "update", "site-001", "12345",
		"--name", "Updated Rate Limit",
		"--request-count", "200",
	})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "12345")
	assert.Contains(t, out, "Updated Rate Limit")
	assert.Contains(t, out, "200")
}

func TestWafRateLimitUpdateCmd_JSONOutput(t *testing.T) {
	ts := newWafRateLimitTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"waf", "rate-limit", "update", "site-001", "12345",
		"--name", "Updated Rate Limit",
	})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, float64(12345), result["id"])
}

func TestWafRateLimitUpdateCmd_RequestBody(t *testing.T) {
	var receivedBody map[string]any
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rateLimitUpdateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "rate-limit", "update", "site-001", "12345",
		"--name", "Updated Rate Limit",
		"--request-count", "200",
	})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "PUT", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/waf/rate-limits/12345", receivedPath)
	assert.Equal(t, "Updated Rate Limit", receivedBody["name"])
	assert.Equal(t, float64(200), receivedBody["request_count"])
	// Flags not provided should not be sent
	_, hasTimeframe := receivedBody["timeframe"]
	assert.False(t, hasTimeframe)
	_, hasBlockTime := receivedBody["block_time"]
	assert.False(t, hasBlockTime)
}

func TestWafRateLimitUpdateCmd_VariablesFlag(t *testing.T) {
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rateLimitUpdateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "rate-limit", "update", "site-001", "12345",
		"--variables", "request-uri,query-string",
	})

	err := cmd.Execute()
	require.NoError(t, err)

	vars, ok := receivedBody["variables"].([]any)
	require.True(t, ok)
	assert.Equal(t, []any{"request-uri", "query-string"}, vars)
}

func TestWafRateLimitUpdateCmd_MissingArg(t *testing.T) {
	ts := newWafRateLimitTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "rate-limit", "update", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 2 arg(s)")
}

// --- Rate Limit Delete Tests ---

func TestWafRateLimitDeleteCmd_TableOutput(t *testing.T) {
	ts := newWafRateLimitTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "rate-limit", "delete", "site-001", "12345"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Rate limit rule deleted successfully")
}

func TestWafRateLimitDeleteCmd_JSONOutput(t *testing.T) {
	ts := newWafRateLimitTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"waf", "rate-limit", "delete", "site-001", "12345"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
}

func TestWafRateLimitDeleteCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rateLimitDeleteResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "rate-limit", "delete", "site-001", "12345"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "DELETE", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/waf/rate-limits/12345", receivedPath)
}

func TestWafRateLimitDeleteCmd_MissingArg(t *testing.T) {
	ts := newWafRateLimitTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "rate-limit", "delete", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 2 arg(s)")
}

// --- Help Tests ---

func TestWafCmd_Help(t *testing.T) {
	cmd := NewWafCmd()

	stdout := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "rate-limit")
	assert.Contains(t, out, "WAF")
}

func TestWafRateLimitCmd_Help(t *testing.T) {
	cmd := NewWafRateLimitCmd()

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
	assert.Contains(t, out, "rate limit")
}
