package cmd

import (
	"fmt"

	"github.com/dallaslabs/appctl/core/store"
	"github.com/spf13/cobra"
)

func newBuildsCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "builds <app>",
		Short: "List TestFlight builds for an app",
		Long: `List all builds uploaded to TestFlight for the given app alias.
Shows the platform, build version, processing state, and upload date.

Only App Store Connect (iOS/macOS) builds are returned — for Android release
builds use "appctl tracks <app>" instead.`,
		Example: `  appctl builds venus
  appctl builds venus --output json
  appctl builds venus --limit 10`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var builds []store.Build
			if serverAddr != "" {
				if err := fetchServerJSON("/api/v1/apps/"+args[0]+"/builds", nil, &builds); err != nil {
					return die(err)
				}
			} else {
				app, err := appByAlias(args[0])
				if err != nil {
					return die(err)
				}
				builds, err = ascClient().ListBuilds(app.ASCAppID)
				if err != nil {
					return die(err)
				}
			}
			if limit > 0 && limit < len(builds) {
				builds = builds[:limit]
			}
			return render(builds, func() {
				if !noHeader {
					fmt.Printf("%-10s %-12s %-24s %-24s\n", "PLATFORM", "VERSION", "STATE", "UPLOADED")
				}
				for _, b := range builds {
					fmt.Printf("%-10s %-12s %-24s %-24s\n", b.Platform, b.Version, trim(b.State, 24), trim(b.UploadedDate, 24))
				}
			})
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of builds to show (0 = all)")
	return cmd
}
