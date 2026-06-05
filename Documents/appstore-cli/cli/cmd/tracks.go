package cmd

import (
	"fmt"
	"strings"

	"github.com/dallaslabs/appctl/core/store"
	"github.com/spf13/cobra"
)

func newTracksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tracks <app>",
		Short: "List Google Play release tracks for an app",
		Long: `List all Google Play release tracks (production, beta, alpha, internal)
for the given app alias, including release status and version codes.

Only apps with a Google Play package configured will work with this command.
Use "appctl apps" to see which apps have Play packages.`,
		Example: `  appctl tracks venus
  appctl tracks cabinet-doors --output json
  appctl tracks venus --no-header`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var tracks []store.Track
			if serverAddr != "" {
				if err := fetchServerJSON("/api/v1/apps/"+args[0]+"/tracks", nil, &tracks); err != nil {
					return die(err)
				}
			} else {
				app, err := appByAlias(args[0])
				if err != nil {
					return die(err)
				}
				if app.PlayPackage == "" {
					return die(fmt.Errorf("app %q has no Google Play package", args[0]))
				}
				client, err := playClient()
				if err != nil {
					return die(err)
				}
				tracks, err = client.ListTracks(app.PlayPackage)
				if err != nil {
					return die(err)
				}
			}
			return render(tracks, func() {
				if !noHeader {
					fmt.Printf("%-14s %-16s %-24s %-24s\n", "TRACK", "STATUS", "VERSION CODES", "VERSION NAME")
				}
				for _, t := range tracks {
					fmt.Printf("%-14s %-16s %-24s %-24s\n", t.Name, trim(t.Status, 16), trim(fmt.Sprint(t.VersionCodes), 24), trim(strings.TrimSpace(t.VersionName), 24))
				}
			})
		},
	}
	return cmd
}
