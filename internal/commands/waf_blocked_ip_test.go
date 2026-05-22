package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/output"
)

var blockedIPListResponse = map[string]any{
	"data": []map[string]any{
		{"ip": "192.168.1.100"},
		{"ip": "10.0.0.50"},
	},
	"message":     "Blocked IPs retrieved successfully",
	"http_status": 200,
}

var blockedIPAddResponse = map[string]any{
	"data":        map[string]any{"ip": "192.168.1.100"},
	"message":     "IP added to blocklist",
	"http_status": 201,
}

var blockedIPRemoveResponse = map[string]any{
	"data":        map[string]any{},
	"message":     "IP removed from blocklist",
	"http_status": 200,
}

func newWafBlockedIPTestServer(validToken string) *httptest.Server {
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
		case method == "GET" && path == "/api/v1/vector/sites/site-001/waf/blocked-ips":
			_ = json.NewEncoder(w).Encode(blockedIPListResponse)

		case method == "POST" && path == "/api/v1/vector/sites/site-001/waf/blocked-ips":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(blockedIPAddResponse)

		case method == "DELETE" && path == "/api/v1/vector/sites/site-001/waf/blocked-ips/192.168.1.100":
			_ = json.NewEncoder(w).Encode(blockedIPRemoveResponse)

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Not Found",
				"http_status": 404,
			})
		}
	}))
}

// --- Blocked IP List Tests ---

func TestWafBlockedIPListCmd_TableOutput(t *testing.T) {
	ts := newWafBlockedIPTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "blocked-ip", "list", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "192.168.1.100")
	assert.Contains(t, out, "10.0.0.50")
	assert.Contains(t, out, "IP")
}

func TestWafBlockedIPListCmd_JSONOutput(t *testing.T) {
	ts := newWafBlockedIPTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"waf", "blocked-ip", "list", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Len(t, result, 2)
	assert.Equal(t, "192.168.1.100", result[0]["ip"])
}

func TestWafBlockedIPListCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(blockedIPListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "blocked-ip", "list", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "GET", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/waf/blocked-ips", receivedPath)
}

func TestWafBlockedIPListCmd_AuthError(t *testing.T) {
	ts := newWafBlockedIPTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildWafCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"waf", "blocked-ip", "list", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestWafBlockedIPListCmd_NoAuth(t *testing.T) {
	cmd, _, _ := buildWafCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"waf", "blocked-ip", "list", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestWafBlockedIPListCmd_MissingArg(t *testing.T) {
	cmd, _, _ := buildWafCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"waf", "blocked-ip", "list"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

// --- Blocked IP Add Tests ---

func TestWafBlockedIPAddCmd_TableOutput(t *testing.T) {
	ts := newWafBlockedIPTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "blocked-ip", "add", "site-001", "192.168.1.100"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "IP 192.168.1.100 added to blocklist.")
}

func TestWafBlockedIPAddCmd_JSONOutput(t *testing.T) {
	ts := newWafBlockedIPTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"waf", "blocked-ip", "add", "site-001", "192.168.1.100"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "192.168.1.100", result["ip"])
}

func TestWafBlockedIPAddCmd_RequestBody(t *testing.T) {
	var receivedBody map[string]any
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(blockedIPAddResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "blocked-ip", "add", "site-001", "192.168.1.100"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/waf/blocked-ips", receivedPath)
	assert.Equal(t, "192.168.1.100", receivedBody["ip"])
}

func TestWafBlockedIPAddCmd_MissingArg(t *testing.T) {
	cmd, _, _ := buildWafCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"waf", "blocked-ip", "add", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 2 arg(s)")
}

// --- Blocked IP Remove Tests ---

func TestWafBlockedIPRemoveCmd_TableOutput(t *testing.T) {
	ts := newWafBlockedIPTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "blocked-ip", "remove", "site-001", "192.168.1.100"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "IP 192.168.1.100 removed from blocklist.")
}

func TestWafBlockedIPRemoveCmd_JSONOutput(t *testing.T) {
	ts := newWafBlockedIPTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"waf", "blocked-ip", "remove", "site-001", "192.168.1.100"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
}

func TestWafBlockedIPRemoveCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(blockedIPRemoveResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "blocked-ip", "remove", "site-001", "192.168.1.100"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "DELETE", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/waf/blocked-ips/192.168.1.100", receivedPath)
}

func TestWafBlockedIPRemoveCmd_MissingArg(t *testing.T) {
	cmd, _, _ := buildWafCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"waf", "blocked-ip", "remove", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 2 arg(s)")
}

// --- Help Tests ---

func TestWafBlockedIPCmd_Help(t *testing.T) {
	cmd := NewWafBlockedIPCmd()

	stdout := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "list")
	assert.Contains(t, out, "add")
	assert.Contains(t, out, "remove")
	assert.Contains(t, out, "blocked IP")
}
