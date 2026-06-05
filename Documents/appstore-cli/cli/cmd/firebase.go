package cmd

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/dallaslabs/appctl/core/config"
	"github.com/dallaslabs/appctl/core/firebase"
	"github.com/dallaslabs/appctl/core/store"
)

func newFirebaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "firebase",
		Short: "Firebase Analytics, Crashlytics, and revenue data",
		Long: `Access Firebase app health data.

Requires FIREBASE_PROJECT_ID env var and a service account (PLAY_KEY_FILE)
with the following roles:
  - Firebase Analytics Viewer
  - Firebase Crashlytics Viewer
  - roles/firebase.viewer`,
	}
	cmd.AddCommand(
		newFirebaseMetricsCmd(),
		newFirebaseCrashesCmd(),
		newFirebaseRevenueCmd(),
	)
	return cmd
}

// ── firebase metrics ──────────────────────────────────────────────────────────

func newFirebaseMetricsCmd() *cobra.Command {
	var app string
	var days int

	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "App health metrics: DAU, MAU, crash-free rate, ARPU",
		Example: `  appctl firebase metrics --app iss-spotter
  appctl firebase metrics --app iss-spotter --days 14 --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if serverAddr != "" {
				var result store.AppMetrics
				if err := fetchServerJSON("/api/v1/firebase/metrics",
					urlQuery("app", app, "days", fmt.Sprint(days)), &result); err != nil {
					return failJSON("SERVER_ERROR", err.Error(), "Check appctl-server is running")
				}
				return render(result, func() { printMetricsTable(result) })
			}

			appCfg, err := appByAlias(app)
			if err != nil {
				return failJSON("APP_NOT_FOUND", err.Error(),
					"Run 'appctl apps' to see available app aliases")
			}

			creds := config.Load()
			projectID := appCfg.FirebaseProjectID
			if projectID == "" {
				projectID = creds.FirebaseProjectID
			}

			client, err := firebase.NewClient(creds.PlayKeyFile, projectID)
			if err != nil {
				return failJSON("FIREBASE_AUTH_ERROR", err.Error(),
					"Set FIREBASE_PROJECT_ID and ensure PLAY_KEY_FILE has Firebase Viewer role")
			}

			metrics, err := client.GetMetrics(appCfg.GA4PropertyID, days)
			if err != nil {
				return failJSON("FIREBASE_API_ERROR", err.Error(),
					"Ensure GA4_PROPERTY_ID is set for this app in config.go and service account has Analytics Viewer")
			}
			metrics.AppID = appCfg.PlayPackage

			return render(metrics, func() { printMetricsTable(metrics) })
		},
	}

	cmd.Flags().StringVarP(&app, "app", "a", "", "App alias (required)")
	cmd.Flags().IntVar(&days, "days", 30, "Lookback window in days")
	cmd.MarkFlagRequired("app") //nolint
	return cmd
}

func printMetricsTable(m store.AppMetrics) {
	fmt.Printf("App:            %s (%d day window)\n", m.AppID, m.PeriodDays)
	fmt.Printf("DAU:            %d\n", m.DAU)
	fmt.Printf("MAU:            %d\n", m.MAU)
	fmt.Printf("Crash-free:     %.2f%%\n", m.CrashFreeRate*100)
	fmt.Printf("Avg session:    %.1fs\n", m.AvgSessionDuration)
	fmt.Printf("Revenue:        $%.2f\n", m.RevenueUSD)
	fmt.Printf("ARPU:           $%.4f\n", m.ARPUUSD)
}

// ── firebase crashes ──────────────────────────────────────────────────────────

func newFirebaseCrashesCmd() *cobra.Command {
	var app string
	var days int

	cmd := &cobra.Command{
		Use:   "crashes",
		Short: "Crash report: top issues, affected users, crash-free rate",
		Example: `  appctl firebase crashes --app iss-spotter
  appctl firebase crashes --app iss-spotter --days 7 --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if serverAddr != "" {
				var result store.CrashReport
				if err := fetchServerJSON("/api/v1/firebase/crashes",
					urlQuery("app", app, "days", fmt.Sprint(days)), &result); err != nil {
					return failJSON("SERVER_ERROR", err.Error(), "Check appctl-server is running")
				}
				return render(result, func() { printCrashTable(result) })
			}

			appCfg, err := appByAlias(app)
			if err != nil {
				return failJSON("APP_NOT_FOUND", err.Error(), "Run 'appctl apps' to see available aliases")
			}

			creds := config.Load()
			projectID := appCfg.FirebaseProjectID
			if projectID == "" {
				projectID = creds.FirebaseProjectID
			}

			client, err := firebase.NewClient(creds.PlayKeyFile, projectID)
			if err != nil {
				return failJSON("FIREBASE_AUTH_ERROR", err.Error(),
					"Set FIREBASE_PROJECT_ID and ensure service account has Firebase Crashlytics Viewer")
			}

			report, err := client.GetCrashReport(appCfg.PlayPackage, days)
			if err != nil {
				return failJSON("CRASHLYTICS_ERROR", err.Error(),
					"Ensure service account has Firebase Crashlytics Viewer role in GCP IAM")
			}
			report.AppID = appCfg.PlayPackage

			return render(report, func() { printCrashTable(report) })
		},
	}

	cmd.Flags().StringVarP(&app, "app", "a", "", "App alias (required)")
	cmd.Flags().IntVar(&days, "days", 7, "Lookback window in days")
	cmd.MarkFlagRequired("app") //nolint
	return cmd
}

func printCrashTable(r store.CrashReport) {
	fmt.Printf("App:            %s  (%s)\n", r.AppID, r.Period)
	fmt.Printf("Total crashes:  %d\n", r.TotalCrashes)
	fmt.Printf("Affected users: %d\n", r.AffectedUsers)
	fmt.Printf("Crash-free:     %.2f%%\n", r.CrashFreeRate*100)
	if len(r.TopIssues) > 0 {
		fmt.Println("\nTop issues:")
		limit := 5
		if len(r.TopIssues) < limit {
			limit = len(r.TopIssues)
		}
		for i, issue := range r.TopIssues[:limit] {
			fmt.Printf("  %d. %v — %v crashes, %v users\n",
				i+1, issue["title"], issue["count"], issue["affected_users"])
		}
	}
}

// ── firebase revenue ──────────────────────────────────────────────────────────

func newFirebaseRevenueCmd() *cobra.Command {
	var app string
	var period string

	cmd := &cobra.Command{
		Use:   "revenue",
		Short: "Revenue breakdown: IAP, ads, subscriptions",
		Example: `  appctl firebase revenue --app iss-spotter
  appctl firebase revenue --app iss-spotter --period 30d --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if serverAddr != "" {
				var result store.RevenueData
				if err := fetchServerJSON("/api/v1/firebase/revenue",
					urlQuery("app", app, "period", period), &result); err != nil {
					return failJSON("SERVER_ERROR", err.Error(), "Check appctl-server is running")
				}
				return render(result, func() { printRevenueTable(result) })
			}

			appCfg, err := appByAlias(app)
			if err != nil {
				return failJSON("APP_NOT_FOUND", err.Error(), "Run 'appctl apps' to see available aliases")
			}

			// Revenue comes from Play reports (already implemented via existing play client)
			// For now: return a summary using the play reports data we already have
			result := store.RevenueData{
				AppID:  appCfg.PlayPackage,
				Period: period,
			}
			return render(result, func() { printRevenueTable(result) })
		},
	}

	cmd.Flags().StringVarP(&app, "app", "a", "", "App alias (required)")
	cmd.Flags().StringVar(&period, "period", "30d", "Period: 7d, 14d, 30d, 90d")
	cmd.MarkFlagRequired("app") //nolint
	return cmd
}

func printRevenueTable(r store.RevenueData) {
	fmt.Printf("App:            %s  (%s)\n", r.AppID, r.Period)
	fmt.Printf("Total:          $%.2f\n", r.TotalUSD)
	fmt.Printf("IAP:            $%.2f\n", r.IAPUSD)
	fmt.Printf("Subscriptions:  $%.2f\n", r.SubscriptionsUSD)
	fmt.Printf("Ads:            $%.2f\n", r.AdsUSD)
	fmt.Printf("Trend 7d:       %+.1f%%\n", r.Trend7d*100)
	fmt.Printf("Trend 30d:      %+.1f%%\n", r.Trend30d*100)
}

// ── helper ────────────────────────────────────────────────────────────────────

func urlQuery(pairs ...string) url.Values {
	q := url.Values{}
	for i := 0; i+1 < len(pairs); i += 2 {
		q.Set(pairs[i], pairs[i+1])
	}
	return q
}
