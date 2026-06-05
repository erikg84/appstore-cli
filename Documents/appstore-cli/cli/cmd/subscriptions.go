package cmd

import (
	"fmt"
	"net/url"

	"github.com/dallaslabs/appctl/core/store"
	"github.com/spf13/cobra"
)

func newSubscriptionsCmd() *cobra.Command {
	parent := &cobra.Command{
		Use:   "subscriptions",
		Short: "Manage subscriptions",
		Long:  `Commands for listing and inspecting subscription products across App Store Connect and Google Play.`,
	}

	var storeFlag string
	listCmd := &cobra.Command{
		Use:   "list <app>",
		Short: "List subscription products for an app",
		Long: `List all auto-renewable subscription products for the given app alias.

App Store: fetches subscription groups and their subscriptions via the ASC API.
Play Store: fetches subscriptions via the new Monetization.Subscriptions API.

Use --store to target one platform or both.`,
		Example: `  appctl subscriptions list venus
  appctl subscriptions list venus --store ios
  appctl subscriptions list venus --store android --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			storeName, err := normalizeStore(storeFlag)
			if err != nil {
				return die(err)
			}

			var items []store.Subscription
			if serverAddr != "" {
				query := url.Values{}
				query.Set("store", storeName)
				if err := fetchServerJSON("/api/v1/apps/"+args[0]+"/subscriptions", query, &items); err != nil {
					return die(err)
				}
			} else {
				app, err := appByAlias(args[0])
				if err != nil {
					return die(err)
				}
				if wantsIOS(storeName) {
					groups, err := ascClient().ListSubscriptionGroups(app.ASCAppID)
					if err != nil {
						return die(err)
					}
					ac := ascClient()
					for _, group := range groups {
						subs, err := ac.ListSubscriptions(group.ID)
						if err != nil {
							return die(err)
						}
						items = append(items, subs...)
					}
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
						androidItems, err := client.ListSubscriptions(app.PlayPackage)
						if err != nil {
							return die(err)
						}
						items = append(items, androidItems...)
					}
				}
			}
			return render(items, func() {
				if !noHeader {
					fmt.Printf("%-24s %-28s %-16s %s\n", "ID", "PRODUCT ID", "STATE", "NAME")
				}
				for _, item := range items {
					fmt.Printf("%-24s %-28s %-16s %s\n", trim(item.ID, 24), trim(item.ProductID, 28), trim(item.State, 16), trim(item.Name, 40))
				}
			})
		},
	}
	listCmd.Flags().StringVar(&storeFlag, "store", "both", "Platform to query: ios, android, or both")
	parent.AddCommand(listCmd)
	return parent
}
