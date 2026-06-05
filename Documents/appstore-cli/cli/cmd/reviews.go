package cmd

import (
	"fmt"
	"net/url"

	"github.com/dallaslabs/appctl/core/store"
	"github.com/spf13/cobra"
)

func newReviewsCmd() *cobra.Command {
	var storeFlag string
	var limit int
	cmd := &cobra.Command{
		Use:   "reviews <app>",
		Short: "List reviews for an app from the App Store and/or Google Play",
		Long: `Fetch customer reviews for the given app alias.

Use --store to target one platform or pull from both at once. Results include
the star rating, reviewer name, title, body text, app version reviewed, and date.

App Store reviews are fetched from the App Store Connect API.
Play reviews are fetched from the Google Play Developer API (last 7 days by default).`,
		Example: `  appctl reviews venus
  appctl reviews venus --store ios
  appctl reviews venus --store android
  appctl reviews iss-spotter --store both --output json
  appctl reviews venus --limit 20`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			storeName, err := normalizeStore(storeFlag)
			if err != nil {
				return die(err)
			}

			var reviews []store.Review
			if serverAddr != "" {
				query := url.Values{}
				query.Set("store", storeName)
				if err := fetchServerJSON("/api/v1/apps/"+args[0]+"/reviews", query, &reviews); err != nil {
					return die(err)
				}
			} else {
				app, err := appByAlias(args[0])
				if err != nil {
					return die(err)
				}
				if wantsIOS(storeName) {
					iosReviews, err := ascClient().ListReviews(app.ASCAppID)
					if err != nil {
						return die(err)
					}
					reviews = append(reviews, iosReviews...)
				}
				if wantsAndroid(storeName) {
					if app.PlayPackage == "" {
						if storeName == "android" {
							return die(fmt.Errorf("app %q has no Google Play package", args[0]))
						}
					} else {
						client, err := playClient()
						if err != nil {
							return die(err)
						}
						androidReviews, err := client.ListReviews(app.PlayPackage)
						if err != nil {
							return die(err)
						}
						reviews = append(reviews, androidReviews...)
					}
				}
			}
			if limit > 0 && limit < len(reviews) {
				reviews = reviews[:limit]
			}
			return render(reviews, func() {
				if !noHeader {
					fmt.Printf("%-8s %-6s %-18s %-18s %-18s %-18s %s\n", "STORE", "RATING", "VERSION", "DATE", "REVIEWER", "TITLE", "BODY")
				}
				for _, r := range reviews {
					fmt.Printf("%-8s %-6d %-18s %-18s %-18s %-18s %s\n", r.Store, r.Rating, trim(r.AppVersion, 18), trim(r.Date, 18), trim(r.ReviewerName, 18), trim(r.Title, 18), trim(r.Body, 60))
				}
			})
		},
	}
	cmd.Flags().StringVar(&storeFlag, "store", "both", "Platform to query: ios, android, or both")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of reviews to show (0 = all)")
	return cmd
}
