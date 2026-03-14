package cli

import (
	"fmt"
	"os"

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
		Short: "Vector CLI — manage your Vector.dev hosting",
		Long:  "Vector CLI — manage your Vector.dev hosting\n\nA command-line tool for managing sites, deployments, and configurations on Vector.dev.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// 1. Load config (defaults if missing)
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}

			// 2. Load credentials (empty if missing)
			creds, err := config.LoadCredentials()
			if err != nil {
				return err
			}

			// 3. Resolve token: --token flag > VECTOR_API_KEY env > stored credentials
			token, _ := cmd.Flags().GetString("token")
			if token == "" {
				token = os.Getenv("VECTOR_API_KEY")
			}
			if token == "" {
				token = creds.ApiKey
			}

			// 4. Build API client
			client := api.NewClient(cfg.ApiURL, token, "")

			// 5. Detect output format from --json/--no-json flags
			jsonFlag, _ := cmd.Flags().GetBool("json")
			noJsonFlag, _ := cmd.Flags().GetBool("no-json")
			format := output.DetectFormat(jsonFlag, noJsonFlag)

			// 6. Create App and store in context
			app := appctx.NewApp(cfg, creds, client, format)
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
	cmd.PersistentFlags().String("token", "", "API token (overrides VECTOR_API_KEY and stored credentials)")
	cmd.PersistentFlags().Bool("json", false, "Force JSON output")
	cmd.PersistentFlags().Bool("no-json", false, "Force table output")

	cmd.AddCommand(commands.NewAuthCmd())

	return cmd
}
