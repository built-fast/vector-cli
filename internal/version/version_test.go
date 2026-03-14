package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultValues(t *testing.T) {
	assert.Equal(t, "dev", Version)
	assert.Equal(t, "unknown", Commit)
	assert.Equal(t, "unknown", Date)
}

func TestFullVersion(t *testing.T) {
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
			assert.Equal(t, tt.want, FullVersion())
		})
	}
}
