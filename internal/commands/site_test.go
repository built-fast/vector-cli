package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/appctx"
	"github.com/built-fast/vector-cli/internal/config"
	"github.com/built-fast/vector-cli/internal/output"
)

// siteListResponse is the standard response for GET /api/v1/vector/sites.
var siteListResponse = map[string]any{
	"data": []map[string]any{
		{
			"id":               "site-001",
			"your_customer_id": "cust_123",
			"status":           "active",
			"dev_domain":       "dev.test.vectorpages.com",
			"tags":             []string{"wordpress", "production"},
			"dev_db_host":      "db.test.rds.amazonaws.com",
			"dev_db_name":      "db_site001",
			"environments": []map[string]any{
				{
					"id":                              "env-001",
					"name":                            "production",
					"is_production":                   true,
					"status":                          "active",
					"php_version":                     "8.3",
					"platform_domain":                 "test--prod.vectorpages.com",
					"custom_domain":                   "example.com",
					"custom_domain_certificate_status": "issued",
					"created_at":                      "2025-01-15T12:00:00+00:00",
					"updated_at":                      "2025-01-15T12:00:00+00:00",
				},
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
	"message":     "Sites retrieved successfully",
	"http_status": 200,
}

// siteShowResponse is the standard response for GET /api/v1/vector/sites/{site}.
var siteShowResponse = map[string]any{
	"data": map[string]any{
		"id":               "site-001",
		"your_customer_id": "cust_123",
		"status":           "active",
		"tags":             []string{"wordpress"},
		"dev_domain":       "dev.test.vectorpages.com",
		"dev_db_host":      "db.test.rds.amazonaws.com",
		"dev_db_name":      "db_site001",
		"environments": []map[string]any{
			{
				"id":              "env-001",
				"name":            "production",
				"is_production":   true,
				"status":          "active",
				"php_version":     "8.3",
				"platform_domain": "test--prod.vectorpages.com",
				"custom_domain":   "example.com",
			},
		},
		"created_at": "2025-01-15T12:00:00+00:00",
		"updated_at": "2025-01-15T12:00:00+00:00",
	},
	"message":     "Site retrieved successfully",
	"http_status": 200,
}

var siteCreateResponse = map[string]any{
	"data": map[string]any{
		"id":               "site-002",
		"your_customer_id": "cust_456",
		"status":           "pending",
		"dev_domain":       "dev.new.vectorpages.com",
		"dev_db_host":      "db.new.rds.amazonaws.com",
		"dev_db_name":      "db_site002",
		"dev_sftp": map[string]any{
			"hostname": "ssh.vectorpages.com",
			"port":     22,
			"username": "new-site",
			"password": "sftp-pass-123",
		},
		"dev_db_username": "db_site002",
		"dev_db_password": "db-pass-456",
		"wp_admin": map[string]any{
			"user":       "admin",
			"email":      "admin@example.com",
			"password":   "wp-pass-789",
			"site_title": "My Blog",
		},
		"environments": []map[string]any{},
		"created_at":   "2025-01-15T12:00:00+00:00",
		"updated_at":   "2025-01-15T12:00:00+00:00",
	},
	"message":     "Vector site creation initiated",
	"http_status": 201,
}

var siteDeleteResponse = map[string]any{
	"data": map[string]any{
		"id":               "site-001",
		"your_customer_id": "cust_123",
		"status":           "terminating",
		"environments":     []any{},
	},
	"message":     "Vector site deletion initiated",
	"http_status": 202,
}

var siteSuspendResponse = map[string]any{
	"data": map[string]any{
		"id":     "site-001",
		"status": "suspended",
	},
	"message":     "Vector site suspension initiated",
	"http_status": 200,
}

var siteUnsuspendResponse = map[string]any{
	"data": map[string]any{
		"id":     "site-001",
		"status": "active",
	},
	"message":     "Vector site unsuspension initiated",
	"http_status": 200,
}

var siteCloneResponse = map[string]any{
	"data": map[string]any{
		"id":               "site-003",
		"your_customer_id": "cust_123",
		"status":           "pending",
		"dev_domain":       "dev.clone.vectorpages.com",
		"dev_db_username":  "db_site003",
		"dev_db_password":  "clone-pass-123",
		"environments":     []any{},
	},
	"message":     "Vector site clone initiated",
	"http_status": 201,
}

var siteResetSFTPResponse = map[string]any{
	"data": map[string]any{
		"id": "site-001",
		"dev_sftp": map[string]any{
			"hostname": "ssh.vectorpages.com",
			"port":     22,
			"username": "test-site",
			"password": "new-sftp-pass",
		},
	},
	"message":     "SFTP password reset successfully.",
	"http_status": 200,
}

var siteResetDBResponse = map[string]any{
	"data": map[string]any{
		"id":              "site-001",
		"dev_db_username": "db_site001",
		"dev_db_password": "new-db-pass",
	},
	"message":     "Database password reset successfully.",
	"http_status": 200,
}

var sitePurgeCacheResponse = map[string]any{
	"data":        map[string]any{},
	"message":     "Cache purged successfully",
	"http_status": 200,
}

var siteLogsResponse = map[string]any{
	"data": map[string]any{
		"logs": map[string]any{
			"tables": []map[string]any{
				{
					"name": "0",
					"columns": []map[string]any{
						{"name": "_time", "type": "datetime"},
						{"name": "message", "type": "string"},
						{"name": "level", "type": "string"},
					},
					"rows": [][]string{
						{"2025-01-15T12:00:00+00:00", "Request completed", "info"},
					},
				},
			},
		},
		"cursor":   "abc123",
		"has_more": true,
	},
	"message":     "Logs retrieved successfully",
	"http_status": 200,
}

var siteWPReconfigResponse = map[string]any{
	"data": map[string]any{
		"id":     "site-001",
		"status": "active",
	},
	"message":     "WordPress configuration regenerated successfully",
	"http_status": 200,
}

var siteUpdateResponse = map[string]any{
	"data": map[string]any{
		"id":               "site-001",
		"your_customer_id": "cust_999",
		"status":           "active",
		"tags":             []string{"updated"},
		"dev_domain":       "dev.test.vectorpages.com",
	},
	"message":     "Vector site updated successfully",
	"http_status": 200,
}

// newSiteTestServer creates an httptest server that handles site endpoints.
func newSiteTestServer(validToken string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check auth
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
		case method == "GET" && path == "/api/v1/vector/sites":
			_ = json.NewEncoder(w).Encode(siteListResponse)

		case method == "GET" && path == "/api/v1/vector/sites/site-001":
			_ = json.NewEncoder(w).Encode(siteShowResponse)

		case method == "GET" && path == "/api/v1/vector/sites/nonexistent":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data":        map[string]any{},
				"message":     "Site not found",
				"http_status": 404,
			})

		case method == "POST" && path == "/api/v1/vector/sites":
			// Validate request body
			body, _ := io.ReadAll(r.Body)
			var reqBody map[string]any
			_ = json.Unmarshal(body, &reqBody)
			if reqBody["your_customer_id"] == nil || reqBody["your_customer_id"] == "" {
				w.WriteHeader(http.StatusUnprocessableEntity)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"errors": map[string][]string{
						"your_customer_id": {"The partner customer id field is required."},
					},
					"message":     "Validation failed",
					"http_status": 422,
				})
				return
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(siteCreateResponse)

		case method == "PUT" && path == "/api/v1/vector/sites/site-001":
			_ = json.NewEncoder(w).Encode(siteUpdateResponse)

		case method == "DELETE" && path == "/api/v1/vector/sites/site-001":
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(siteDeleteResponse)

		case method == "PUT" && path == "/api/v1/vector/sites/site-001/suspend":
			_ = json.NewEncoder(w).Encode(siteSuspendResponse)

		case method == "PUT" && path == "/api/v1/vector/sites/site-001/unsuspend":
			_ = json.NewEncoder(w).Encode(siteUnsuspendResponse)

		case method == "POST" && path == "/api/v1/vector/sites/site-001/clone":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(siteCloneResponse)

		case method == "POST" && path == "/api/v1/vector/sites/site-001/sftp/reset-password":
			_ = json.NewEncoder(w).Encode(siteResetSFTPResponse)

		case method == "POST" && path == "/api/v1/vector/sites/site-001/db/reset-password":
			_ = json.NewEncoder(w).Encode(siteResetDBResponse)

		case method == "POST" && path == "/api/v1/vector/sites/site-001/purge-cache":
			_ = json.NewEncoder(w).Encode(sitePurgeCacheResponse)

		case method == "GET" && path == "/api/v1/vector/sites/site-001/logs":
			_ = json.NewEncoder(w).Encode(siteLogsResponse)

		case method == "POST" && path == "/api/v1/vector/sites/site-001/wp/reconfig":
			_ = json.NewEncoder(w).Encode(siteWPReconfigResponse)

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Not Found",
				"http_status": 404,
			})
		}
	}))
}

// buildSiteCmd creates a root + site command wired with an App context.
func buildSiteCmd(baseURL, token string, format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
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

	siteCmd := NewSiteCmd()
	root.AddCommand(siteCmd)

	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

// buildSiteCmdNoAuth creates a root + site command with no auth token.
func buildSiteCmdNoAuth(format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
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

	siteCmd := NewSiteCmd()
	root.AddCommand(siteCmd)

	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

// --- Site List Tests ---

func TestSiteListCmd_TableOutput(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "list"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "site-001")
	assert.Contains(t, out, "cust_123")
	assert.Contains(t, out, "active")
	assert.Contains(t, out, "dev.test.vectorpages.com")
	assert.Contains(t, out, "wordpress, production")
}

func TestSiteListCmd_JSONOutput(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"site", "list"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Len(t, result, 1)
	assert.Equal(t, "site-001", result[0]["id"])
}

func TestSiteListCmd_Pagination(t *testing.T) {
	var receivedPage, receivedPerPage string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPage = r.URL.Query().Get("page")
		receivedPerPage = r.URL.Query().Get("per_page")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(siteListResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "list", "--page", "2", "--per-page", "10"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "2", receivedPage)
	assert.Equal(t, "10", receivedPerPage)
}

func TestSiteListCmd_AuthError(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildSiteCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"site", "list"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestSiteListCmd_NoAuth(t *testing.T) {
	cmd, _, _ := buildSiteCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"site", "list"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

// --- Site Show Tests ---

func TestSiteShowCmd_TableOutput(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "show", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "site-001")
	assert.Contains(t, out, "cust_123")
	assert.Contains(t, out, "active")
	assert.Contains(t, out, "wordpress")
	// Should contain environments table
	assert.Contains(t, out, "Environments:")
	assert.Contains(t, out, "env-001")
	assert.Contains(t, out, "production")
	assert.Contains(t, out, "example.com")
}

func TestSiteShowCmd_JSONOutput(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"site", "show", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "site-001", result["id"])
	assert.Equal(t, "cust_123", result["your_customer_id"])
}

func TestSiteShowCmd_NotFound(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "show", "nonexistent"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 4, apiErr.ExitCode)
}

func TestSiteShowCmd_MissingArg(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "show"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

// --- Site Create Tests ---

func TestSiteCreateCmd_TableOutput(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "create", "--customer-id", "cust_456"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "site-002")
	assert.Contains(t, out, "pending")
	assert.Contains(t, out, "sftp-pass-123")
	assert.Contains(t, out, "db-pass-456")
	assert.Contains(t, out, "wp-pass-789")
}

func TestSiteCreateCmd_JSONOutput(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"site", "create", "--customer-id", "cust_456"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "site-002", result["id"])
	assert.Equal(t, "pending", result["status"])
}

func TestSiteCreateCmd_MissingCustomerID(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "create"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 3, apiErr.ExitCode)
}

func TestSiteCreateCmd_RequestBody(t *testing.T) {
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(siteCreateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "create",
		"--customer-id", "cust_789",
		"--php-version", "8.3",
		"--tags", "wordpress,staging",
		"--wp-admin-email", "admin@test.com",
	})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "cust_789", receivedBody["your_customer_id"])
	assert.Equal(t, "8.3", receivedBody["dev_php_version"])
	assert.Equal(t, "admin@test.com", receivedBody["wp_admin_email"])
	tags, ok := receivedBody["tags"].([]any)
	require.True(t, ok)
	assert.Equal(t, "wordpress", tags[0])
	assert.Equal(t, "staging", tags[1])
}

// --- Site Update Tests ---

func TestSiteUpdateCmd_TableOutput(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "update", "site-001", "--customer-id", "cust_999"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "site-001")
	assert.Contains(t, out, "cust_999")
}

func TestSiteUpdateCmd_JSONOutput(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"site", "update", "site-001", "--customer-id", "cust_999"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "site-001", result["id"])
}

func TestSiteUpdateCmd_RequestBody(t *testing.T) {
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(siteUpdateResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "update", "site-001", "--customer-id", "new_id", "--tags", "tag1,tag2"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "new_id", receivedBody["your_customer_id"])
	tags, ok := receivedBody["tags"].([]any)
	require.True(t, ok)
	assert.Equal(t, "tag1", tags[0])
	assert.Equal(t, "tag2", tags[1])
}

// --- Site Delete Tests ---

func TestSiteDeleteCmd_WithForce(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "delete", "site-001", "--force"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "deletion initiated")
}

func TestSiteDeleteCmd_JSONOutput(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"site", "delete", "site-001", "--force"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "site-001", result["id"])
	assert.Equal(t, "terminating", result["status"])
}

func TestSiteDeleteCmd_ConfirmAbort(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	// Override confirmReader to return "n"
	origReader := confirmReader
	confirmReader = strings.NewReader("n\n")
	t.Cleanup(func() { confirmReader = origReader })

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "delete", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Aborted")
}

func TestSiteDeleteCmd_ConfirmYes(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	origReader := confirmReader
	confirmReader = strings.NewReader("y\n")
	t.Cleanup(func() { confirmReader = origReader })

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "delete", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "deletion initiated")
}

func TestSiteDeleteCmd_HTTPMethod(t *testing.T) {
	var receivedMethod string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(siteDeleteResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "delete", "site-001", "--force"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "DELETE", receivedMethod)
}

// --- Site Clone Tests ---

func TestSiteCloneCmd_TableOutput(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "clone", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "site-003")
	assert.Contains(t, out, "pending")
	assert.Contains(t, out, "clone-pass-123")
}

func TestSiteCloneCmd_JSONOutput(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"site", "clone", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "site-003", result["id"])
}

func TestSiteCloneCmd_RequestBody(t *testing.T) {
	var receivedBody map[string]any
	var receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(siteCloneResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "clone", "site-001", "--customer-id", "new_cust", "--php-version", "8.4"})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "/api/v1/vector/sites/site-001/clone", receivedPath)
	assert.Equal(t, "new_cust", receivedBody["your_customer_id"])
	assert.Equal(t, "8.4", receivedBody["dev_php_version"])
}

// --- Site Suspend/Unsuspend Tests ---

func TestSiteSuspendCmd_TableOutput(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "suspend", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "suspend initiated")
}

func TestSiteSuspendCmd_JSONOutput(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"site", "suspend", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "suspended", result["status"])
}

func TestSiteSuspendCmd_HTTPMethod(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(siteSuspendResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "suspend", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "PUT", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/suspend", receivedPath)
}

func TestSiteUnsuspendCmd_TableOutput(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "unsuspend", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "unsuspend initiated")
}

func TestSiteUnsuspendCmd_JSONOutput(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"site", "unsuspend", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "active", result["status"])
}

// --- Site Reset SFTP Password Tests ---

func TestSiteResetSFTPPasswordCmd_TableOutput(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "reset-sftp-password", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "ssh.vectorpages.com")
	assert.Contains(t, out, "test-site")
	assert.Contains(t, out, "new-sftp-pass")
}

func TestSiteResetSFTPPasswordCmd_JSONOutput(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"site", "reset-sftp-password", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	sftp := result["dev_sftp"].(map[string]any)
	assert.Equal(t, "new-sftp-pass", sftp["password"])
}

func TestSiteResetSFTPPasswordCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(siteResetSFTPResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "reset-sftp-password", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/sftp/reset-password", receivedPath)
}

// --- Site Reset DB Password Tests ---

func TestSiteResetDBPasswordCmd_TableOutput(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "reset-db-password", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "db_site001")
	assert.Contains(t, out, "new-db-pass")
}

func TestSiteResetDBPasswordCmd_JSONOutput(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"site", "reset-db-password", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "new-db-pass", result["dev_db_password"])
}

// --- Site Purge Cache Tests ---

func TestSitePurgeCacheCmd_FullPurge(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "purge-cache", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Cache purged successfully")
}

func TestSitePurgeCacheCmd_JSONOutput(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"site", "purge-cache", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	// data is empty object for full purge
	assert.NotNil(t, result)
}

func TestSitePurgeCacheCmd_WithTag(t *testing.T) {
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sitePurgeCacheResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "purge-cache", "site-001", "--cache-tag", "images"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "images", receivedBody["cache_tag"])
}

func TestSitePurgeCacheCmd_WithURL(t *testing.T) {
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sitePurgeCacheResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "purge-cache", "site-001", "--url", "https://example.com/style.css"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/style.css", receivedBody["url"])
}

// --- Site Logs Tests ---

func TestSiteLogsCmd_TableOutput(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "logs", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "Request completed")
	assert.Contains(t, out, "info")
	assert.Contains(t, out, "--cursor abc123")
}

func TestSiteLogsCmd_JSONOutput(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"site", "logs", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "abc123", result["cursor"])
	assert.Equal(t, true, result["has_more"])
}

func TestSiteLogsCmd_QueryParams(t *testing.T) {
	var receivedQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(siteLogsResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "logs", "site-001",
		"--start-time", "now-24h",
		"--level", "error",
		"--limit", "500",
		"--environment", "production",
	})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, receivedQuery, "start_time=now-24h")
	assert.Contains(t, receivedQuery, "level=error")
	assert.Contains(t, receivedQuery, "limit=500")
	assert.Contains(t, receivedQuery, "environment=production")
}

// --- Site WP Reconfig Tests ---

func TestSiteWPReconfigCmd_TableOutput(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "wp-reconfig", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "WordPress configuration regenerated successfully")
}

func TestSiteWPReconfigCmd_JSONOutput(t *testing.T) {
	ts := newSiteTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"site", "wp-reconfig", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "site-001", result["id"])
}

func TestSiteWPReconfigCmd_HTTPPath(t *testing.T) {
	var receivedMethod, receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(siteWPReconfigResponse)
	}))
	defer ts.Close()

	cmd, _, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "wp-reconfig", "site-001"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/wp/reconfig", receivedPath)
}

// --- Help Text Tests ---

func TestSiteCmd_Help(t *testing.T) {
	cmd := NewSiteCmd()
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
	assert.Contains(t, out, "clone")
	assert.Contains(t, out, "suspend")
	assert.Contains(t, out, "unsuspend")
	assert.Contains(t, out, "ssh-key")
	assert.Contains(t, out, "purge-cache")
	assert.Contains(t, out, "logs")
}

func TestSiteCreateCmd_Help(t *testing.T) {
	cmd := NewSiteCmd()
	cmd.SetContext(context.Background())

	stdout := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetArgs([]string{"create", "--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "--customer-id")
	assert.Contains(t, out, "--php-version")
	assert.Contains(t, out, "--tags")
}

// --- Server Error Tests ---

func TestSiteListCmd_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message":     "Internal server error",
			"http_status": 500,
		})
	}))
	defer ts.Close()

	cmd, _, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "list"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 5, apiErr.ExitCode)
}

// --- Site Create --wait Tests ---

// siteActivePollResponse is the polled response for a site that has become active.
// It does NOT contain one-time credentials (those are only in the initial POST response).
var siteActivePollResponse = map[string]any{
	"data": map[string]any{
		"id":               "site-002",
		"your_customer_id": "cust_456",
		"status":           "active",
		"dev_domain":       "dev.new.vectorpages.com",
		"dev_db_host":      "db.new.rds.amazonaws.com",
		"dev_db_name":      "db_site002",
		"environments": []map[string]any{
			{
				"id":              "env-002",
				"name":            "production",
				"is_production":   true,
				"status":          "active",
				"php_version":     "8.3",
				"platform_domain": "new--prod.vectorpages.com",
			},
		},
		"created_at": "2025-01-15T12:00:00+00:00",
		"updated_at": "2025-01-15T12:05:00+00:00",
	},
	"message":     "Site retrieved successfully",
	"http_status": 200,
}

// newSiteWaitTestServer creates a test server that handles:
// - POST /sites -> returns siteCreateResponse (with credentials)
// - GET /sites/{id} -> returns successive poll responses (without credentials)
func newSiteWaitTestServer(validToken string, pollResponses []countingResponse) *httptest.Server {
	var pollCount atomic.Int64

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
		case method == "POST" && path == "/api/v1/vector/sites":
			body, _ := io.ReadAll(r.Body)
			var reqBody map[string]any
			_ = json.Unmarshal(body, &reqBody)
			if reqBody["your_customer_id"] == nil || reqBody["your_customer_id"] == "" {
				w.WriteHeader(http.StatusUnprocessableEntity)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"errors": map[string][]string{
						"your_customer_id": {"The partner customer id field is required."},
					},
					"message":     "Validation failed",
					"http_status": 422,
				})
				return
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(siteCreateResponse)

		case method == "GET" && path == "/api/v1/vector/sites/site-002":
			idx := int(pollCount.Add(1)) - 1
			if idx >= len(pollResponses) {
				idx = len(pollResponses) - 1
			}
			resp := pollResponses[idx]
			if resp.httpStatus != 0 {
				w.WriteHeader(resp.httpStatus)
			}
			_ = json.NewEncoder(w).Encode(resp.body)

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Not Found",
				"http_status": 404,
			})
		}
	}))
}

func makeSitePollResponse(id, status string) countingResponse {
	return countingResponse{
		httpStatus: http.StatusOK,
		body: map[string]any{
			"data": map[string]any{
				"id":               id,
				"your_customer_id": "cust_456",
				"status":           status,
				"dev_domain":       "dev.new.vectorpages.com",
			},
			"message":     "Site retrieved successfully",
			"http_status": 200,
		},
	}
}

func TestSiteCreateCmd_WaitSuccess(t *testing.T) {
	overrideWaitGlobals(t, false)

	ts := newSiteWaitTestServer("valid-token", []countingResponse{
		makeSitePollResponse("site-002", "pending"),
		makeSitePollResponse("site-002", "provisioning"),
		{
			httpStatus: http.StatusOK,
			body:       siteActivePollResponse,
		},
	})
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "create", "--customer-id", "cust_456", "--wait", "--poll-interval", "1s", "--timeout", "30s"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()

	// Credentials should be printed before polling (from the initial POST response)
	assert.Contains(t, out, "sftp-pass-123")
	assert.Contains(t, out, "db-pass-456")
	assert.Contains(t, out, "wp-pass-789")

	// Final state should be shown after polling completes
	assert.Contains(t, out, "Site site-002 active in")
	assert.Contains(t, out, "active")
}

func TestSiteCreateCmd_WaitFailure(t *testing.T) {
	overrideWaitGlobals(t, false)

	ts := newSiteWaitTestServer("valid-token", []countingResponse{
		makeSitePollResponse("site-002", "pending"),
		makeSitePollResponse("site-002", "failed"),
	})
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"site", "create", "--customer-id", "cust_456", "--wait", "--poll-interval", "1s", "--timeout", "30s"})

	err := cmd.Execute()
	require.Error(t, err)

	// Even on failure, credentials should have been printed
	out := stdout.String()
	assert.Contains(t, out, "sftp-pass-123")

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 1, apiErr.ExitCode)
	assert.Contains(t, apiErr.Message, "failed status")
}

func TestSiteCreateCmd_WaitJSON(t *testing.T) {
	overrideWaitGlobals(t, false)

	ts := newSiteWaitTestServer("valid-token", []countingResponse{
		makeSitePollResponse("site-002", "pending"),
		{
			httpStatus: http.StatusOK,
			body:       siteActivePollResponse,
		},
	})
	defer ts.Close()

	cmd, stdout, _ := buildSiteCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"site", "create", "--customer-id", "cust_456", "--wait", "--poll-interval", "1s", "--timeout", "30s"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))

	// Final status should be active
	assert.Equal(t, "site-002", result["id"])
	assert.Equal(t, "active", result["status"])

	// One-time credentials from the initial POST should be merged into the final JSON
	sftp, ok := result["dev_sftp"].(map[string]any)
	require.True(t, ok, "dev_sftp should be merged into final JSON")
	assert.Equal(t, "sftp-pass-123", sftp["password"])

	assert.Equal(t, "db_site002", result["dev_db_username"])
	assert.Equal(t, "db-pass-456", result["dev_db_password"])

	wp, ok := result["wp_admin"].(map[string]any)
	require.True(t, ok, "wp_admin should be merged into final JSON")
	assert.Equal(t, "wp-pass-789", wp["password"])
}
