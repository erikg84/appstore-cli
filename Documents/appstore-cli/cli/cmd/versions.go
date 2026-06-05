package cmd

import (
	"fmt"

	"github.com/dallaslabs/appctl/core/store"
	"github.com/spf13/cobra"
)

func newVersionsCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "versions <app>",
		Short: "List App Store versions for an app",
		Long: `List all App Store versions (iOS, macOS, etc.) for the given app alias,
including the platform, version string, state (e.g. READY_FOR_SALE, IN_REVIEW),
and the date the version was created.`,
		Example: `  appctl versions venus
  appctl versions iss-spotter --output json
  appctl versions venus --limit 5`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var versions []store.Version
			if serverAddr != "" {
				if err := fetchServerJSON("/api/v1/apps/"+args[0]+"/versions", nil, &versions); err != nil {
					return die(err)
				}
			} else {
				app, err := appByAlias(args[0])
				if err != nil {
					return die(err)
				}
				versions, err = ascClient().ListVersions(app.ASCAppID)
				if err != nil {
					return die(err)
				}
			}
			if limit > 0 && limit < len(versions) {
				versions = versions[:limit]
			}
			return render(versions, func() {
				if !noHeader {
					fmt.Printf("%-10s %-16s %-24s %-24s\n", "PLATFORM", "VERSION", "STATE", "CREATED")
				}
				for _, v := range versions {
					fmt.Printf("%-10s %-16s %-24s %-24s\n", v.Platform, v.VersionString, trim(v.State, 24), trim(v.CreatedDate, 24))
				}
			})
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of versions to show (0 = all)")
	return cmd
}
