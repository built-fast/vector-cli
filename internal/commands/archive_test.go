package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/appctx"
	"github.com/built-fast/vector-cli/internal/config"
	"github.com/built-fast/vector-cli/internal/output"
)

var archiveImportCreateResponse = map[string]any{
	"data": map[string]any{
		"id":                "imp-100",
		"vector_site_id":    "site-001",
		"status":            "pending",
		"filename":          "archive.tar.gz",
		"content_length":    float64(10485760),
		"upload_url":        "__UPLOAD_URL__",
		"upload_expires_at": "2025-01-15T13:00:00+00:00",
		"created_at":        "2025-01-15T12:00:00+00:00",
	},
	"message":     "Import session created successfully",
	"http_status": 201,
}

var archiveImportRunResponse = map[string]any{
	"data": map[string]any{
		"id":             "imp-100",
		"vector_site_id": "site-001",
		"status":         "importing",
		"filename":       "archive.tar.gz",
		"created_at":     "2025-01-15T12:00:00+00:00",
	},
	"message":     "Archive import started",
	"http_status": 202,
}

func newArchiveImportTestServer(validToken string) *httptest.Server {
	mux := http.NewServeMux()

	// Upload endpoint (presigned URL simulation — no auth required)
	mux.HandleFunc("/upload/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		// Consume the body
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})

	// API endpoints
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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
		case method == "POST" && path == "/api/v1/vector/sites/site-001/imports":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(archiveImportCreateResponse)

		case method == "POST" && path == "/api/v1/vector/sites/site-001/imports/imp-100/run":
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(archiveImportRunResponse)

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "Not Found",
				"http_status": 404,
			})
		}
	})

	return httptest.NewServer(mux)
}

func buildArchiveCmd(baseURL, token string, format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	stdout := new(bytes.Buffer)

	root := &cobra.Command{Use: "vector"}
	root.AddCommand(NewArchiveCmd())

	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		client := api.NewClient(baseURL, token, "")
		app := appctx.NewApp(&config.Config{}, &config.Credentials{}, client, format, "test")
		app.Output = output.NewWriter(stdout, format)
		cmd.SetContext(appctx.WithApp(cmd.Context(), app))
		return nil
	}

	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

func buildArchiveCmdNoAuth(format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	stdout := new(bytes.Buffer)

	root := &cobra.Command{Use: "vector"}
	root.AddCommand(NewArchiveCmd())

	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		client := api.NewClient("", "", "")
		app := appctx.NewApp(&config.Config{}, &config.Credentials{}, client, format, "")
		app.Output = output.NewWriter(stdout, format)
		cmd.SetContext(appctx.WithApp(cmd.Context(), app))
		return nil
	}

	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

func createTempArchiveFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "archive.tar.gz")
	err := os.WriteFile(path, []byte("fake archive content for testing"), 0644)
	require.NoError(t, err)
	return path
}

// --- Archive Import Tests ---

func TestArchiveImportCmd_TableOutput(t *testing.T) {
	ts := newArchiveImportTestServer("valid-token")
	defer ts.Close()

	// Patch the upload URL in the create response to point to our test server
	archiveImportCreateResponse["data"].(map[string]any)["upload_url"] = ts.URL + "/upload/imp-100"
	defer func() {
		archiveImportCreateResponse["data"].(map[string]any)["upload_url"] = "__UPLOAD_URL__"
	}()

	tmpFile := createTempArchiveFile(t)

	cmd, stdout, stderr := buildArchiveCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"archive", "import", "site-001", tmpFile})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "imp-100")
	assert.Contains(t, out, "importing")

	errOut := stderr.String()
	assert.Contains(t, errOut, "Creating import session...")
	assert.Contains(t, errOut, "Uploading archive.tar.gz")
	assert.Contains(t, errOut, "Upload complete.")
	assert.Contains(t, errOut, "Starting import...")
	assert.Contains(t, errOut, "Import started.")
}

func TestArchiveImportCmd_JSONOutput(t *testing.T) {
	ts := newArchiveImportTestServer("valid-token")
	defer ts.Close()

	archiveImportCreateResponse["data"].(map[string]any)["upload_url"] = ts.URL + "/upload/imp-100"
	defer func() {
		archiveImportCreateResponse["data"].(map[string]any)["upload_url"] = "__UPLOAD_URL__"
	}()

	tmpFile := createTempArchiveFile(t)

	cmd, stdout, _ := buildArchiveCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"archive", "import", "site-001", tmpFile})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "imp-100", result["id"])
	assert.Equal(t, "importing", result["status"])
}

func TestArchiveImportCmd_HTTPPaths(t *testing.T) {
	var createMethod, createPath string
	var createBody map[string]any
	var runMethod, runPath string
	var uploadMethod, uploadPath string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		method := r.Method

		switch {
		case method == "PUT" && path == "/upload/imp-100":
			uploadMethod = method
			uploadPath = path
			_, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)

		case method == "POST" && path == "/api/v1/vector/sites/site-001/imports":
			createMethod = method
			createPath = path
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &createBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			// Return create response with this server's upload URL
			resp := map[string]any{
				"data": map[string]any{
					"id":         "imp-100",
					"status":     "pending",
					"upload_url": "http://" + r.Host + "/upload/imp-100",
				},
			}
			_ = json.NewEncoder(w).Encode(resp)

		case method == "POST" && path == "/api/v1/vector/sites/site-001/imports/imp-100/run":
			runMethod = method
			runPath = path
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(archiveImportRunResponse)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	tmpFile := createTempArchiveFile(t)

	cmd, _, _ := buildArchiveCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"archive", "import", "site-001", tmpFile})

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "POST", createMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/imports", createPath)
	assert.Equal(t, "archive.tar.gz", createBody["filename"])
	assert.NotNil(t, createBody["content_length"])

	assert.Equal(t, "PUT", uploadMethod)
	assert.Equal(t, "/upload/imp-100", uploadPath)

	assert.Equal(t, "POST", runMethod)
	assert.Equal(t, "/api/v1/vector/sites/site-001/imports/imp-100/run", runPath)
}

func TestArchiveImportCmd_WithOptions(t *testing.T) {
	var createBody map[string]any

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		method := r.Method

		switch {
		case method == "PUT":
			_, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)

		case method == "POST" && path == "/api/v1/vector/sites/site-001/imports":
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &createBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			resp := map[string]any{
				"data": map[string]any{
					"id":         "imp-100",
					"status":     "pending",
					"upload_url": "http://" + r.Host + "/upload/imp-100",
				},
			}
			_ = json.NewEncoder(w).Encode(resp)

		case method == "POST":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(archiveImportRunResponse)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	tmpFile := createTempArchiveFile(t)

	cmd, _, _ := buildArchiveCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{
		"archive", "import", "site-001", tmpFile,
		"--drop-tables",
		"--disable-foreign-keys",
		"--search-replace-from", "example.org",
		"--search-replace-to", "example.com",
	})

	err := cmd.Execute()
	require.NoError(t, err)

	options, ok := createBody["options"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, options["drop_tables"])
	assert.Equal(t, true, options["disable_foreign_keys"])

	sr, ok := options["search_replace"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "example.org", sr["from"])
	assert.Equal(t, "example.com", sr["to"])
}

func TestArchiveImportCmd_FileNotFound(t *testing.T) {
	ts := newArchiveImportTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildArchiveCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"archive", "import", "site-001", "/nonexistent/file.tar.gz"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot open file")
}

func TestArchiveImportCmd_MissingArgs(t *testing.T) {
	ts := newArchiveImportTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildArchiveCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"archive", "import", "site-001"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 2 arg(s)")
}

func TestArchiveImportCmd_MissingUploadURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":     "imp-100",
				"status": "pending",
			},
		})
	}))
	defer ts.Close()

	tmpFile := createTempArchiveFile(t)

	cmd, _, _ := buildArchiveCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"archive", "import", "site-001", tmpFile})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "import session response missing upload URL or import ID")
}

func TestArchiveImportCmd_AuthError(t *testing.T) {
	ts := newArchiveImportTestServer("valid-token")
	defer ts.Close()

	tmpFile := createTempArchiveFile(t)

	cmd, _, _ := buildArchiveCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"archive", "import", "site-001", tmpFile})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

func TestArchiveImportCmd_NoAuth(t *testing.T) {
	tmpFile := createTempArchiveFile(t)

	cmd, _, _ := buildArchiveCmdNoAuth(output.Table)
	cmd.SetArgs([]string{"archive", "import", "site-001", tmpFile})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

// --- Help Tests ---

func TestArchiveCmd_Help(t *testing.T) {
	cmd := NewArchiveCmd()

	stdout := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "import")
	assert.Contains(t, out, "Manage site archives")
}

func TestArchiveImportCmd_Help(t *testing.T) {
	cmd := NewArchiveCmd()

	stdout := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetArgs([]string{"import", "--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "Import a site archive from a local file")
	assert.Contains(t, out, "--drop-tables")
	assert.Contains(t, out, "--disable-foreign-keys")
	assert.Contains(t, out, "--search-replace-from")
	assert.Contains(t, out, "--search-replace-to")
}
