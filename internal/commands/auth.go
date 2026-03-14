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
			if app.Format == output.JSON {
				var raw json.RawMessage = body
				return output.PrintJSON(cmd.OutOrStdout(), raw)
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

			if app.Format == output.JSON {
				return output.PrintJSON(cmd.OutOrStdout(), map[string]string{
					"message": "Logged out successfully.",
				})
			}

			output.PrintMessage(cmd.OutOrStdout(), "Logged out successfully.")
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
