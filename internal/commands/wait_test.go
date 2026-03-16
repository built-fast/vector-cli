package commands

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/built-fast/vector-cli/internal/api"
)

func TestAddWaitFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	addWaitFlags(cmd)

	waitFlag := cmd.Flags().Lookup("wait")
	require.NotNil(t, waitFlag)
	assert.Equal(t, "false", waitFlag.DefValue)

	intervalFlag := cmd.Flags().Lookup("poll-interval")
	require.NotNil(t, intervalFlag)
	assert.Equal(t, "1m0s", intervalFlag.DefValue)

	timeoutFlag := cmd.Flags().Lookup("timeout")
	require.NotNil(t, timeoutFlag)
	assert.Equal(t, "5m0s", timeoutFlag.DefValue)
}

func TestGetWaitConfig_Disabled(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	addWaitFlags(cmd)
	require.NoError(t, cmd.ParseFlags([]string{}))

	enabled, _, _, err := getWaitConfig(cmd)
	require.NoError(t, err)
	assert.False(t, enabled)
}

func TestGetWaitConfig_Enabled(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	addWaitFlags(cmd)
	require.NoError(t, cmd.ParseFlags([]string{"--wait", "--poll-interval", "5s", "--timeout", "2m"}))

	enabled, interval, timeout, err := getWaitConfig(cmd)
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.Equal(t, 5*time.Second, interval)
	assert.Equal(t, 2*time.Minute, timeout)
}

func TestGetWaitConfig_PollIntervalTooSmall(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	addWaitFlags(cmd)
	require.NoError(t, cmd.ParseFlags([]string{"--wait", "--poll-interval", "500ms"}))

	_, _, _, err := getWaitConfig(cmd)
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 1, apiErr.ExitCode)
	assert.Contains(t, apiErr.Message, "at least")
}

func TestGetWaitConfig_TimeoutTooLarge(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	addWaitFlags(cmd)
	require.NoError(t, cmd.ParseFlags([]string{"--wait", "--timeout", "31m"}))

	_, _, _, err := getWaitConfig(cmd)
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 1, apiErr.ExitCode)
	assert.Contains(t, apiErr.Message, "must not exceed")
}

func TestGetWaitConfig_IntervalExceedsTimeout(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	addWaitFlags(cmd)
	require.NoError(t, cmd.ParseFlags([]string{"--wait", "--poll-interval", "10m", "--timeout", "5m"}))

	_, _, _, err := getWaitConfig(cmd)
	require.Error(t, err)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 1, apiErr.ExitCode)
	assert.Contains(t, apiErr.Message, "must not exceed timeout")
}

// Compile-time references to satisfy the unused linter.
// Comprehensive tests for waitForResource are in US-007.
var (
	_ = (*waitConfig)(nil)
	_ = (*waitResult)(nil)
	_ = waitForResource
	_ = pollOnce
)
