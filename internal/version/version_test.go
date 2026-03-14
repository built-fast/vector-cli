package version

import (
	"testing"
)

func TestDefaultValues(t *testing.T) {
	if Version != "dev" {
		t.Errorf("expected Version = %q, got %q", "dev", Version)
	}
	if Commit != "unknown" {
		t.Errorf("expected Commit = %q, got %q", "unknown", Commit)
	}
	if Date != "unknown" {
		t.Errorf("expected Date = %q, got %q", "unknown", Date)
	}
}

func TestFullVersion(t *testing.T) {
	// Save originals and restore after test
	origVersion, origCommit, origDate := Version, Commit, Date
	t.Cleanup(func() {
		Version, Commit, Date = origVersion, origCommit, origDate
	})

	tests := []struct {
		name    string
		version string
		commit  string
		date    string
		want    string
	}{
		{
			name:    "defaults",
			version: "dev",
			commit:  "unknown",
			date:    "unknown",
			want:    "vector vdev (unknown) built unknown",
		},
		{
			name:    "injected values",
			version: "1.0.0",
			commit:  "abc1234",
			date:    "2026-03-14",
			want:    "vector v1.0.0 (abc1234) built 2026-03-14",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Version, Commit, Date = tt.version, tt.commit, tt.date
			got := FullVersion()
			if got != tt.want {
				t.Errorf("FullVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}
