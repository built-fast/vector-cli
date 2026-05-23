package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/appctx"
	"github.com/built-fast/vector-cli/internal/config"
	"github.com/built-fast/vector-cli/internal/output"
)

// countingResponse defines an HTTP response for newCountingTestServer.
type countingResponse struct {
	httpStatus int
	body       map[string]any
}

// newCountingTestServer returns a test server that returns different responses
// on successive GET requests. After exhausting the response list, it repeats
// the last response.
func newCountingTestServer(validToken string, responses []countingResponse) *httptest.Server {
	var callCount atomic.Int64

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

		idx := int(callCount.Add(1)) - 1
		if idx >= len(responses) {
			idx = len(responses) - 1
		}

		resp := responses[idx]
		w.Header().Set("Content-Type", "application/json")
		if resp.httpStatus != 0 {
			w.WriteHeader(resp.httpStatus)
		}
		_ = json.NewEncoder(w).Encode(resp.body)
	}))
}

// newWaitApp creates an App wired to the given test server for wait tests.
func newWaitApp(baseURL, token string, format output.Format) (*appctx.App, *bytes.Buffer) {
	stdout := new(bytes.Buffer)
	client := api.NewClient(baseURL, token, "test-agent")
	app := appctx.NewApp(config.DefaultConfig(), client, "")
	app.Output = output.NewWriter(stdout, format)
	return app, stdout
}

// overrideWaitGlobals overrides isTerminalForWait and altScreenWriter for a test,
// restoring originals on cleanup.
func overrideWaitGlobals(t *testing.T, isTTY bool) *bytes.Buffer {
	t.Helper()
	origIsTerminal := isTerminalForWait
	origWriter := altScreenWriter
	altBuf := new(bytes.Buffer)
	isTerminalForWait = func() bool { return isTTY }
	altScreenWriter = altBuf
	t.Cleanup(func() {
		isTerminalForWait = origIsTerminal
		altScreenWriter = origWriter
	})
	return altBuf
}

// makeOKResponse creates a standard API envelope with the given status field value.
func makeOKResponse(status string) countingResponse {
	return countingResponse{
		httpStatus: http.StatusOK,
		body: map[string]any{
			"data": map[string]any{
				"id":     "res-001",
				"status": status,
				"name":   "test-resource",
			},
			"message":     "Resource retrieved",
			"http_status": 200,
		},
	}
}

// makeErrorResponse creates a 500 server error response.
func makeErrorResponse() countingResponse {
	return countingResponse{
		httpStatus: http.StatusInternalServerError,
		body: map[string]any{
			"message":     "Internal Server Error",
			"http_status": 500,
		},
	}
}

// baseWaitConfig returns a waitConfig with short intervals suitable for tests.
func baseWaitConfig() *waitConfig {
	return &waitConfig{
		ResourceID:       "res-001",
		PollPath:         "/api/v1/vector/test/res-001",
		Interval:         10 * time.Millisecond,
		Timeout:          500 * time.Millisecond,
		TerminalStatuses: map[string]bool{"active": true, "deployed": true, "completed": true},
		FailedStatuses:   map[string]bool{"failed": true},
		Noun:             "Resource",
		FormatDisplay: func(data map[string]any) []string {
			return []string{"Status: " + getString(data, "status")}
		},
	}
}

// --- Flag registration and validation tests ---

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

// --- waitForResource tests ---

func TestWaitForResource_CompletesOnTerminalStatus(t *testing.T) {
	overrideWaitGlobals(t, false)

	ts := newCountingTestServer("test-token", []countingResponse{
		makeOKResponse("pending"),
		makeOKResponse("pending"),
		makeOKResponse("active"),
	})
	defer ts.Close()

	app, _ := newWaitApp(ts.URL, "test-token", output.Table)
	cfg := baseWaitConfig()
	cfg.PollPath = "/api/v1/vector/test/res-001"

	result, err := waitForResource(context.Background(), app, cfg)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "active", result.Status)
	assert.NotZero(t, result.Elapsed)

	var item map[string]any
	require.NoError(t, json.Unmarshal(result.FinalData, &item))
	assert.Equal(t, "res-001", item["id"])
}

func TestWaitForResource_DetectsFailureStatus(t *testing.T) {
	overrideWaitGlobals(t, false)

	ts := newCountingTestServer("test-token", []countingResponse{
		makeOKResponse("pending"),
		makeOKResponse("failed"),
	})
	defer ts.Close()

	app, _ := newWaitApp(ts.URL, "test-token", output.Table)
	cfg := baseWaitConfig()

	result, err := waitForResource(context.Background(), app, cfg)
	require.Error(t, err)
	assert.Nil(t, result)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 1, apiErr.ExitCode)
	assert.Contains(t, apiErr.Message, "failed status")
	assert.Contains(t, apiErr.Message, "failed")
}

func TestWaitForResource_HandlesTransientPollErrors(t *testing.T) {
	overrideWaitGlobals(t, false)

	ts := newCountingTestServer("test-token", []countingResponse{
		makeOKResponse("pending"),
		makeErrorResponse(),        // 500 on 2nd poll
		makeOKResponse("deployed"), // success on 3rd
	})
	defer ts.Close()

	app, _ := newWaitApp(ts.URL, "test-token", output.Table)
	cfg := baseWaitConfig()

	result, err := waitForResource(context.Background(), app, cfg)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "deployed", result.Status)
}

func TestWaitForResource_RespectsContextCancellation(t *testing.T) {
	overrideWaitGlobals(t, false)

	// Server always returns pending — the context cancellation should stop it.
	ts := newCountingTestServer("test-token", []countingResponse{
		makeOKResponse("pending"),
	})
	defer ts.Close()

	app, _ := newWaitApp(ts.URL, "test-token", output.Table)
	cfg := baseWaitConfig()
	cfg.Timeout = 5 * time.Second // long enough that timeout doesn't fire first

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after a short delay to simulate Ctrl+C.
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	result, err := waitForResource(ctx, app, cfg)
	require.Error(t, err)
	assert.Nil(t, result)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 1, apiErr.ExitCode)
	assert.Contains(t, apiErr.Message, "canceled")
}

func TestWaitForResource_TimesOut(t *testing.T) {
	overrideWaitGlobals(t, false)

	// Server always returns pending — should time out.
	ts := newCountingTestServer("test-token", []countingResponse{
		makeOKResponse("pending"),
	})
	defer ts.Close()

	app, _ := newWaitApp(ts.URL, "test-token", output.Table)
	cfg := baseWaitConfig()
	cfg.Interval = 10 * time.Millisecond
	cfg.Timeout = 50 * time.Millisecond

	result, err := waitForResource(context.Background(), app, cfg)
	require.Error(t, err)
	assert.Nil(t, result)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 1, apiErr.ExitCode)
	assert.Contains(t, apiErr.Message, "timed out")
}

func TestWaitForResource_AbortsAfterMaxConsecutiveFailures(t *testing.T) {
	overrideWaitGlobals(t, false)

	// Build a response list with maxConsecutiveErrors 500s.
	responses := make([]countingResponse, maxConsecutiveErrors)
	for i := range responses {
		responses[i] = makeErrorResponse()
	}

	ts := newCountingTestServer("test-token", responses)
	defer ts.Close()

	app, _ := newWaitApp(ts.URL, "test-token", output.Table)
	cfg := baseWaitConfig()
	cfg.Timeout = 5 * time.Second // long enough that timeout doesn't fire first

	result, err := waitForResource(context.Background(), app, cfg)
	require.Error(t, err)
	assert.Nil(t, result)

	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 1, apiErr.ExitCode)
	assert.Contains(t, apiErr.Message, "consecutive poll failures")
}

func TestWaitForResource_JSONModeNoANSI(t *testing.T) {
	altBuf := overrideWaitGlobals(t, true) // TTY=true, but JSON mode should suppress

	ts := newCountingTestServer("test-token", []countingResponse{
		makeOKResponse("pending"),
		makeOKResponse("active"),
	})
	defer ts.Close()

	app, _ := newWaitApp(ts.URL, "test-token", output.JSON)
	cfg := baseWaitConfig()

	result, err := waitForResource(context.Background(), app, cfg)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "active", result.Status)

	// Alt screen buffer should have no ANSI sequences.
	altOutput := altBuf.String()
	assert.NotContains(t, altOutput, "\033[")
}

func TestWaitForResource_NonTTYNoANSI(t *testing.T) {
	altBuf := overrideWaitGlobals(t, false) // Non-TTY

	ts := newCountingTestServer("test-token", []countingResponse{
		makeOKResponse("pending"),
		makeOKResponse("completed"),
	})
	defer ts.Close()

	app, _ := newWaitApp(ts.URL, "test-token", output.Table)
	cfg := baseWaitConfig()

	result, err := waitForResource(context.Background(), app, cfg)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "completed", result.Status)

	// Alt screen buffer should have no ANSI sequences.
	altOutput := altBuf.String()
	assert.NotContains(t, altOutput, "\033[")
}

func TestWaitForResource_TTYWritesANSI(t *testing.T) {
	altBuf := overrideWaitGlobals(t, true) // TTY + Table format

	ts := newCountingTestServer("test-token", []countingResponse{
		makeOKResponse("pending"),
		makeOKResponse("active"),
	})
	defer ts.Close()

	app, _ := newWaitApp(ts.URL, "test-token", output.Table)
	cfg := baseWaitConfig()

	result, err := waitForResource(context.Background(), app, cfg)
	require.NoError(t, err)
	require.NotNil(t, result)

	altOutput := altBuf.String()
	// TTY mode should have alt screen enter and exit sequences.
	assert.Contains(t, altOutput, ansiAltScreenEnter, "expected alt screen enter sequence")
	assert.Contains(t, altOutput, ansiAltScreenExit, "expected alt screen exit sequence")
}
