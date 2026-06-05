package cmd

import (
	"fmt"
	"net/url"

	"github.com/dallaslabs/appctl/core/store"
	"github.com/spf13/cobra"
)

func newIAPCmd() *cobra.Command {
	parent := &cobra.Command{
		Use:   "iap",
		Short: "Manage in-app purchases",
		Long: `Commands for listing and inspecting in-app purchase products
across App Store Connect and Google Play.`,
	}

	var storeFlag string
	listCmd := &cobra.Command{
		Use:   "list <app>",
		Short: "List in-app purchase products for an app",
		Long: `List all one-time in-app purchase products configured for the given app alias.

App Store: fetches consumable and non-consumable IAPs via the ASC API.
Play Store: fetches one-time products via the new Monetization API.

Use --store to target one platform or both.`,
		Example: `  appctl iap list venus
  appctl iap list venus --store ios
  appctl iap list venus --store android
  appctl iap list cabinet-doors --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			storeName, err := normalizeStore(storeFlag)
			if err != nil {
				return die(err)
			}

			var items []store.IAP
			if serverAddr != "" {
				query := url.Values{}
				query.Set("store", storeName)
				if err := fetchServerJSON("/api/v1/apps/"+args[0]+"/iap", query, &items); err != nil {
					return die(err)
				}
			} else {
				app, err := appByAlias(args[0])
				if err != nil {
					return die(err)
				}
				if wantsIOS(storeName) {
					iosItems, err := ascClient().ListIAPs(app.ASCAppID)
					if err != nil {
						return die(err)
					}
					items = append(items, iosItems...)
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
						androidItems, err := client.ListIAPs(app.PlayPackage)
						if err != nil {
							return die(err)
						}
						items = append(items, androidItems...)
					}
				}
			}
			return render(items, func() {
				if !noHeader {
					fmt.Printf("%-24s %-28s %-18s %-16s %s\n", "ID", "PRODUCT ID", "TYPE", "STATE", "NAME")
				}
				for _, item := range items {
					fmt.Printf("%-24s %-28s %-18s %-16s %s\n", trim(item.ID, 24), trim(item.ProductID, 28), trim(item.ProductType, 18), trim(item.State, 16), trim(item.Name, 40))
				}
			})
		},
	}
	listCmd.Flags().StringVar(&storeFlag, "store", "both", "Platform to query: ios, android, or both")
	parent.AddCommand(listCmd)
	return parent
}
