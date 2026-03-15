package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/appctx"
	"github.com/built-fast/vector-cli/internal/config"
	"github.com/built-fast/vector-cli/internal/output"
)

// stdinFd is the file descriptor used for reading terminal input.
// Override in tests to use a pipe instead.
var stdinFd = int(os.Stdin.Fd())

// stdinReader is the reader used for reading non-terminal (piped) input.
// Override in tests.
var stdinReader io.Reader = os.Stdin

// NewAuthCmd creates the auth command group.
func NewAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
		Long:  "Manage Vector API authentication including login, logout, and status.",
	}

	cmd.AddCommand(newAuthLoginCmd())
	cmd.AddCommand(newAuthLogoutCmd())
	cmd.AddCommand(newAuthStatusCmd())

	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with the Vector API",
		Long:  "Validate an API token via the ping endpoint and save it to credentials.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appctx.FromContext(cmd.Context())
			if app == nil {
				return fmt.Errorf("app not initialized")
			}

			token := app.Client.Token

			// If no token from flag/env/stored credentials, prompt interactively
			if token == "" {
				var err error
				token, err = promptForToken(cmd.ErrOrStderr())
				if err != nil {
					return err
				}
			}

			if token == "" {
				return &api.APIError{
					Message:    "No API token provided.",
					HTTPStatus: 401,
					ExitCode:   2,
				}
			}

			// Build a client with the provided token
			client := api.NewClient(app.Client.BaseURL, token, app.Client.UserAgent)

			// Validate via GET /api/v1/ping
			resp, err := client.Get(cmd.Context(), "/api/v1/ping", nil)
			if err != nil {
				var apiErr *api.APIError
				if errors.As(err, &apiErr) {
					if apiErr.HTTPStatus == 401 || apiErr.HTTPStatus == 403 {
						return &api.APIError{
							Message:    "Invalid API token.",
							HTTPStatus: apiErr.HTTPStatus,
							ExitCode:   2,
						}
					}
					return apiErr
				}
				// Network error
				return &api.APIError{
					Message:  fmt.Sprintf("Network error: %s", err),
					ExitCode: 5,
				}
			}
			defer func() { _ = resp.Body.Close() }()

			// Read and parse the response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("reading response: %w", err)
			}

			// Save credentials
			creds := &config.Credentials{ApiKey: token}
			if err := config.SaveCredentials(creds); err != nil {
				return fmt.Errorf("saving credentials: %w", err)
			}

			// Output
			if app.Output.Format() == output.JSON {
				var raw json.RawMessage = body
				return app.Output.JSON(raw)
			}

			output.PrintMessage(cmd.OutOrStdout(), "Successfully authenticated.")
			return nil
		},
	}
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored credentials",
		Long:  "Log out by deleting stored API credentials from disk.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appctx.FromContext(cmd.Context())
			if app == nil {
				return fmt.Errorf("app not initialized")
			}

			if err := config.ClearCredentials(); err != nil {
				return fmt.Errorf("clearing credentials: %w", err)
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(map[string]string{
					"message": "Logged out successfully.",
				})
			}

			output.PrintMessage(cmd.OutOrStdout(), "Logged out successfully.")
			return nil
		},
	}
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		Long:  "Check whether you are authenticated and display account details.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appctx.FromContext(cmd.Context())
			if app == nil {
				return fmt.Errorf("app not initialized")
			}

			// Not authenticated if no token
			if app.Client.Token == "" {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Not logged in.")
				return &api.APIError{
					Message:  "Not logged in.",
					ExitCode: 2,
				}
			}

			// Ping the API
			resp, err := app.Client.Get(cmd.Context(), "/api/v1/ping", nil)
			if err != nil {
				var apiErr *api.APIError
				if errors.As(err, &apiErr) {
					if apiErr.HTTPStatus == 401 || apiErr.HTTPStatus == 403 {
						_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Not logged in.")
						return &api.APIError{
							Message:  "Not logged in.",
							ExitCode: 2,
						}
					}
					return apiErr
				}
				return &api.APIError{
					Message:  fmt.Sprintf("Network error: %s", err),
					ExitCode: 5,
				}
			}
			defer func() { _ = resp.Body.Close() }()

			// Parse ping response to extract data.response
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("reading response: %w", err)
			}

			var parsed struct {
				Data struct {
					Response string `json:"response"`
				} `json:"data"`
			}
			if err := json.Unmarshal(body, &parsed); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			configDir, _ := config.ConfigDir()

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(map[string]any{
					"authenticated": true,
					"token_source":  app.TokenSource,
					"config_dir":    configDir,
					"api_url":       app.Config.ApiURL,
					"ping":          parsed.Data.Response,
				})
			}

			app.Output.KeyValue([]output.KeyValue{
				{Key: "Token source", Value: app.TokenSource},
				{Key: "Config directory", Value: configDir},
				{Key: "API URL", Value: app.Config.ApiURL},
				{Key: "Ping", Value: parsed.Data.Response},
			})
			return nil
		},
	}
}

// promptForToken prompts the user for an API token on stderr.
// If stdin is a terminal, input is masked. Otherwise, it reads plain text.
func promptForToken(w io.Writer) (string, error) {
	_, _ = fmt.Fprint(w, "Enter API token: ")

	if term.IsTerminal(stdinFd) {
		tokenBytes, err := term.ReadPassword(stdinFd)
		_, _ = fmt.Fprintln(w) // newline after masked input
		if err != nil {
			return "", fmt.Errorf("reading token: %w", err)
		}
		return string(tokenBytes), nil
	}

	// Non-terminal: read a line from stdin
	var token string
	buf := make([]byte, 4096)
	n, err := stdinReader.Read(buf)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("reading token: %w", err)
	}
	token = string(buf[:n])
	// Trim trailing newline
	if len(token) > 0 && token[len(token)-1] == '\n' {
		token = token[:len(token)-1]
	}
	if len(token) > 0 && token[len(token)-1] == '\r' {
		token = token[:len(token)-1]
	}
	return token, nil
}
