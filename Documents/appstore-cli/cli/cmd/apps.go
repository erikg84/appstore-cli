package cmd

import (
	"fmt"

	"github.com/dallaslabs/appctl/core/store"
	"github.com/spf13/cobra"
)

func newAppsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "apps",
		Short: "List all configured apps",
		Long: `List every app registered in appctl's config, showing the app alias,
display name, App Store Connect app ID, and Google Play package name.

The alias is used as the <app> argument in all other commands.`,
		Example: `  appctl apps
  appctl apps --output json
  appctl apps --no-header`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var apps []store.AppSummary
			if serverAddr != "" {
				if err := fetchServerJSON("/api/v1/apps", nil, &apps); err != nil {
					return die(err)
				}
			} else {
				apps = appSummaries()
			}
			return render(apps, func() {
				if !noHeader {
					fmt.Printf("%-16s %-28s %-12s %-48s\n", "ALIAS", "NAME", "ASC ID", "PLAY PACKAGE")
				}
				for _, app := range apps {
					fmt.Printf("%-16s %-28s %-12s %-48s\n", app.Alias, trim(app.Name, 28), app.ASCID, trim(app.PlayPackage, 48))
				}
			})
		},
	}
}
