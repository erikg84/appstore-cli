package cmd

import (
	"fmt"
	"strings"

	"github.com/dallaslabs/appctl/core/store"
	"github.com/spf13/cobra"
)

func newUsersCmd() *cobra.Command {
	parent := &cobra.Command{
		Use:   "users",
		Short: "List team members with access to your App Store and Play accounts",
		Long: `Commands for listing team members and their roles across App Store Connect
and Google Play Console.

App Store Connect users are fetched from the ASC Users API.
Google Play users are fetched from the Play Developer API (limited to accounts
accessible by the service account key).`,
	}
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all account users from both stores",
		Long: `List all team members who have access to your App Store Connect and/or
Google Play Console accounts, along with their roles and permissions.`,
		Example: `  appctl users list
  appctl users list --output json
  appctl users list --no-header`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var users []store.AppUser
			if serverAddr != "" {
				if err := fetchServerJSON("/api/v1/users", nil, &users); err != nil {
					return die(err)
				}
			} else {
				ascUsers, err := ascClient().ListUsers()
				if err != nil {
					return die(err)
				}
				users = append(users, ascUsers...)
				playUsers, err := listPlayUsers()
				if err != nil {
					return die(err)
				}
				users = append(users, playUsers...)
			}
			return render(users, func() {
				if !noHeader {
					fmt.Printf("%-28s %-32s %-30s %-10s\n", "ID", "EMAIL", "ROLES", "ALL APPS")
				}
				for _, user := range users {
					fmt.Printf("%-28s %-32s %-30s %-10t\n", trim(user.ID, 28), trim(user.Email, 32), trim(strings.Join(user.Roles, ","), 30), user.AllAppsVisible)
				}
			})
		},
	}
	parent.AddCommand(listCmd)
	return parent
}
