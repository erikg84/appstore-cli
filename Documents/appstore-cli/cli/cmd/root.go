package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	serverAddr   string
	outputFormat string
	noHeader     bool
)

var rootCmd = &cobra.Command{
	Use:   "appctl",
	Short: "Unified CLI for App Store Connect and Google Play Console",
	Long: `appctl lets you query and manage your iOS and Android apps from a single tool.

It connects directly to:
  • Apple App Store Connect REST API  (JWT/ES256 auth)
  • Google Play Developer API         (OAuth2 service account)
  • Google Play Reporting API         (dynamic app discovery)
  • Google Cloud Storage pubsite bucket (earnings, sales, stats CSVs)

Credentials are loaded from environment variables (see below) with sensible
defaults for this machine. You can also proxy all calls through a running
appctl-server with --server.

Environment variables:
  ASC_KEY_FILE             Path to primary .p8 key     (default: ~/Downloads/AuthKey_746FSCD2PK.p8)
  ASC_KEY_ID               Primary App Store key ID     (default: 746FSCD2PK)
  ASC_ISSUER_ID            App Store Connect issuer ID  (default: e68d90fb-c9b8-41dd-ad08-947afe7459ae)
  ASC_ANALYTICS_KEY_FILE   Optional analytics-only key  (default: ~/Downloads/AuthKey_79PWD5WB3S.p8)
  ASC_ANALYTICS_KEY_ID     Optional analytics key ID    (default: 79PWD5WB3S)
  ASC_ANALYTICS_ISSUER_ID  Optional analytics issuer    (default: ASC_ISSUER_ID)
  PLAY_KEY_FILE            Path to service account JSON (default: ~/Downloads/play-service-account.json)`,
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		switch outputFormat {
		case "table", "json":
			return nil
		default:
			return fmt.Errorf("invalid --output %q — expected table or json", outputFormat)
		}
	},
}

func Execute() error {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
	}
	return err
}

func init() {
	rootCmd.PersistentFlags().StringVar(&serverAddr, "server", "", "Proxy requests through a running appctl-server (e.g. http://localhost:8080)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table or json")
	rootCmd.PersistentFlags().BoolVar(&noHeader, "no-header", false, "Suppress column headers in table output")

	rootCmd.AddCommand(
		newAppsCmd(),
		newVersionsCmd(),
		newBuildsCmd(),
		newTracksCmd(),
		newReviewsCmd(),
		newInstallsCmd(),
		newIAPCmd(),
		newSubscriptionsCmd(),
		newTestFlightCmd(),
		newReportsCmd(),
		newUsersCmd(),
		newFirebaseCmd(),
		newDiagCmd(),
		newCheatsheetCmd(),
	)
}
