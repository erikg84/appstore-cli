package cmd

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/dallaslabs/appctl/core/config"
	"github.com/dallaslabs/appctl/core/store"
	"github.com/spf13/cobra"
)

func newInstallsCmd() *cobra.Command {
	var storeFlag string
	var breakdown string

	cmd := &cobra.Command{
		Use:   "installs <app> [YYYYMM]",
		Short: "Show install and download stats for an app",
		Long: `Fetch install/download statistics from App Store Connect analytics and
Google Play install reports.

Use --store to target iOS, Android, or both. Android supports install report
breakdowns such as country, device, and OS version.`,
		Example: `  appctl installs venus
  appctl installs venus 202512 --store android
  appctl installs venus 202512 --store android --breakdown country
  appctl installs venus --output json`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appByAlias(args[0])
			if err != nil {
				return die(err)
			}
			yearMonth := ""
			if len(args) > 1 {
				yearMonth = args[1]
			}
			storeName, err := normalizeStore(storeFlag)
			if err != nil {
				return die(err)
			}
			if strings.TrimSpace(breakdown) == "" {
				breakdown = "overview"
			}

			var stats []store.InstallStat
			if serverAddr != "" {
				query := url.Values{}
				query.Set("store", storeName)
				query.Set("month", yearMonth)
				query.Set("breakdown", breakdown)
				if err := fetchServerJSON("/api/v1/apps/"+args[0]+"/installs", query, &stats); err != nil {
					return die(err)
				}
			} else {
				stats, err = loadInstallStats(app, args[0], storeName, yearMonth, breakdown)
				if err != nil {
					return die(err)
				}
			}

			sort.Slice(stats, func(i, j int) bool {
				if stats[i].Date == stats[j].Date {
					if stats[i].Source == stats[j].Source {
						if stats[i].Dimension == stats[j].Dimension {
							return stats[i].DimValue < stats[j].DimValue
						}
						return stats[i].Dimension < stats[j].Dimension
					}
					return stats[i].Source < stats[j].Source
				}
				return stats[i].Date < stats[j].Date
			})

			return render(stats, func() {
				if !noHeader {
					fmt.Printf("%-8s %-12s %-16s %-10s %-12s %-14s %-12s %s\n", "STORE", "DATE", "APP", "INSTALLS", "UNINSTALLS", "ACTIVE", "DIMENSION", "VALUE")
				}
				for _, stat := range stats {
					fmt.Printf("%-8s %-12s %-16s %-10d %-12d %-14d %-12s %s\n",
						stat.Source,
						trim(stat.Date, 12),
						trim(stat.App, 16),
						stat.Installs,
						stat.Uninstalls,
						stat.ActiveDevices,
						trim(stat.Dimension, 12),
						trim(stat.DimValue, 24),
					)
				}
			})
		},
	}

	cmd.Flags().StringVar(&storeFlag, "store", "both", "Platform to query: ios, android, or both")
	cmd.Flags().StringVar(&breakdown, "breakdown", "overview", "Android breakdown: overview, country, device, os_version, app_version, language, or carrier")
	return cmd
}

func loadInstallStats(app config.App, alias, storeName, yearMonth, breakdown string) ([]store.InstallStat, error) {
	var stats []store.InstallStat

	if wantsIOS(storeName) {
		iosStats, err := loadIOSInstallStats(app.ASCAppID, alias, yearMonth, breakdown)
		if err != nil {
			if storeName == "both" && app.PlayPackage != "" {
				fmt.Fprintf(os.Stderr, "warning: %v\n", err)
			} else {
				return nil, err
			}
		} else {
			stats = append(stats, iosStats...)
		}
	}
	if wantsAndroid(storeName) {
		if app.PlayPackage == "" {
			if storeName == "android" {
				return nil, fmt.Errorf("app %q has no Google Play package", alias)
			}
		} else {
			client, err := playClient()
			if err != nil {
				return nil, err
			}
			rows, err := client.DownloadInstallStats(app.PlayPackage, yearMonth, breakdown)
			if err != nil {
				return nil, err
			}
			stats = append(stats, store.ParseInstallStats(rows, "android", alias)...)
		}
	}
	return stats, nil
}

func loadIOSInstallStats(appID, appName, yearMonth, breakdown string) ([]store.InstallStat, error) {
	if strings.TrimSpace(appID) == "" {
		return nil, fmt.Errorf("app %q has no App Store Connect ID", appName)
	}
	analyticsClient := ascAnalyticsClient()
	downloadRows, err := analyticsClient.GetDownloadStats(appID)
	if err != nil {
		downloadRows, err = ascClient().GetDownloadStats(appID)
		if err != nil {
			return nil, fmt.Errorf("ios download stats unavailable: %w", err)
		}
	}
	downloadStats := store.ParseInstallStats(downloadRows, "ios", appName)

	deletionRows, err := analyticsClient.GetInstallDeletionStats(appID)
	if err != nil {
		deletionRows, err = ascClient().GetInstallDeletionStats(appID)
	}
	if err == nil {
		downloadStats = store.MergeInstallStats(downloadStats, store.ParseInstallStats(deletionRows, "ios", appName))
	}

	downloadStats = store.FilterInstallStatsMonth(downloadStats, yearMonth)
	if strings.TrimSpace(breakdown) != "" && breakdown != "overview" {
		filtered := make([]store.InstallStat, 0, len(downloadStats))
		for _, stat := range downloadStats {
			if stat.Dimension == breakdown {
				filtered = append(filtered, stat)
			}
		}
		downloadStats = filtered
	}
	return downloadStats, nil
}
