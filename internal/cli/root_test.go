package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/built-fast/vector-cli/internal/version"
)

func TestNewRootCmd_Use(t *testing.T) {
	cmd := NewRootCmd()
	if cmd.Use != "vector" {
		t.Errorf("expected Use = %q, got %q", "vector", cmd.Use)
	}
}

func TestNewRootCmd_VersionFlag(t *testing.T) {
	// Save and restore version vars.
	origVersion := version.Version
	origCommit := version.Commit
	origDate := version.Date
	t.Cleanup(func() {
		version.Version = origVersion
		version.Commit = origCommit
		version.Date = origDate
	})

	version.Version = "1.2.3"
	version.Commit = "abc1234"
	version.Date = "2026-01-01"

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	want := "vector v1.2.3 (abc1234) built 2026-01-01"
	if got != want {
		t.Errorf("--version output = %q, want %q", got, want)
	}
}

func TestNewRootCmd_JSONFlagRegistered(t *testing.T) {
	cmd := NewRootCmd()
	f := cmd.PersistentFlags().Lookup("json")
	if f == nil {
		t.Fatal("expected --json persistent flag to be registered")
	}
	if f.DefValue != "false" {
		t.Errorf("expected --json default = %q, got %q", "false", f.DefValue)
	}
}

func TestNewRootCmd_NoJSONFlagRegistered(t *testing.T) {
	cmd := NewRootCmd()
	f := cmd.PersistentFlags().Lookup("no-json")
	if f == nil {
		t.Fatal("expected --no-json persistent flag to be registered")
	}
	if f.DefValue != "false" {
		t.Errorf("expected --no-json default = %q, got %q", "false", f.DefValue)
	}
}

func TestNewRootCmd_HelpText(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	checks := []string{
		"vector",
		"Vector CLI",
		"--json",
		"--no-json",
		"--version",
	}

	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("help output missing %q", check)
		}
	}
}

func TestNewRootCmd_NoArgsShowsHelp(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Usage:") {
		t.Error("expected help output to contain 'Usage:'")
	}
}
