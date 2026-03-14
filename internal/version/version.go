// Package version provides build-time version information via ldflags injection.
package version

import "fmt"

// Version, Commit, and Date are set at build time via ldflags.
// Example: go build -ldflags "-X github.com/built-fast/vector-cli/internal/version.Version=1.0.0"
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// FullVersion returns a formatted version string.
func FullVersion() string {
	return fmt.Sprintf("vector v%s (%s) built %s", Version, Commit, Date)
}
