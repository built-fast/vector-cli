package commands

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/appctx"
	"github.com/built-fast/vector-cli/internal/config"
	"github.com/built-fast/vector-cli/internal/output"
)

// buildDoctorCmd wires a root + doctor command with an App context. baseURL
// points the client at a test server; tokenSource mirrors how the real root
// resolves the token (flag/env/keyring) so the auth check can report it.
func buildDoctorCmd(baseURL, token, tokenSource string, format output.Format) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	root := &cobra.Command{
		Use: "vector",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.DefaultConfig()
			cfg.APIURL = baseURL
			client := api.NewClient(baseURL, token, "test-agent")
			app := appctx.NewApp(cfg, client, tokenSource)
			app.Output = output.NewWriter(stdout, format)
			cmd.SetContext(appctx.WithApp(cmd.Context(), app))
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(NewDoctorCmd())
	root.SetOut(stdout)
	root.SetErr(stderr)

	return root, stdout, stderr
}

func TestDoctorCmd_AllHealthy(t *testing.T) {
	srv := newTestServer("good-token")
	defer srv.Close()

	cmd, stdout, _ := buildDoctorCmd(srv.URL, "good-token", "keyring", output.Table)
	cmd.SetArgs([]string{"doctor"})

	require.NoError(t, cmd.Execute())

	out := stdout.String()
	assert.Contains(t, out, "OK")
	assert.Contains(t, out, "token from keyring")
	assert.Contains(t, out, "authenticated as john@example.com (Acme Inc)")
	assert.NotContains(t, out, "FAIL")
}

func TestDoctorCmd_AllHealthyJSON(t *testing.T) {
	srv := newTestServer("good-token")
	defer srv.Close()

	cmd, stdout, _ := buildDoctorCmd(srv.URL, "good-token", "env", output.JSON)
	cmd.SetArgs([]string{"doctor"})

	require.NoError(t, cmd.Execute())

	var result struct {
		OK     bool          `json:"ok"`
		APIURL string        `json:"api_url"`
		Checks []doctorCheck `json:"checks"`
	}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))

	assert.True(t, result.OK)
	assert.Equal(t, srv.URL, result.APIURL)
	require.Len(t, result.Checks, 3)

	byName := map[string]doctorCheck{}
	for _, c := range result.Checks {
		byName[c.Name] = c
	}
	assert.Equal(t, doctorPass, byName["cli"].Status)
	assert.Equal(t, doctorPass, byName["auth"].Status)
	assert.Equal(t, doctorPass, byName["api"].Status)
}

func TestDoctorCmd_NoToken(t *testing.T) {
	cmd, stdout, _ := buildDoctorCmd("http://localhost", "", "", output.JSON)
	cmd.SetArgs([]string{"doctor"})

	// Doctor reports health via status, never errors on a successful run.
	require.NoError(t, cmd.Execute())

	var result struct {
		OK     bool          `json:"ok"`
		Checks []doctorCheck `json:"checks"`
	}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))

	assert.False(t, result.OK)

	byName := map[string]doctorCheck{}
	for _, c := range result.Checks {
		byName[c.Name] = c
	}
	assert.Equal(t, doctorFail, byName["auth"].Status)
	assert.NotEmpty(t, byName["auth"].Hint)
	assert.Equal(t, doctorSkip, byName["api"].Status)
}

func TestDoctorCmd_InvalidToken(t *testing.T) {
	srv := newTestServer("good-token")
	defer srv.Close()

	cmd, stdout, _ := buildDoctorCmd(srv.URL, "wrong-token", "flag", output.JSON)
	cmd.SetArgs([]string{"doctor"})

	require.NoError(t, cmd.Execute())

	var result struct {
		OK     bool          `json:"ok"`
		Checks []doctorCheck `json:"checks"`
	}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))

	assert.False(t, result.OK)

	byName := map[string]doctorCheck{}
	for _, c := range result.Checks {
		byName[c.Name] = c
	}
	// Token is present, so auth passes; the API rejects it.
	assert.Equal(t, doctorPass, byName["auth"].Status)
	assert.Equal(t, doctorFail, byName["api"].Status)
	assert.Contains(t, byName["api"].Detail, "rejected")
}

func TestDoctorCmd_HelpText(t *testing.T) {
	cmd, stdout, _ := buildDoctorCmd("http://localhost", "", "", output.Table)
	cmd.SetArgs([]string{"doctor", "--help"})

	require.NoError(t, cmd.Execute())
	assert.Contains(t, stdout.String(), "health checks")
}
