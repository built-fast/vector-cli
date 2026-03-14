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

var blockedReferrerListResponse = map[string]any{
	"data": []map[string]any{
		{"hostname": "spam.example.com"},
		{"hostname": "bad-referrer.net"},
	},
	"message":     "Blocked referrers retrieved successfully",
	"http_status": 200,
}

var blockedReferrerAddResponse = map[string]any{
	"data":        map[string]any{"hostname": "spam.example.com"},
	"message":     "Hostname added to blocked referrers",
	"http_status": 201,
}

var blockedReferrerRemoveResponse = map[string]any{
	"data":        map[string]any{},
	"message":     "Hostname removed from blocked referrers",
	"http_status": 200,
}

func newWafBlockedReferrerTestServer(validToken string) *httptest.Server {
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
		case method == "GET" && path == "/api/v1/vector/sites/site-001/waf/blocked-referrers":
			_ = json.NewEncoder(w).Encode(blockedReferrerListResponse)

		case method == "POST" && path == "/api/v1/vector/sites/site-001/waf/blocked-referrers":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(blockedReferrerAddResponse)

		case method == "DELETE" && path == "/api/v1/vector/sites/site-001/waf/blocked-referrers/spam.example.com":
			_ = json.NewEncoder(w).Encode(blockedReferrerRemoveResponse)

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Not Found",
				"http_status": 404,
			})
		}
	}))
}

// --- Blocked Referrer List Tests ---

func TestWafBlockedReferrerListCmd_TableOutput(t *testing.T) {
	ts := newWafBlockedReferrerTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "blocked-referrer", "list", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "spam.example.com")
	assert.Contains(t, out, "bad-referrer.net")
	assert.Contains(t, out, "HOSTNAME")
}

func TestWafBlockedReferrerListCmd_JSONOutput(t *testing.T) {
	ts := newWafBlockedReferrerTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"waf", "blocked-referrer", "list", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Len(t, result, 2)
	assert.Equal(t, "spam.example.com", result[0]["hostname"])
}

func TestWafBlockedReferrerListCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(blockedReferrerListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "blocked-referrer", "list", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "GET", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/waf/blocked-referrers", receivedPath)
}

func TestWafBlockedReferrerListCmd_AuthError(t *testing.T) {
	ts := newWafBlockedReferrerTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildWafCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"waf", "blocked-referrer", "list", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestWafBlockedReferrerListCmd_NoAuth(t *testing.T) {
	cmd, _, _ := buildWafCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"waf", "blocked-referrer", "list", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestWafBlockedReferrerListCmd_MissingArg(t *testing.T) {
	cmd, _, _ := buildWafCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"waf", "blocked-referrer", "list"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

// --- Blocked Referrer Add Tests ---

func TestWafBlockedReferrerAddCmd_TableOutput(t *testing.T) {
	ts := newWafBlockedReferrerTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "blocked-referrer", "add", "site-001", "spam.example.com"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Hostname spam.example.com added to blocked referrers.")
}

func TestWafBlockedReferrerAddCmd_JSONOutput(t *testing.T) {
	ts := newWafBlockedReferrerTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"waf", "blocked-referrer", "add", "site-001", "spam.example.com"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "spam.example.com", result["hostname"])
}

func TestWafBlockedReferrerAddCmd_RequestBody(t *testing.T) {
	var receivedBody map[string]any
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(blockedReferrerAddResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "blocked-referrer", "add", "site-001", "spam.example.com"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/waf/blocked-referrers", receivedPath)
	assert.Equal(t, "spam.example.com", receivedBody["hostname"])
}

func TestWafBlockedReferrerAddCmd_MissingArg(t *testing.T) {
	cmd, _, _ := buildWafCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"waf", "blocked-referrer", "add", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 2 arg(s)")
}

// --- Blocked Referrer Remove Tests ---

func TestWafBlockedReferrerRemoveCmd_TableOutput(t *testing.T) {
	ts := newWafBlockedReferrerTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "blocked-referrer", "remove", "site-001", "spam.example.com"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Hostname spam.example.com removed from blocked referrers.")
}

func TestWafBlockedReferrerRemoveCmd_JSONOutput(t *testing.T) {
	ts := newWafBlockedReferrerTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"waf", "blocked-referrer", "remove", "site-001", "spam.example.com"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
}

func TestWafBlockedReferrerRemoveCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(blockedReferrerRemoveResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "blocked-referrer", "remove", "site-001", "spam.example.com"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "DELETE", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/waf/blocked-referrers/spam.example.com", receivedPath)
}

func TestWafBlockedReferrerRemoveCmd_MissingArg(t *testing.T) {
	cmd, _, _ := buildWafCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"waf", "blocked-referrer", "remove", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 2 arg(s)")
}

// --- Help Tests ---

func TestWafBlockedReferrerCmd_Help(t *testing.T) {
	cmd := NewWafBlockedReferrerCmd()

	stdout := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "list")
	assert.Contains(t, out, "add")
	assert.Contains(t, out, "remove")
	assert.Contains(t, out, "blocked referrer")
}
