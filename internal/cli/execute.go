package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/commands"
)

// Execute creates the root command and runs it.
// It returns 0 on success, or an appropriate exit code on error.
func Execute() int {
	commands.RefreshSkillsIfVersionChanged()
	cmd := NewRootCmd()
	if err := cmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		var apiErr *api.APIError
		if errors.As(err, &apiErr) {
			return apiErr.ExitCode
		}
		return 1
	}
	return 0
}
