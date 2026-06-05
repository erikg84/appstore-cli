package cmd

import (
	"fmt"

	"github.com/dallaslabs/appctl/core/store"
	"github.com/spf13/cobra"
)

func newTestFlightCmd() *cobra.Command {
	parent := &cobra.Command{
		Use:   "testflight",
		Short: "Inspect TestFlight beta groups and testers",
		Long:  `Commands for listing TestFlight beta groups and testers for your App Store apps.`,
	}
	parent.AddCommand(newTestFlightGroupsCmd(), newTestFlightTestersCmd())
	return parent
}

func newTestFlightGroupsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "groups <app>",
		Short: "List TestFlight beta groups for an app",
		Long: `List all TestFlight beta groups configured for the given app alias,
including whether each group is internal (team members only) or external,
and the current tester count.`,
		Example: `  appctl testflight groups venus
  appctl testflight groups venus --output json
  appctl testflight groups venus --no-header`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var groups []store.BetaGroup
			if serverAddr != "" {
				if err := fetchServerJSON("/api/v1/apps/"+args[0]+"/testflight/groups", nil, &groups); err != nil {
					return die(err)
				}
			} else {
				app, err := appByAlias(args[0])
				if err != nil {
					return die(err)
				}
				groups, err = ascClient().ListBetaGroups(app.ASCAppID)
				if err != nil {
					return die(err)
				}
			}
			return render(groups, func() {
				if !noHeader {
					fmt.Printf("%-36s %-28s %-10s %-12s\n", "ID", "NAME", "INTERNAL", "TESTERS")
				}
				for _, g := range groups {
					fmt.Printf("%-36s %-28s %-10t %-12d\n", trim(g.ID, 36), trim(g.Name, 28), g.IsInternal, g.TesterCount)
				}
			})
		},
	}
}

func newTestFlightTestersCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "testers <app>",
		Short: "List TestFlight testers for an app",
		Long: `List all external TestFlight testers for the given app alias,
including their email, name, and current invitation state.`,
		Example: `  appctl testflight testers venus
  appctl testflight testers venus --output json
  appctl testflight testers venus --limit 50`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var testers []store.BetaTester
			if serverAddr != "" {
				if err := fetchServerJSON("/api/v1/apps/"+args[0]+"/testflight/testers", nil, &testers); err != nil {
					return die(err)
				}
			} else {
				app, err := appByAlias(args[0])
				if err != nil {
					return die(err)
				}
				testers, err = ascClient().ListBetaTesters(app.ASCAppID)
				if err != nil {
					return die(err)
				}
			}
			if limit > 0 && limit < len(testers) {
				testers = testers[:limit]
			}
			return render(testers, func() {
				if !noHeader {
					fmt.Printf("%-36s %-30s %-18s %-18s %-12s\n", "ID", "EMAIL", "FIRST", "LAST", "STATE")
				}
				for _, t := range testers {
					fmt.Printf("%-36s %-30s %-18s %-18s %-12s\n", trim(t.ID, 36), trim(t.Email, 30), trim(t.FirstName, 18), trim(t.LastName, 18), trim(t.State, 12))
				}
			})
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of testers to show (0 = all)")
	return cmd
}
