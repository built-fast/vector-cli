package commands

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

// pingResponse is the standard response from GET /api/v1/ping.
var pingResponse = map[string]any{
	"data":        map[string]any{"response": "pong"},
	"message":     "API health check successful",
	"http_status": 200,
}

// newTestServer creates an httptest server that responds to /api/v1/ping.
// validToken is the token that triggers a 200; anything else gets 401.
func newTestServer(validToken string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/ping" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

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
		_ = json.NewEncoder(w).Encode(pingResponse)
	}))
}

// buildAuthLoginCmd creates a root + auth + login command wired with an App context.
func buildAuthLoginCmd(baseURL, token string, format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	root := &cobra.Command{
		Use: "vector",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			client := api.NewClient(baseURL, token, "test-agent")
			app := appctx.NewApp(
				config.DefaultConfig(),
				&config.Credentials{},
				client,
				format,
			)
			cmd.SetContext(appctx.WithApp(cmd.Context(), app))
			return nil
		},
	}

	authCmd := NewAuthCmd()
	root.AddCommand(authCmd)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

func TestAuthLoginCmd_ValidToken_TableOutput(t *testing.T) {
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())

	ts := newTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAuthLoginCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"auth", "login"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "Successfully authenticated.", strings.TrimSpace(stdout.String()))

	// Verify credentials were saved
	creds, err := config.LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "valid-token", creds.ApiKey)
}

func TestAuthLoginCmd_ValidToken_JSONOutput(t *testing.T) {
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())

	ts := newTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAuthLoginCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"auth", "login"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "pong", result["data"].(map[string]any)["response"])
	assert.Equal(t, "API health check successful", result["message"])
	assert.Equal(t, float64(200), result["http_status"])
}

func TestAuthLoginCmd_InvalidToken(t *testing.T) {
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())

	ts := newTestServer("valid-token")
	defer ts.Close()

	cmd, _, stderr := buildAuthLoginCmd(ts.URL, "bad-token", output.Table)
	cmd.SetArgs([]string{"auth", "login"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
	assert.Equal(t, "Invalid API token.", apiErr.Message)

	// stderr should show the error (via the root command's silence + execute.go pattern)
	_ = stderr // error is returned, not printed by login itself
}

func TestAuthLoginCmd_NetworkError(t *testing.T) {
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())

	// Use an unreachable URL to trigger a network error
	cmd, _, _ := buildAuthLoginCmd("http://127.0.0.1:1", "some-token", output.Table)
	cmd.SetArgs([]string{"auth", "login"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 5, apiErr.ExitCode)
}

func TestAuthLoginCmd_OverwritesExistingCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)

	// Pre-existing credentials
	oldCreds := &config.Credentials{ApiKey: "old-token"}
	require.NoError(t, config.SaveCredentials(oldCreds))

	ts := newTestServer("new-token")
	defer ts.Close()

	cmd, _, _ := buildAuthLoginCmd(ts.URL, "new-token", output.Table)
	cmd.SetArgs([]string{"auth", "login"})

	err := cmd.Execute()
	require.NoError(t, err)

	creds, err := config.LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "new-token", creds.ApiKey)
}

func TestAuthLoginCmd_TokenFromEnv(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)

	ts := newTestServer("env-token")
	defer ts.Close()

	// Token comes through the client (simulating env resolution in PersistentPreRunE)
	cmd, stdout, _ := buildAuthLoginCmd(ts.URL, "env-token", output.Table)
	cmd.SetArgs([]string{"auth", "login"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "Successfully authenticated.", strings.TrimSpace(stdout.String()))

	creds, err := config.LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "env-token", creds.ApiKey)
}

func TestAuthLoginCmd_PipedInput(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)

	ts := newTestServer("piped-token")
	defer ts.Close()

	// Override stdinFd to a non-terminal fd and stdinReader to our pipe
	origFd := stdinFd
	origReader := stdinReader
	t.Cleanup(func() {
		stdinFd = origFd
		stdinReader = origReader
	})

	// Use a pipe fd (not a terminal)
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer r.Close()

	stdinFd = int(r.Fd())
	stdinReader = r

	// Write token to pipe
	_, err = w.Write([]byte("piped-token\n"))
	require.NoError(t, err)
	w.Close()

	// No token in client — forces interactive prompt
	cmd, stdout, stderr := buildAuthLoginCmd(ts.URL, "", output.Table)
	cmd.SetArgs([]string{"auth", "login"})

	err = cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "Successfully authenticated.", strings.TrimSpace(stdout.String()))
	assert.Contains(t, stderr.String(), "Enter API token: ")

	creds, loadErr := config.LoadCredentials()
	require.NoError(t, loadErr)
	assert.Equal(t, "piped-token", creds.ApiKey)
}

func TestAuthLoginCmd_NoTokenProvided(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)

	// Override stdinFd/Reader to return empty input
	origFd := stdinFd
	origReader := stdinReader
	t.Cleanup(func() {
		stdinFd = origFd
		stdinReader = origReader
	})

	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer r.Close()

	stdinFd = int(r.Fd())
	stdinReader = r
	w.Close() // EOF immediately

	cmd, _, _ := buildAuthLoginCmd("http://localhost:0", "", output.Table)
	cmd.SetArgs([]string{"auth", "login"})

	err = cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
}

// Integration test: full flow with root command
func TestAuthLogin_Integration_ValidToken(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)

	ts := newTestServer("integration-token")
	defer ts.Close()

	// Write config with test server URL
	cfg := &config.Config{ApiURL: ts.URL}
	require.NoError(t, config.SaveConfig(cfg))

	root := buildRootWithAuth()
	stdout := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetArgs([]string{"--no-json", "auth", "login", "--token", "integration-token"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Equal(t, "Successfully authenticated.", strings.TrimSpace(stdout.String()))

	// Verify credentials file
	data, err := os.ReadFile(filepath.Join(tmpDir, "credentials.json"))
	require.NoError(t, err)

	var creds config.Credentials
	require.NoError(t, json.Unmarshal(data, &creds))
	assert.Equal(t, "integration-token", creds.ApiKey)
}

func TestAuthLogin_Integration_InvalidToken(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)

	ts := newTestServer("valid-token")
	defer ts.Close()

	cfg := &config.Config{ApiURL: ts.URL}
	require.NoError(t, config.SaveConfig(cfg))

	root := buildRootWithAuth()
	root.SetArgs([]string{"auth", "login", "--token", "wrong-token"})

	err := root.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
	assert.Equal(t, "Invalid API token.", apiErr.Message)

	// Credentials should NOT be saved
	_, err = os.Stat(filepath.Join(tmpDir, "credentials.json"))
	assert.True(t, os.IsNotExist(err))
}

func TestAuthLogin_Integration_EnvToken(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)
	t.Setenv("VECTOR_API_KEY", "env-integration-token")

	ts := newTestServer("env-integration-token")
	defer ts.Close()

	cfg := &config.Config{ApiURL: ts.URL}
	require.NoError(t, config.SaveConfig(cfg))

	root := buildRootWithAuth()
	stdout := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetArgs([]string{"auth", "login"})

	err := root.Execute()
	require.NoError(t, err)

	creds, err := config.LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "env-integration-token", creds.ApiKey)
}

// buildRootWithAuth creates a real root command (with PersistentPreRunE) + auth subcommand.
func buildRootWithAuth() *cobra.Command {
	root := &cobra.Command{
		Use: "vector",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			creds, err := config.LoadCredentials()
			if err != nil {
				return err
			}
			token, _ := cmd.Flags().GetString("token")
			if token == "" {
				token = os.Getenv("VECTOR_API_KEY")
			}
			if token == "" {
				token = creds.ApiKey
			}
			client := api.NewClient(cfg.ApiURL, token, "")
			jsonFlag, _ := cmd.Flags().GetBool("json")
			noJsonFlag, _ := cmd.Flags().GetBool("no-json")
			format := output.DetectFormat(jsonFlag, noJsonFlag)
			app := appctx.NewApp(cfg, creds, client, format)
			cmd.SetContext(appctx.WithApp(cmd.Context(), app))
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().String("token", "", "API token")
	root.PersistentFlags().Bool("json", false, "Force JSON output")
	root.PersistentFlags().Bool("no-json", false, "Force table output")

	root.AddCommand(NewAuthCmd())
	return root
}
