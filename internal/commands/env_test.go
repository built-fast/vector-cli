package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/appctx"
	"github.com/built-fast/vector-cli/internal/config"
	"github.com/built-fast/vector-cli/internal/output"
)

var envListResponse = map[string]any{
	"data": []map[string]any{
		{
			"id":              "env-001",
			"vector_site_id":  "site-001",
			"name":            "production",
			"is_production":   true,
			"status":          "active",
			"php_version":     "8.3",
			"tags":            []string{"live"},
			"platform_domain": "test--prod.vectorpages.com",
			"custom_domain":   "example.com",
			"dns_target":      "site-abc.b-cdn.net",
			"database_host":   "db.rds.amazonaws.com",
			"database_name":   "db_env001",
			"custom_domain_certificate": map[string]any{
				"status":                "issued",
				"dns_validation_records": nil,
			},
			"created_at": "2025-01-15T12:00:00+00:00",
			"updated_at": "2025-01-15T12:00:00+00:00",
		},
	},
	"meta": map[string]any{
		"current_page": 1,
		"last_page":    1,
		"total":        1,
	},
	"message":     "Environments retrieved successfully",
	"http_status": 200,
}

var envShowResponse = map[string]any{
	"data": map[string]any{
		"id":              "env-001",
		"vector_site_id":  "site-001",
		"name":            "production",
		"is_production":   true,
		"status":          "active",
		"php_version":     "8.3",
		"tags":            []string{"live"},
		"platform_domain": "test--prod.vectorpages.com",
		"custom_domain":   "example.com",
		"dns_target":      "site-abc.b-cdn.net",
		"database_host":   "db.rds.amazonaws.com",
		"database_name":   "db_env001",
		"custom_domain_certificate": map[string]any{
			"status":                "issued",
			"dns_validation_records": nil,
		},
		"created_at": "2025-01-15T12:00:00+00:00",
		"updated_at": "2025-01-15T12:00:00+00:00",
	},
	"message":     "Environment retrieved successfully",
	"http_status": 200,
}

var envCreateResponse = map[string]any{
	"data": map[string]any{
		"id":              "env-002",
		"vector_site_id":  "site-001",
		"name":            "staging",
		"is_production":   false,
		"status":          "pending",
		"php_version":     "8.3",
		"tags":            []string{},
		"platform_domain": "test--staging.vectorpages.com",
		"custom_domain":   "",
	},
	"message":     "Environment creation initiated",
	"http_status": 201,
}

var envUpdateResponse = map[string]any{
	"data": map[string]any{
		"id":            "env-001",
		"name":          "production",
		"status":        "active",
		"tags":          []string{"updated"},
		"custom_domain": "new.example.com",
		"dns_target":    "site-abc.b-cdn.net",
	},
	"message":     "Environment updated successfully",
	"http_status": 200,
}

var envUpdateDomainChangeResponse = map[string]any{
	"data": map[string]any{
		"id":            "env-001",
		"name":          "production",
		"status":        "active",
		"tags":          []string{"live"},
		"custom_domain": "new.example.com",
		"dns_target":    "site-abc.b-cdn.net",
		"pending_domain_change": map[string]any{
			"id":         "dc-001",
			"status":     "pending",
			"old_domain": "old.example.com",
			"new_domain": "new.example.com",
		},
	},
	"message":     "Environment update initiated, domain change in progress",
	"http_status": 202,
}

var envDeleteResponse = map[string]any{
	"data": map[string]any{
		"id":     "env-001",
		"status": "terminating",
	},
	"message":     "Environment deletion initiated",
	"http_status": 202,
}

func newEnvTestServer(validToken string) *httptest.Server {
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
		case method == "GET" && path == "/api/v1/vector/environments":
			_ = json.NewEncoder(w).Encode(envListResponse)

		case method == "GET" && path == "/api/v1/vector/environments/env-001":
			_ = json.NewEncoder(w).Encode(envShowResponse)

		case method == "GET" && path == "/api/v1/vector/environments/nonexistent":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data":        map[string]any{},
				"message":     "Environment not found",
				"http_status": 404,
			})

		case method == "POST" && path == "/api/v1/vector/sites/site-001/environments":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(envCreateResponse)

		case method == "PUT" && path == "/api/v1/vector/environments/env-001":
			// Check if domain change
			body, _ := io.ReadAll(r.Body)
			var reqBody map[string]any
			_ = json.Unmarshal(body, &reqBody)
			if _, hasDomain := reqBody["custom_domain"]; hasDomain {
				w.WriteHeader(http.StatusAccepted)
				_ = json.NewEncoder(w).Encode(envUpdateDomainChangeResponse)
			} else {
				_ = json.NewEncoder(w).Encode(envUpdateResponse)
			}

		case method == "DELETE" && path == "/api/v1/vector/environments/env-001":
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(envDeleteResponse)

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Not Found",
				"http_status": 404,
			})
		}
	}))
}

func buildEnvCmd(baseURL, token string, format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
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

	envCmd := NewEnvCmd()
	root.AddCommand(envCmd)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

func buildEnvCmdNoAuth(format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
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

	envCmd := NewEnvCmd()
	root.AddCommand(envCmd)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

// --- Env List Tests ---

func TestEnvListCmd_TableOutput(t *testing.T) {
	ts := newEnvTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "list", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "env-001")
	assert.Contains(t, out, "production")
	assert.Contains(t, out, "Yes")
	assert.Contains(t, out, "active")
	assert.Contains(t, out, "8.3")
	assert.Contains(t, out, "test--prod.vectorpages.com")
	assert.Contains(t, out, "example.com")
}

func TestEnvListCmd_JSONOutput(t *testing.T) {
	ts := newEnvTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"env", "list", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Len(t, result, 1)
	assert.Equal(t, "env-001", result[0]["id"])
}

func TestEnvListCmd_SiteQueryParam(t *testing.T) {
	var receivedQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(envListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "list", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, receivedQuery, "site=site-001")
}

func TestEnvListCmd_AuthError(t *testing.T) {
	ts := newEnvTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildEnvCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"env", "list", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestEnvListCmd_NoAuth(t *testing.T) {
	cmd, _, _ := buildEnvCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"env", "list", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestEnvListCmd_MissingArg(t *testing.T) {
	ts := newEnvTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "list"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

// --- Env Show Tests ---

func TestEnvShowCmd_TableOutput(t *testing.T) {
	ts := newEnvTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "show", "env-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "env-001")
	assert.Contains(t, out, "site-001")
	assert.Contains(t, out, "production")
	assert.Contains(t, out, "Yes")
	assert.Contains(t, out, "8.3")
	assert.Contains(t, out, "example.com")
	assert.Contains(t, out, "site-abc.b-cdn.net")
	assert.Contains(t, out, "issued")
}

func TestEnvShowCmd_JSONOutput(t *testing.T) {
	ts := newEnvTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"env", "show", "env-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "env-001", result["id"])
	assert.Equal(t, "production", result["name"])
}

func TestEnvShowCmd_NotFound(t *testing.T) {
	ts := newEnvTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "show", "nonexistent"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 4, apiErr.ExitCode)
}

// --- Env Create Tests ---

func TestEnvCreateCmd_TableOutput(t *testing.T) {
	ts := newEnvTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "create", "site-001", "--name", "staging", "--php-version", "8.3"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "env-002")
	assert.Contains(t, out, "staging")
	assert.Contains(t, out, "pending")
}

func TestEnvCreateCmd_JSONOutput(t *testing.T) {
	ts := newEnvTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"env", "create", "site-001", "--name", "staging", "--php-version", "8.3"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "env-002", result["id"])
}

func TestEnvCreateCmd_PostsToSitePath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(envCreateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "create", "site-001", "--name", "staging", "--php-version", "8.3"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/environments", receivedPath)
}

func TestEnvCreateCmd_RequestBody(t *testing.T) {
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(envCreateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "create", "site-001",
		"--name", "staging",
		"--php-version", "8.3",
		"--production",
		"--tags", "test,staging",
		"--custom-domain", "staging.example.com",
	})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "staging", receivedBody["name"])
	assert.Equal(t, "8.3", receivedBody["php_version"])
	assert.Equal(t, true, receivedBody["is_production"])
	assert.Equal(t, "staging.example.com", receivedBody["custom_domain"])
	tags, ok := receivedBody["tags"].([]any)
	require.True(t, ok)
	assert.Equal(t, "test", tags[0])
	assert.Equal(t, "staging", tags[1])
}

func TestEnvCreateCmd_MissingName(t *testing.T) {
	ts := newEnvTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "create", "site-001", "--php-version", "8.3"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 3, apiErr.ExitCode)
}

func TestEnvCreateCmd_MissingPHPVersion(t *testing.T) {
	ts := newEnvTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "create", "site-001", "--name", "staging"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 3, apiErr.ExitCode)
}

// --- Env Update Tests ---

func TestEnvUpdateCmd_TableOutput(t *testing.T) {
	ts := newEnvTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "update", "env-001", "--tags", "updated"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "env-001")
	assert.Contains(t, out, "updated")
}

func TestEnvUpdateCmd_JSONOutput(t *testing.T) {
	ts := newEnvTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"env", "update", "env-001", "--tags", "updated"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "env-001", result["id"])
}

func TestEnvUpdateCmd_DomainChange202(t *testing.T) {
	ts := newEnvTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "update", "env-001", "--custom-domain", "new.example.com"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "Domain change initiated")
	assert.Contains(t, out, "old.example.com")
	assert.Contains(t, out, "new.example.com")
}

func TestEnvUpdateCmd_ClearCustomDomain(t *testing.T) {
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(envUpdateDomainChangeResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "update", "env-001", "--clear-custom-domain"})

	err := cmd.Execute()
	require.NoError(t, err)

	// custom_domain should be null (Go nil)
	assert.Contains(t, receivedBody, "custom_domain")
	assert.Nil(t, receivedBody["custom_domain"])
}

func TestEnvUpdateCmd_CustomDomainAndClearError(t *testing.T) {
	ts := newEnvTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "update", "env-001", "--custom-domain", "foo.com", "--clear-custom-domain"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 3, apiErr.ExitCode)
	assert.Contains(t, apiErr.Message, "cannot be used together")
}

func TestEnvUpdateCmd_RequestBody(t *testing.T) {
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(envUpdateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "update", "env-001", "--tags", "tag1,tag2"})

	err := cmd.Execute()
	require.NoError(t, err)

	tags, ok := receivedBody["tags"].([]any)
	require.True(t, ok)
	assert.Equal(t, "tag1", tags[0])
	assert.Equal(t, "tag2", tags[1])
}

// --- Env Delete Tests ---

func TestEnvDeleteCmd_WithForce(t *testing.T) {
	ts := newEnvTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "delete", "env-001", "--force"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "deletion initiated")
}

func TestEnvDeleteCmd_JSONOutput(t *testing.T) {
	ts := newEnvTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"env", "delete", "env-001", "--force"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "env-001", result["id"])
	assert.Equal(t, "terminating", result["status"])
}

func TestEnvDeleteCmd_ConfirmAbort(t *testing.T) {
	ts := newEnvTestServer("valid-token")
	defer ts.Close()

	origReader := confirmReader
	confirmReader = strings.NewReader("n\n")
	t.Cleanup(func() { confirmReader = origReader })

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "delete", "env-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Aborted")
}

func TestEnvDeleteCmd_ConfirmYes(t *testing.T) {
	ts := newEnvTestServer("valid-token")
	defer ts.Close()

	origReader := confirmReader
	confirmReader = strings.NewReader("y\n")
	t.Cleanup(func() { confirmReader = origReader })

	cmd, stdout, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "delete", "env-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "deletion initiated")
}

func TestEnvDeleteCmd_HTTPMethod(t *testing.T) {
	var receivedMethod string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(envDeleteResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "delete", "env-001", "--force"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "DELETE", receivedMethod)
}

// --- Help Text Tests ---

func TestEnvCmd_Help(t *testing.T) {
	cmd := NewEnvCmd()
	cmd.SetContext(context.Background())

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
	assert.Contains(t, out, "secret")
	assert.Contains(t, out, "db")
}

func TestEnvSecretCmd_Help(t *testing.T) {
	cmd := NewEnvCmd()
	cmd.SetContext(context.Background())

	stdout := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetArgs([]string{"secret", "--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "list")
	assert.Contains(t, out, "show")
	assert.Contains(t, out, "create")
	assert.Contains(t, out, "update")
	assert.Contains(t, out, "delete")
}

func TestEnvDBCmd_Help(t *testing.T) {
	cmd := NewEnvCmd()
	cmd.SetContext(context.Background())

	stdout := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetArgs([]string{"db", "--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "promote")
	assert.Contains(t, out, "promote-status")
}

// --- Server Error Test ---

func TestEnvListCmd_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message":     "Internal server error",
			"http_status": 500,
		})
	}))
	defer ts.Close()

	cmd, _, _ := buildEnvCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"env", "list", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 5, apiErr.ExitCode)
}
