package appctx_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/appctx"
	"github.com/built-fast/vector-cli/internal/config"
	"github.com/built-fast/vector-cli/internal/output"
)

func TestNewApp(t *testing.T) {
	cfg := &config.Config{ApiURL: "https://example.com"}
	creds := &config.Credentials{ApiKey: "test-key"}
	client := api.NewClient("https://example.com", "test-key", "")
	format := output.JSON

	app := appctx.NewApp(cfg, creds, client, format, "")

	require.NotNil(t, app)
	assert.Equal(t, cfg, app.Config)
	assert.Equal(t, creds, app.Credentials)
	assert.Equal(t, client, app.Client)
	assert.Equal(t, format, app.Format)
}

func TestContextRoundTrip(t *testing.T) {
	cfg := &config.Config{ApiURL: "https://example.com"}
	creds := &config.Credentials{ApiKey: "test-key"}
	client := api.NewClient("https://example.com", "test-key", "")
	app := appctx.NewApp(cfg, creds, client, output.Table, "")

	ctx := appctx.WithApp(context.Background(), app)
	got := appctx.FromContext(ctx)

	require.NotNil(t, got)
	assert.Equal(t, app, got)
}

func TestFromContext_NotSet(t *testing.T) {
	got := appctx.FromContext(context.Background())
	assert.Nil(t, got)
}

func TestFromContext_WrongType(t *testing.T) {
	// Using a different key type should not collide
	type otherKey struct{}
	ctx := context.WithValue(context.Background(), otherKey{}, "not an app")
	got := appctx.FromContext(ctx)
	assert.Nil(t, got)
}
