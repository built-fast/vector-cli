package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/appctx"
	"github.com/built-fast/vector-cli/internal/output"
	"github.com/built-fast/vector-cli/internal/version"
)

// Doctor check statuses, mirrored in the /vector:doctor plugin command.
const (
	doctorPass = "pass" // working correctly
	doctorWarn = "warn" // non-critical issue
	doctorSkip = "skip" // check not run (e.g. unauthenticated)
	doctorFail = "fail" // broken, needs attention
)

// doctorCheck is a single diagnostic result.
type doctorCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
	Hint   string `json:"hint,omitempty"`
}

// NewDoctorCmd creates the doctor command.
func NewDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose CLI setup, authentication, and API connectivity",
		Long: "Run health checks on the Vector CLI: binary version, configured " +
			"authentication, and live API connectivity. Backs the Claude Code " +
			"/vector:doctor command and is useful for troubleshooting. Always exits 0 " +
			"on a successful run; health is reported in the status of each check.",
		Example: `  # Run all health checks
  vector doctor

  # Machine-readable output (used by /vector:doctor)
  vector doctor --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(cmd)
		},
	}
}

// runDoctor gathers the health checks and renders them. It does not use
// requireApp: diagnosing the unauthenticated state is part of its job.
func runDoctor(cmd *cobra.Command) error {
	app := appctx.FromContext(cmd.Context())
	if app == nil {
		return fmt.Errorf("app not initialized")
	}

	auth := doctorAuthCheck(app)
	checks := []doctorCheck{
		doctorCLICheck(),
		auth,
		doctorAPICheck(cmd, app, auth.Status == doctorPass),
	}

	ok := true
	for _, c := range checks {
		if c.Status == doctorFail {
			ok = false
		}
	}

	if app.Output.Format() == output.JSON {
		return app.Output.JSON(map[string]any{
			"ok":      ok,
			"api_url": app.Config.APIURL,
			"checks":  checks,
		})
	}

	rows := make([][]string, 0, len(checks))
	for _, c := range checks {
		rows = append(rows, []string{c.Name, doctorStatusLabel(c.Status), c.Detail})
	}
	app.Output.Table([]string{"CHECK", "STATUS", "DETAIL"}, rows)

	for _, c := range checks {
		if c.Hint != "" {
			app.Output.Message(fmt.Sprintf("→ %s: %s", c.Name, c.Hint))
		}
	}

	return nil
}

// doctorCLICheck reports the running binary version; it never fails.
func doctorCLICheck() doctorCheck {
	return doctorCheck{
		Name:   "cli",
		Status: doctorPass,
		Detail: version.FullVersion(),
	}
}

// doctorAuthCheck verifies a token is configured, without contacting the API.
func doctorAuthCheck(app *appctx.App) doctorCheck {
	if app.Client.Token == "" {
		return doctorCheck{
			Name:   "auth",
			Status: doctorFail,
			Detail: "no API token configured",
			Hint:   "run 'vector auth login', pass --token, or set VECTOR_API_KEY",
		}
	}

	source := app.TokenSource
	if source == "" {
		source = "unknown source"
	}
	return doctorCheck{
		Name:   "auth",
		Status: doctorPass,
		Detail: "token from " + source,
	}
}

// doctorAPICheck validates the token against the live API. It is skipped when
// no token is configured (the auth check already reported that).
func doctorAPICheck(cmd *cobra.Command, app *appctx.App, haveToken bool) doctorCheck {
	if !haveToken {
		return doctorCheck{
			Name:   "api",
			Status: doctorSkip,
			Detail: "skipped (not authenticated)",
		}
	}

	resp, err := app.Client.Get(cmd.Context(), "/api/v1/auth/whoami", nil)
	if err != nil {
		var apiErr *api.APIError
		if errors.As(err, &apiErr) {
			if apiErr.HTTPStatus == 401 || apiErr.HTTPStatus == 403 {
				return doctorCheck{
					Name:   "api",
					Status: doctorFail,
					Detail: "token rejected (invalid or expired)",
					Hint:   "run 'vector auth login' to re-authenticate",
				}
			}
			return doctorCheck{
				Name:   "api",
				Status: doctorFail,
				Detail: fmt.Sprintf("API error: %s", apiErr.Message),
				Hint:   "check the Vector status page and try again",
			}
		}
		return doctorCheck{
			Name:   "api",
			Status: doctorFail,
			Detail: fmt.Sprintf("network error: %s", err),
			Hint:   "check your network connection or VPN",
		}
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return doctorCheck{
			Name:   "api",
			Status: doctorFail,
			Detail: fmt.Sprintf("reading response: %s", err),
		}
	}

	var whoami whoamiResponse
	if err := json.Unmarshal(body, &whoami); err != nil {
		return doctorCheck{
			Name:   "api",
			Status: doctorWarn,
			Detail: "connected, but the response was not recognized",
		}
	}

	return doctorCheck{
		Name:   "api",
		Status: doctorPass,
		Detail: fmt.Sprintf("authenticated as %s (%s)", whoami.Data.User.Email, whoami.Data.Account.Name),
	}
}

// doctorStatusLabel renders a status for table output.
func doctorStatusLabel(status string) string {
	switch status {
	case doctorPass:
		return "OK"
	case doctorWarn:
		return "WARN"
	case doctorSkip:
		return "SKIP"
	case doctorFail:
		return "FAIL"
	default:
		return status
	}
}
