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

var allowedReferrerListResponse = map[string]any{
	"data": []map[string]any{
		{"hostname": "trusted.example.com"},
		{"hostname": "partner.example.org"},
	},
	"message":     "Allowed referrers retrieved successfully",
	"http_status": 200,
}

var allowedReferrerAddResponse = map[string]any{
	"data":        map[string]any{"hostname": "trusted.example.com"},
	"message":     "Hostname added to allowed referrers",
	"http_status": 201,
}

var allowedReferrerRemoveResponse = map[string]any{
	"data":        map[string]any{},
	"message":     "Hostname removed from allowed referrers",
	"http_status": 200,
}

func newWafAllowedReferrerTestServer(validToken string) *httptest.Server {
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
		case method == "GET" && path == "/api/v1/vector/sites/site-001/waf/allowed-referrers":
			_ = json.NewEncoder(w).Encode(allowedReferrerListResponse)

		case method == "POST" && path == "/api/v1/vector/sites/site-001/waf/allowed-referrers":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(allowedReferrerAddResponse)

		case method == "DELETE" && path == "/api/v1/vector/sites/site-001/waf/allowed-referrers/trusted.example.com":
			_ = json.NewEncoder(w).Encode(allowedReferrerRemoveResponse)

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Not Found",
				"http_status": 404,
			})
		}
	}))
}

// --- Allowed Referrer List Tests ---

func TestWafAllowedReferrerListCmd_TableOutput(t *testing.T) {
	ts := newWafAllowedReferrerTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "allowed-referrer", "list", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "trusted.example.com")
	assert.Contains(t, out, "partner.example.org")
	assert.Contains(t, out, "HOSTNAME")
}

func TestWafAllowedReferrerListCmd_JSONOutput(t *testing.T) {
	ts := newWafAllowedReferrerTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"waf", "allowed-referrer", "list", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Len(t, result, 2)
	assert.Equal(t, "trusted.example.com", result[0]["hostname"])
}

func TestWafAllowedReferrerListCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(allowedReferrerListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "allowed-referrer", "list", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "GET", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/waf/allowed-referrers", receivedPath)
}

func TestWafAllowedReferrerListCmd_AuthError(t *testing.T) {
	ts := newWafAllowedReferrerTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildWafCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"waf", "allowed-referrer", "list", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestWafAllowedReferrerListCmd_NoAuth(t *testing.T) {
	cmd, _, _ := buildWafCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"waf", "allowed-referrer", "list", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestWafAllowedReferrerListCmd_MissingArg(t *testing.T) {
	cmd, _, _ := buildWafCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"waf", "allowed-referrer", "list"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

// --- Allowed Referrer Add Tests ---

func TestWafAllowedReferrerAddCmd_TableOutput(t *testing.T) {
	ts := newWafAllowedReferrerTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "allowed-referrer", "add", "site-001", "trusted.example.com"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Hostname trusted.example.com added to allowed referrers.")
}

func TestWafAllowedReferrerAddCmd_JSONOutput(t *testing.T) {
	ts := newWafAllowedReferrerTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"waf", "allowed-referrer", "add", "site-001", "trusted.example.com"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "trusted.example.com", result["hostname"])
}

func TestWafAllowedReferrerAddCmd_RequestBody(t *testing.T) {
	var receivedBody map[string]any
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(allowedReferrerAddResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "allowed-referrer", "add", "site-001", "trusted.example.com"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/waf/allowed-referrers", receivedPath)
	assert.Equal(t, "trusted.example.com", receivedBody["hostname"])
}

func TestWafAllowedReferrerAddCmd_MissingArg(t *testing.T) {
	cmd, _, _ := buildWafCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"waf", "allowed-referrer", "add", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 2 arg(s)")
}

// --- Allowed Referrer Remove Tests ---

func TestWafAllowedReferrerRemoveCmd_TableOutput(t *testing.T) {
	ts := newWafAllowedReferrerTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "allowed-referrer", "remove", "site-001", "trusted.example.com"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Hostname trusted.example.com removed from allowed referrers.")
}

func TestWafAllowedReferrerRemoveCmd_JSONOutput(t *testing.T) {
	ts := newWafAllowedReferrerTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildWafCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"waf", "allowed-referrer", "remove", "site-001", "trusted.example.com"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
}

func TestWafAllowedReferrerRemoveCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(allowedReferrerRemoveResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildWafCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"waf", "allowed-referrer", "remove", "site-001", "trusted.example.com"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "DELETE", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/waf/allowed-referrers/trusted.example.com", receivedPath)
}

func TestWafAllowedReferrerRemoveCmd_MissingArg(t *testing.T) {
	cmd, _, _ := buildWafCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"waf", "allowed-referrer", "remove", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 2 arg(s)")
}

// --- Help Tests ---

func TestWafAllowedReferrerCmd_Help(t *testing.T) {
	cmd := NewWafAllowedReferrerCmd()

	stdout := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "list")
	assert.Contains(t, out, "add")
	assert.Contains(t, out, "remove")
	assert.Contains(t, out, "allowed referrer")
}
