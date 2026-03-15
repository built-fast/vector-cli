package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
	"golang.org/x/term"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/appctx"
	"github.com/built-fast/vector-cli/internal/config"
	"github.com/built-fast/vector-cli/internal/output"
)

// whoamiResponse represents the parsed response from GET /api/v1/auth/whoami.
type whoamiResponse struct {
	Data struct {
		User struct {
			ID    int    `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"user"`
		Token struct {
			Name       string   `json:"name"`
			Abilities  []string `json:"abilities"`
			ExpiresAt  *string  `json:"expires_at"`
			LastUsedAt *string  `json:"last_used_at"`
		} `json:"token"`
		Account struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"account"`
	} `json:"data"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"http_status"`
}

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
		Long:  "Validate an API token and store it in the system keyring.",
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

			// Validate via GET /api/v1/auth/whoami
			resp, err := client.Get(cmd.Context(), "/api/v1/auth/whoami", nil)
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

			var whoami whoamiResponse
			if err := json.Unmarshal(body, &whoami); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			// Save token to system keyring
			if err := config.Save(token); err != nil {
				if errors.Is(err, config.ErrKeyringDisabled) {
					return fmt.Errorf("cannot store token: keyring is disabled. Use --token flag or VECTOR_API_KEY environment variable instead")
				}
				return fmt.Errorf("saving token: %w", err)
			}

			// Output
			if app.Output.Format() == output.JSON {
				var raw json.RawMessage = body
				return app.Output.JSON(raw)
			}

			output.PrintMessage(cmd.OutOrStdout(), fmt.Sprintf(
				"Authenticated as %s (%s). Token stored in system keyring.",
				whoami.Data.User.Email, whoami.Data.Account.Name,
			))
			return nil
		},
	}
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored credentials",
		Long:  "Log out by deleting stored API credentials from the system keyring.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appctx.FromContext(cmd.Context())
			if app == nil {
				return fmt.Errorf("app not initialized")
			}

			if err := config.Delete(); err != nil {
				if errors.Is(err, config.ErrKeyringDisabled) {
					msg := "Keyring is disabled. No stored credentials to remove."
					if app.Output.Format() == output.JSON {
						return app.Output.JSON(map[string]string{
							"message": msg,
						})
					}
					output.PrintMessage(cmd.OutOrStdout(), msg)
					return nil
				}
				if !errors.Is(err, keyring.ErrNotFound) {
					return fmt.Errorf("clearing token: %w", err)
				}
			}

			msg := "Logged out successfully. Token removed from system keyring."
			if app.Output.Format() == output.JSON {
				return app.Output.JSON(map[string]string{
					"message": msg,
				})
			}

			output.PrintMessage(cmd.OutOrStdout(), msg)
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

			// Validate via GET /api/v1/auth/whoami
			resp, err := app.Client.Get(cmd.Context(), "/api/v1/auth/whoami", nil)
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

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("reading response: %w", err)
			}

			var whoami whoamiResponse
			if err := json.Unmarshal(body, &whoami); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			configDir, _ := config.ConfigDir()

			expires := "Never"
			if whoami.Data.Token.ExpiresAt != nil {
				expires = *whoami.Data.Token.ExpiresAt
			}

			if app.Output.Format() == output.JSON {
				return app.Output.JSON(map[string]any{
					"authenticated": true,
					"user":          whoami.Data.User,
					"token":         whoami.Data.Token,
					"account":       whoami.Data.Account,
					"token_source":  app.TokenSource,
					"config_dir":    configDir,
					"api_url":       app.Config.ApiURL,
				})
			}

			app.Output.KeyValue([]output.KeyValue{
				{Key: "User", Value: fmt.Sprintf("%s (%s)", whoami.Data.User.Name, whoami.Data.User.Email)},
				{Key: "Account", Value: whoami.Data.Account.Name},
				{Key: "Token", Value: whoami.Data.Token.Name},
				{Key: "Abilities", Value: strings.Join(whoami.Data.Token.Abilities, ", ")},
				{Key: "Expires", Value: expires},
				{Key: "Token source", Value: app.TokenSource},
				{Key: "API URL", Value: app.Config.ApiURL},
				{Key: "Config directory", Value: configDir},
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
