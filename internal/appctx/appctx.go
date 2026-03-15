// Package appctx provides the App struct, context helpers, and global flags.
package appctx

import (
	"context"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/config"
	"github.com/built-fast/vector-cli/internal/output"
)

// contextKey is an unexported type for context keys to avoid collisions.
type contextKey struct{}

// App holds shared application state accessible to all commands.
type App struct {
	Config      *config.Config
	Credentials *config.Credentials
	Client      *api.Client
	Output      *output.Writer
	TokenSource string // "flag", "env", "keyring", or ""
}

// NewApp creates a new App with the given dependencies.
func NewApp(cfg *config.Config, creds *config.Credentials, client *api.Client, tokenSource string) *App {
	return &App{
		Config:      cfg,
		Credentials: creds,
		Client:      client,
		TokenSource: tokenSource,
	}
}

// WithApp stores an App in the context.
func WithApp(ctx context.Context, app *App) context.Context {
	return context.WithValue(ctx, contextKey{}, app)
}

// FromContext retrieves the App from the context. Returns nil if not set.
func FromContext(ctx context.Context) *App {
	app, _ := ctx.Value(contextKey{}).(*App)
	return app
}
