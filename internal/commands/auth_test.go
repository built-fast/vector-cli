package commands

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/appctx"
	"github.com/built-fast/vector-cli/internal/config"
	"github.com/built-fast/vector-cli/internal/output"
)

// whoamiTestResponse is the standard response from GET /api/v1/auth/whoami.
var whoamiTestResponse = map[string]any{
	"data": map[string]any{
		"user": map[string]any{
			"id":    1,
			"name":  "John Doe",
			"email": "john@example.com",
		},
		"token": map[string]any{
			"name":         "vector-cli",
			"abilities":    []string{"*"},
			"expires_at":   nil,
			"last_used_at": "2026-03-14T12:00:00.000000Z",
		},
		"account": map[string]any{
			"id":   1,
			"name": "Acme Inc",
		},
	},
	"message":     "Success",
	"http_status": 200,
}

// newTestServer creates an httptest server that responds to /api/v1/auth/whoami.
// validToken is the token that triggers a 200; anything else gets 401.
func newTestServer(validToken string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/whoami" {
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
		_ = json.NewEncoder(w).Encode(whoamiTestResponse)
	}))
}

// buildAuthLoginCmd creates a root + auth + login command wired with an App context.
func buildAuthLoginCmd(baseURL, token string, format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	stdout := new(bytes.Buffer)
	root := &cobra.Command{
		Use: "vector",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			client := api.NewClient(baseURL, token, "test-agent")
			app := appctx.NewApp(
				config.DefaultConfig(),
				&config.Credentials{},
				client,
				"",
			)
			app.Output = output.NewWriter(stdout, format)
			cmd.SetContext(appctx.WithApp(cmd.Context(), app))
			return nil
		},
	}

	authCmd := NewAuthCmd()
	root.AddCommand(authCmd)

	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

func TestAuthLoginCmd_ValidToken_TableOutput(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())
	t.Setenv("VECTOR_NO_KEYRING", "")

	ts := newTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAuthLoginCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"auth", "login"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "Authenticated as john@example.com (Acme Inc). Token stored in system keyring.", strings.TrimSpace(stdout.String()))

	// Verify credentials were saved
	creds, err := config.LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "valid-token", creds.ApiKey)
}

func TestAuthLoginCmd_ValidToken_JSONOutput(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())
	t.Setenv("VECTOR_NO_KEYRING", "")

	ts := newTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAuthLoginCmd(ts.URL, "valid-token", output.JSON)
	cmd.SetArgs([]string{"auth", "login"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	data := result["data"].(map[string]any)
	user := data["user"].(map[string]any)
	assert.Equal(t, "john@example.com", user["email"])
	assert.Equal(t, "Acme Inc", data["account"].(map[string]any)["name"])
	assert.Equal(t, "Success", result["message"])
}

func TestAuthLoginCmd_InvalidToken(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())
	t.Setenv("VECTOR_NO_KEYRING", "")

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
	keyring.MockInit()
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)
	t.Setenv("VECTOR_NO_KEYRING", "")

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

func TestAuthLoginCmd_KeyringDisabled(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())
	t.Setenv("VECTOR_NO_KEYRING", "1")

	ts := newTestServer("valid-token")
	defer ts.Close()

	cmd, _, _ := buildAuthLoginCmd(ts.URL, "valid-token", output.Table)
	cmd.SetArgs([]string{"auth", "login"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot store token: keyring is disabled")
	assert.Contains(t, err.Error(), "--token flag or VECTOR_API_KEY environment variable")
}

func TestAuthLoginCmd_TokenFromEnv(t *testing.T) {
	keyring.MockInit()
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)
	t.Setenv("VECTOR_NO_KEYRING", "")

	ts := newTestServer("env-token")
	defer ts.Close()

	// Token comes through the client (simulating env resolution in PersistentPreRunE)
	cmd, stdout, _ := buildAuthLoginCmd(ts.URL, "env-token", output.Table)
	cmd.SetArgs([]string{"auth", "login"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "Authenticated as john@example.com (Acme Inc). Token stored in system keyring.", strings.TrimSpace(stdout.String()))

	creds, err := config.LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "env-token", creds.ApiKey)
}

func TestAuthLoginCmd_PipedInput(t *testing.T) {
	keyring.MockInit()
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)
	t.Setenv("VECTOR_NO_KEYRING", "")

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
	assert.Equal(t, "Authenticated as john@example.com (Acme Inc). Token stored in system keyring.", strings.TrimSpace(stdout.String()))
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
	keyring.MockInit()
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)
	t.Setenv("VECTOR_NO_KEYRING", "")

	ts := newTestServer("integration-token")
	defer ts.Close()

	// Write config with test server URL
	cfg := &config.Config{ApiURL: ts.URL}
	require.NoError(t, config.SaveConfig(cfg))

	root, stdout := buildRootWithAuth()
	root.SetArgs([]string{"--no-json", "auth", "login", "--token", "integration-token"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Equal(t, "Authenticated as john@example.com (Acme Inc). Token stored in system keyring.", strings.TrimSpace(stdout.String()))

	// Verify credentials stored in keyring
	creds, err := config.LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "integration-token", creds.ApiKey)
}

func TestAuthLogin_Integration_InvalidToken(t *testing.T) {
	keyring.MockInit()
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)
	t.Setenv("VECTOR_NO_KEYRING", "")

	ts := newTestServer("valid-token")
	defer ts.Close()

	cfg := &config.Config{ApiURL: ts.URL}
	require.NoError(t, config.SaveConfig(cfg))

	root, _ := buildRootWithAuth()
	root.SetArgs([]string{"auth", "login", "--token", "wrong-token"})

	err := root.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
	assert.Equal(t, "Invalid API token.", apiErr.Message)

	// Credentials should NOT be saved
	creds, err := config.LoadCredentials()
	require.NoError(t, err)
	assert.Empty(t, creds.ApiKey)
}

func TestAuthLogin_Integration_EnvToken(t *testing.T) {
	keyring.MockInit()
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)
	t.Setenv("VECTOR_API_KEY", "env-integration-token")
	t.Setenv("VECTOR_NO_KEYRING", "")

	ts := newTestServer("env-integration-token")
	defer ts.Close()

	cfg := &config.Config{ApiURL: ts.URL}
	require.NoError(t, config.SaveConfig(cfg))

	root, _ := buildRootWithAuth()
	root.SetArgs([]string{"auth", "login"})

	err := root.Execute()
	require.NoError(t, err)

	creds, err := config.LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "env-integration-token", creds.ApiKey)
}

// --- Auth Logout Tests ---

func buildAuthLogoutCmd(format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	stdout := new(bytes.Buffer)
	root := &cobra.Command{
		Use: "vector",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			client := api.NewClient("http://localhost", "", "test-agent")
			app := appctx.NewApp(
				config.DefaultConfig(),
				&config.Credentials{},
				client,
				"",
			)
			app.Output = output.NewWriter(stdout, format)
			cmd.SetContext(appctx.WithApp(cmd.Context(), app))
			return nil
		},
	}

	authCmd := NewAuthCmd()
	root.AddCommand(authCmd)

	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

func TestAuthLogoutCmd_TableOutput(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())
	t.Setenv("VECTOR_NO_KEYRING", "")

	// Save credentials first
	require.NoError(t, config.SaveCredentials(&config.Credentials{ApiKey: "some-token"}))

	cmd, stdout, _ := buildAuthLogoutCmd(output.Table)
	cmd.SetArgs([]string{"auth", "logout"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "Logged out successfully. Token removed from system keyring.", strings.TrimSpace(stdout.String()))

	// Verify credentials were removed from keyring
	creds, err := config.LoadCredentials()
	require.NoError(t, err)
	assert.Empty(t, creds.ApiKey)
}

func TestAuthLogoutCmd_JSONOutput(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())
	t.Setenv("VECTOR_NO_KEYRING", "")

	require.NoError(t, config.SaveCredentials(&config.Credentials{ApiKey: "some-token"}))

	cmd, stdout, _ := buildAuthLogoutCmd(output.JSON)
	cmd.SetArgs([]string{"auth", "logout"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]string
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "Logged out successfully. Token removed from system keyring.", result["message"])
}

func TestAuthLogoutCmd_AlreadyLoggedOut(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())
	t.Setenv("VECTOR_NO_KEYRING", "")

	// No credentials stored — should succeed silently
	cmd, stdout, _ := buildAuthLogoutCmd(output.Table)
	cmd.SetArgs([]string{"auth", "logout"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "Logged out successfully. Token removed from system keyring.", strings.TrimSpace(stdout.String()))
}

func TestAuthLogoutCmd_KeyringDisabled(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())
	t.Setenv("VECTOR_NO_KEYRING", "1")

	cmd, stdout, _ := buildAuthLogoutCmd(output.Table)
	cmd.SetArgs([]string{"auth", "logout"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Equal(t, "Keyring is disabled. No stored credentials to remove.", strings.TrimSpace(stdout.String()))
}

func TestAuthLogoutCmd_KeyringDisabled_JSONOutput(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())
	t.Setenv("VECTOR_NO_KEYRING", "1")

	cmd, stdout, _ := buildAuthLogoutCmd(output.JSON)
	cmd.SetArgs([]string{"auth", "logout"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]string
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "Keyring is disabled. No stored credentials to remove.", result["message"])
}

func TestAuthLogout_Integration_RemovesCredentials(t *testing.T) {
	keyring.MockInit()
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)
	t.Setenv("VECTOR_NO_KEYRING", "")

	// Save config and credentials
	require.NoError(t, config.SaveConfig(&config.Config{ApiURL: "http://localhost"}))
	require.NoError(t, config.SaveCredentials(&config.Credentials{ApiKey: "test-token"}))

	// Verify credentials exist in keyring
	creds, err := config.LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "test-token", creds.ApiKey)

	root, stdout := buildRootWithAuth()
	root.SetArgs([]string{"--no-json", "auth", "logout"})

	err = root.Execute()
	require.NoError(t, err)
	assert.Equal(t, "Logged out successfully. Token removed from system keyring.", strings.TrimSpace(stdout.String()))

	// Credentials should be gone from keyring
	creds, err = config.LoadCredentials()
	require.NoError(t, err)
	assert.Empty(t, creds.ApiKey)
}

// buildRootWithAuth creates a real root command (with PersistentPreRunE) + auth subcommand.
func buildRootWithAuth() (*cobra.Command, *bytes.Buffer) {
	stdout := new(bytes.Buffer)
	root := &cobra.Command{
		Use: "vector",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			var token, tokenSource string
			token, _ = cmd.Flags().GetString("token")
			if token != "" {
				tokenSource = "flag"
			}
			if token == "" {
				token = os.Getenv("VECTOR_API_KEY")
				if token != "" {
					tokenSource = "env"
				}
			}
			if token == "" {
				if t, err := config.Load(); err == nil && t != "" {
					token = t
					tokenSource = "keyring"
				}
			}
			creds := &config.Credentials{ApiKey: token}
			client := api.NewClient(cfg.ApiURL, token, "")
			jsonFlag, _ := cmd.Flags().GetBool("json")
			noJsonFlag, _ := cmd.Flags().GetBool("no-json")
			format := output.DetectFormat(jsonFlag, noJsonFlag)
			app := appctx.NewApp(cfg, creds, client, tokenSource)
			app.Output = output.NewWriter(stdout, format)
			cmd.SetContext(appctx.WithApp(cmd.Context(), app))
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().String("token", "", "API token")
	root.PersistentFlags().Bool("json", false, "Force JSON output")
	root.PersistentFlags().Bool("no-json", false, "Force table output")

	root.SetOut(stdout)
	root.AddCommand(NewAuthCmd())
	return root, stdout
}

// --- Auth Status Tests ---

// buildAuthStatusCmd creates a root + auth + status command wired with an App context.
func buildAuthStatusCmd(baseURL, token, tokenSource string, format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	stdout := new(bytes.Buffer)
	root := &cobra.Command{
		Use: "vector",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.DefaultConfig()
			cfg.ApiURL = baseURL
			client := api.NewClient(baseURL, token, "test-agent")
			app := appctx.NewApp(
				cfg,
				&config.Credentials{ApiKey: token},
				client,
				tokenSource,
			)
			app.Output = output.NewWriter(stdout, format)
			cmd.SetContext(appctx.WithApp(cmd.Context(), app))
			return nil
		},
	}

	authCmd := NewAuthCmd()
	root.AddCommand(authCmd)

	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

func TestAuthStatusCmd_Authenticated_TableOutput(t *testing.T) {
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())

	ts := newTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAuthStatusCmd(ts.URL, "valid-token", "keyring", output.Table)
	cmd.SetArgs([]string{"auth", "status"})

	err := cmd.Execute()
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "John Doe (john@example.com)")
	assert.Contains(t, out, "Acme Inc")
	assert.Contains(t, out, "vector-cli")
	assert.Contains(t, out, "keyring")
	assert.Contains(t, out, ts.URL)
}

func TestAuthStatusCmd_Authenticated_JSONOutput(t *testing.T) {
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())

	ts := newTestServer("valid-token")
	defer ts.Close()

	cmd, stdout, _ := buildAuthStatusCmd(ts.URL, "valid-token", "flag", output.JSON)
	cmd.SetArgs([]string{"auth", "status"})

	err := cmd.Execute()
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, true, result["authenticated"])
	assert.Equal(t, "flag", result["token_source"])
	assert.Equal(t, ts.URL, result["api_url"])
	assert.NotEmpty(t, result["config_dir"])

	user := result["user"].(map[string]any)
	assert.Equal(t, "john@example.com", user["email"])
	assert.Equal(t, "John Doe", user["name"])

	account := result["account"].(map[string]any)
	assert.Equal(t, "Acme Inc", account["name"])

	token := result["token"].(map[string]any)
	assert.Equal(t, "vector-cli", token["name"])
}

func TestAuthStatusCmd_NotAuthenticated(t *testing.T) {
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())

	cmd, _, stderr := buildAuthStatusCmd("http://localhost", "", "", output.Table)
	cmd.SetArgs([]string{"auth", "status"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
	assert.Contains(t, stderr.String(), "Not logged in.")
}

func TestAuthStatusCmd_InvalidToken(t *testing.T) {
	t.Setenv("VECTOR_CONFIG_DIR", t.TempDir())

	ts := newTestServer("valid-token")
	defer ts.Close()

	cmd, _, stderr := buildAuthStatusCmd(ts.URL, "bad-token", "keyring", output.Table)
	cmd.SetArgs([]string{"auth", "status"})

	err := cmd.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
	assert.Contains(t, stderr.String(), "Not logged in.")
}

// Integration test: login → status → logout → status
func TestAuthStatus_Integration_FullFlow(t *testing.T) {
	keyring.MockInit()
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)
	t.Setenv("VECTOR_NO_KEYRING", "")

	ts := newTestServer("flow-token")
	defer ts.Close()

	// Save config with test server URL
	cfg := &config.Config{ApiURL: ts.URL}
	require.NoError(t, config.SaveConfig(cfg))

	// Step 1: Login
	root, stdout := buildRootWithAuth()
	root.SetArgs([]string{"--no-json", "auth", "login", "--token", "flow-token"})
	require.NoError(t, root.Execute())
	assert.Equal(t, "Authenticated as john@example.com (Acme Inc). Token stored in system keyring.", strings.TrimSpace(stdout.String()))

	// Step 2: Status shows authenticated
	root2, stdout2 := buildRootWithAuth()
	root2.SetArgs([]string{"--no-json", "auth", "status"})
	require.NoError(t, root2.Execute())

	out := stdout2.String()
	assert.Contains(t, out, "John Doe (john@example.com)")
	assert.Contains(t, out, "Acme Inc")
	assert.Contains(t, out, "keyring")
	assert.Contains(t, out, ts.URL)

	// Step 3: Logout
	root3, stdout3 := buildRootWithAuth()
	root3.SetArgs([]string{"--no-json", "auth", "logout"})
	require.NoError(t, root3.Execute())
	assert.Equal(t, "Logged out successfully. Token removed from system keyring.", strings.TrimSpace(stdout3.String()))

	// Step 4: Status shows not authenticated
	root4, _ := buildRootWithAuth()
	stderr4 := new(bytes.Buffer)
	root4.SetErr(stderr4)
	root4.SetArgs([]string{"--no-json", "auth", "status"})
	err := root4.Execute()
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 2, apiErr.ExitCode)
	assert.Contains(t, stderr4.String(), "Not logged in.")
}
