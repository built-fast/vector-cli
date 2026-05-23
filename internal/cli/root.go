package cli

import (
	"fmt"
	"os"

	"github.com/itchyny/gojq"
	"github.com/spf13/cobra"

	"github.com/built-fast/vector-cli/internal/api"
	"github.com/built-fast/vector-cli/internal/appctx"
	"github.com/built-fast/vector-cli/internal/commands"
	"github.com/built-fast/vector-cli/internal/config"
	"github.com/built-fast/vector-cli/internal/output"
	"github.com/built-fast/vector-cli/internal/version"
)

// NewRootCmd creates and returns the root cobra command.
func NewRootCmd() *cobra.Command {
	var showVersion bool

	cmd := &cobra.Command{
		Use:   "vector",
		Short: "Vector CLI — manage your Vector hosting",
		Long:  "Vector CLI — manage your Vector hosting\n\nA command-line tool for managing sites, deployments, and configurations via the Vector Pro API by BuiltFast (builtfast.com).",
		Example: `  # Force JSON output for any command
  vector --json site list

  # Use a one-off token without logging in
  vector --token mytoken123 site list

  # Filter JSON output with built-in jq
  vector site list --jq '.[].id'`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// 1. Load config (defaults if missing)
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}

			// 2. Resolve token: --token flag > VECTOR_API_KEY env > keyring
			var token, tokenSource string
			token, _ = cmd.Flags().GetString("token")
			if token != "" {
				tokenSource = "flag"
			}
			if token == "" {
				token = os.Getenv("VECTOR_API_KEY")
				if token != "" {
					tokenSource = "env"
				}
			}
			if token == "" {
				if t, err := config.Load(); err == nil && t != "" {
					token = t
					tokenSource = "keyring"
				}
			}

			// 3. Build API client
			client := api.NewClient(cfg.APIURL, token, "")

			// 4. Detect output format from --json/--no-json flags
			jsonFlag, _ := cmd.Flags().GetBool("json")
			noJSONFlag, _ := cmd.Flags().GetBool("no-json")
			format := output.DetectFormat(jsonFlag, noJSONFlag)

			// 5. Handle --jq flag
			jqExpr, _ := cmd.Flags().GetString("jq")
			var writerOpts []output.WriterOption

			if jqExpr != "" {
				if noJSONFlag {
					return fmt.Errorf("--jq and --no-json cannot be used together")
				}

				query, err := gojq.Parse(jqExpr)
				if err != nil {
					return fmt.Errorf("invalid jq expression: %w", err)
				}

				code, err := gojq.Compile(query)
				if err != nil {
					return fmt.Errorf("failed to compile jq expression: %w", err)
				}

				// jq implies JSON output
				format = output.JSON
				writerOpts = append(writerOpts, output.WithJQ(jqExpr, code))
			}

			// 6. Create App and store in context
			app := appctx.NewApp(cfg, client, tokenSource)
			app.Output = output.NewWriter(os.Stdout, format, writerOpts...)
			cmd.SetContext(appctx.WithApp(cmd.Context(), app))

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), version.FullVersion())
				return nil
			}
			return cmd.Help()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().BoolVar(&showVersion, "version", false, "Print version information and exit")
	cmd.PersistentFlags().String("token", "", "API token (overrides VECTOR_API_KEY and keyring)")
	cmd.PersistentFlags().Bool("json", false, "Force JSON output")
	cmd.PersistentFlags().Bool("no-json", false, "Force table output")
	cmd.PersistentFlags().String("jq", "", `Filter JSON output with a jq expression (built-in, no external jq required)`)

	cmd.AddCommand(commands.NewAuthCmd())
	cmd.AddCommand(commands.NewSiteCmd())
	cmd.AddCommand(commands.NewEnvCmd())
	cmd.AddCommand(commands.NewDeployCmd())
	cmd.AddCommand(commands.NewSSLCmd())
	cmd.AddCommand(commands.NewPHPVersionsCmd())
	cmd.AddCommand(commands.NewEventCmd())
	cmd.AddCommand(commands.NewAccountCmd())
	cmd.AddCommand(commands.NewWebhookCmd())
	cmd.AddCommand(commands.NewBackupCmd())
	cmd.AddCommand(commands.NewRestoreCmd())
	cmd.AddCommand(commands.NewWafCmd())
	cmd.AddCommand(commands.NewDBCmd())
	cmd.AddCommand(commands.NewArchiveCmd())
	cmd.AddCommand(commands.NewMcpCmd())
	cmd.AddCommand(commands.NewSkillCmd())

	return cmd
}
