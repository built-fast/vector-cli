package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/appctx"
)

const (
	maxTimeout           = 30 * time.Minute
	minPollInterval      = 1 * time.Second
	maxConsecutiveErrors = 10
)

// waitConfig holds all parameters needed by waitForResource.
type waitConfig struct {
	// ResourceID is the identifier of the resource being waited on.
	ResourceID string

	// PollPath is the API path to GET for status checks.
	PollPath string

	// Interval is the duration between poll requests.
	Interval time.Duration

	// Timeout is the maximum duration to wait before giving up.
	Timeout time.Duration

	// TerminalStatuses is the set of statuses that indicate completion.
	TerminalStatuses map[string]bool

	// FailedStatuses is the set of statuses that indicate failure.
	FailedStatuses map[string]bool

	// Noun is a human-readable label for the resource (e.g., "Deployment", "Site").
	Noun string

	// FormatDisplay is an optional callback that formats poll data for display.
	FormatDisplay func(data map[string]any) []string
}

// waitResult holds the outcome of a wait operation.
type waitResult struct {
	// FinalData is the parsed response data from the last successful poll.
	FinalData json.RawMessage

	// Status is the terminal status that ended the wait.
	Status string

	// Elapsed is the total time spent waiting.
	Elapsed time.Duration
}

// addWaitFlags registers --wait, --poll-interval, and --timeout on a command.
func addWaitFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("wait", false, "Wait for the operation to complete")
	cmd.Flags().Duration("poll-interval", 60*time.Second, "Interval between status polls (minimum 1s)")
	cmd.Flags().Duration("timeout", 5*time.Minute, "Maximum time to wait (maximum 30m)")
}

// getWaitConfig reads wait-related flags from the command and returns a
// partially populated waitConfig. The caller must set ResourceID, PollPath,
// TerminalStatuses, FailedStatuses, Noun, and FormatDisplay.
func getWaitConfig(cmd *cobra.Command) (enabled bool, interval, timeout time.Duration, err error) {
	enabled, _ = cmd.Flags().GetBool("wait")
	interval, _ = cmd.Flags().GetDuration("poll-interval")
	timeout, _ = cmd.Flags().GetDuration("timeout")

	if !enabled {
		return false, 0, 0, nil
	}

	if interval < minPollInterval {
		return false, 0, 0, &api.APIError{
			Message:  fmt.Sprintf("poll interval must be at least %s", minPollInterval),
			ExitCode: 1,
		}
	}

	if timeout > maxTimeout {
		return false, 0, 0, &api.APIError{
			Message:  fmt.Sprintf("timeout must not exceed %s", maxTimeout),
			ExitCode: 1,
		}
	}

	if interval > timeout {
		return false, 0, 0, &api.APIError{
			Message:  "poll interval must not exceed timeout",
			ExitCode: 1,
		}
	}

	return enabled, interval, timeout, nil
}

// waitForResource polls the API until the resource reaches a terminal or failed
// status, the timeout expires, or the context is cancelled (e.g., Ctrl+C).
func waitForResource(ctx context.Context, app *appctx.App, cfg *waitConfig) (*waitResult, error) {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	deadline := time.After(cfg.Timeout)
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	start := time.Now()
	consecutiveErrors := 0
	var lastErr error

	for {
		select {
		case <-ctx.Done():
			return nil, &api.APIError{
				Message:  fmt.Sprintf("%s wait cancelled", cfg.Noun),
				ExitCode: 1,
			}
		case <-deadline:
			return nil, &api.APIError{
				Message:  fmt.Sprintf("timed out waiting for %s %s after %s", cfg.Noun, cfg.ResourceID, cfg.Timeout),
				ExitCode: 1,
			}
		case <-ticker.C:
			data, status, err := pollOnce(ctx, app, cfg)
			if err != nil {
				consecutiveErrors++
				lastErr = err
				if consecutiveErrors >= maxConsecutiveErrors {
					return nil, &api.APIError{
						Message:  fmt.Sprintf("aborting after %d consecutive poll failures for %s %s: %v", maxConsecutiveErrors, cfg.Noun, cfg.ResourceID, lastErr),
						ExitCode: 1,
					}
				}
				continue
			}

			consecutiveErrors = 0

			if cfg.FailedStatuses[status] {
				return nil, &api.APIError{
					Message:  fmt.Sprintf("%s %s reached failed status: %s", cfg.Noun, cfg.ResourceID, status),
					ExitCode: 1,
				}
			}

			if cfg.TerminalStatuses[status] {
				return &waitResult{
					FinalData: data,
					Status:    status,
					Elapsed:   time.Since(start),
				}, nil
			}
		}
	}
}

// pollOnce performs a single GET request and extracts the status field.
func pollOnce(ctx context.Context, app *appctx.App, cfg *waitConfig) (json.RawMessage, string, error) {
	resp, err := app.Client.Get(ctx, cfg.PollPath, nil)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read poll response: %w", err)
	}

	data, err := parseResponseData(body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse poll response: %w", err)
	}

	var item map[string]any
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal poll data: %w", err)
	}

	status := getString(item, "status")
	if status == "" {
		return nil, "", fmt.Errorf("poll response missing status field")
	}

	return data, status, nil
}
