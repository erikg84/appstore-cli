package cmd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/dallaslabs/appctl/core/config"
	"github.com/dallaslabs/appctl/core/play"
	"github.com/dallaslabs/appctl/core/store"
	"github.com/spf13/cobra"
)

func newReportsCmd() *cobra.Command {
	parent := &cobra.Command{
		Use:   "reports",
		Short: "Fetch financial and sales reports from both stores",
		Long: `Download and display financial reports from App Store Connect and Google Play.

App Store reports come from the ASC Reporting API (sales, proceeds, downloads).
Google Play reports are fetched directly from the GCS pubsite bucket where Google
deposits earnings, sales, installs, crash stats, reviews, and acquisition CSVs.`,
	}
	parent.AddCommand(newSalesReportCmd())
	parent.AddCommand(newPlayReportsCmd())
	return parent
}

func newSalesReportCmd() *cobra.Command {
	var vendor string
	var date string
	var frequency string
	cmd := &cobra.Command{
		Use:   "sales",
		Short: "Fetch App Store Connect sales / proceeds report",
		Long: `Download a sales or proceeds report from App Store Connect.

Reports are returned as tabular data showing units sold, revenue, product ID,
country code, and currency. Use --frequency to select daily, weekly, or monthly
aggregation.

The vendor number defaults to 92547182. Override with --vendor or ASC_VENDOR_ID.`,
		Example: `  appctl reports sales --date 2025-12
  appctl reports sales --date 2025-12 --frequency MONTHLY
  appctl reports sales --date 2025-12 --frequency DAILY --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if vendor == "" {
				vendor = config.Load().ASCVendorID
			}
			if vendor == "" || date == "" {
				return die(fmt.Errorf("--date is required (--vendor defaults to ASC_VENDOR_ID)"))
			}
			if frequency == "" {
				frequency = "MONTHLY"
			}

			var reports []store.SalesReport
			if serverAddr != "" {
				query := url.Values{}
				query.Set("vendor", vendor)
				query.Set("date", date)
				query.Set("frequency", strings.ToUpper(frequency))
				if err := fetchServerJSON("/api/v1/reports/sales", query, &reports); err != nil {
					return die(err)
				}
			} else {
				var err error
				reports, err = ascClient().GetSalesReport(vendor, date, strings.ToUpper(frequency))
				if err != nil {
					return die(err)
				}
			}
			return render(reports, func() {
				if !noHeader {
					fmt.Printf("%-12s %-8s %-12s %-20s %-10s %-10s\n", "DATE", "UNITS", "REVENUE", "PRODUCT", "COUNTRY", "CURRENCY")
				}
				for _, report := range reports {
					fmt.Printf("%-12s %-8d %-12.2f %-20s %-10s %-10s\n", report.Date, report.Units, report.Revenue, trim(report.ProductID, 20), report.CountryCode, report.Currency)
				}
			})
		},
	}
	cmd.Flags().StringVar(&vendor, "vendor", "", "App Store Connect vendor number (required)")
	cmd.Flags().StringVar(&date, "date", "", "Report date: YYYY-MM for monthly, YYYY-MM-DD for daily (required)")
	cmd.Flags().StringVar(&frequency, "frequency", "MONTHLY", "Aggregation period: MONTHLY, WEEKLY, or DAILY")
	return cmd
}

func newPlayReportsCmd() *cobra.Command {
	parent := &cobra.Command{
		Use:   "play",
		Short: "Google Play GCS financial report commands",
		Long: `Download financial and statistics reports from Google Play's GCS pubsite bucket.

Google deposits CSV/ZIP report files into a private GCS bucket at the end of each
month. Reports are organized into categories: earnings, sales, reviews,
stats/crashes, stats/installs, and acquisition.`,
	}

	filesCmd := &cobra.Command{
		Use:   "files",
		Short: "List available Play report files in GCS",
		Long: `List all report files available in the Google Play GCS pubsite bucket.

Use --category to filter to a specific report type. Available categories:
  earnings        — monthly earnings/proceeds
  sales           — unit sales and revenue
  reviews         — user reviews export
  stats/crashes   — crash rate and ANR stats
  stats/installs  — install/uninstall counts
  acquisition     — store listing funnel data`,
		Example: `  appctl reports play files
  appctl reports play files --category earnings
  appctl reports play files --category stats/crashes --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			category, _ := cmd.Flags().GetString("category")
			client, err := playReportClient()
			if err != nil {
				return die(err)
			}
			files, err := client.ListReportFiles(category)
			if err != nil {
				return die(err)
			}
			return render(files, func() {
				if !noHeader {
					fmt.Printf("%-25s %-40s %s\n", "CATEGORY", "NAME", "SIZE")
				}
				for _, f := range files {
					fmt.Printf("%-25s %-40s %d\n", f.Category, f.Name, f.Size)
				}
			})
		},
	}
	filesCmd.Flags().String("category", "", "Filter by category prefix (earnings, sales, reviews, stats/crashes, stats/installs, acquisition)")

	earningsCmd := &cobra.Command{
		Use:   "earnings [YYYYMM]",
		Short: "Download Play earnings report from GCS",
		Long: `Download and display the Google Play earnings report for a given month.

The report is fetched as a ZIP+CSV from the GCS pubsite bucket. When no month
argument is provided, the most recent available month is used.

Output is tab-separated CSV rows (header row first).`,
		Example: `  appctl reports play earnings
  appctl reports play earnings 202406
  appctl reports play earnings 202406 --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			month := ""
			if len(args) > 0 {
				month = args[0]
			}
			client, err := playReportClient()
			if err != nil {
				return die(err)
			}
			rows, err := client.DownloadEarnings(month)
			if err != nil {
				return die(err)
			}
			return render(rows, func() {
				for _, row := range rows {
					fmt.Println(strings.Join(row, "\t"))
				}
			})
		},
	}

	salesCmd := &cobra.Command{
		Use:   "sales [YYYYMM]",
		Short: "Download Play sales report from GCS",
		Long: `Download and display the Google Play sales report for a given month.

The report is fetched as a ZIP+CSV from the GCS pubsite bucket. When no month
argument is provided, the most recent available month is used.

Output is tab-separated CSV rows (header row first).`,
		Example: `  appctl reports play sales
  appctl reports play sales 202406
  appctl reports play sales 202406 --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			month := ""
			if len(args) > 0 {
				month = args[0]
			}
			client, err := playReportClient()
			if err != nil {
				return die(err)
			}
			rows, err := client.DownloadSales(month)
			if err != nil {
				return die(err)
			}
			return render(rows, func() {
				for _, row := range rows {
					fmt.Println(strings.Join(row, "\t"))
				}
			})
		},
	}

	var installsBreakdown string
	installsCmd := &cobra.Command{
		Use:   "installs <app> [YYYYMM]",
		Short: "Download Play install statistics from GCS",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appByAlias(args[0])
			if err != nil {
				return die(err)
			}
			if app.PlayPackage == "" {
				return die(fmt.Errorf("app %q has no Google Play package", args[0]))
			}
			month := ""
			if len(args) > 1 {
				month = args[1]
			}
			var stats []store.InstallStat
			if serverAddr != "" {
				query := url.Values{}
				query.Set("package", app.PlayPackage)
				query.Set("month", month)
				query.Set("breakdown", installsBreakdown)
				if err := fetchServerJSON("/api/v1/reports/play/installs", query, &stats); err != nil {
					return die(err)
				}
			} else {
				client, err := playReportClient()
				if err != nil {
					return die(err)
				}
				rows, err := client.DownloadInstallStats(app.PlayPackage, month, installsBreakdown)
				if err != nil {
					return die(err)
				}
				stats = store.ParseInstallStats(rows, "android", args[0])
			}
			return render(stats, func() {
				if !noHeader {
					fmt.Printf("%-12s %-16s %-10s %-12s %-14s %-12s %s\n", "DATE", "APP", "INSTALLS", "UNINSTALLS", "ACTIVE", "DIMENSION", "VALUE")
				}
				for _, stat := range stats {
					fmt.Printf("%-12s %-16s %-10d %-12d %-14d %-12s %s\n", stat.Date, trim(stat.App, 16), stat.Installs, stat.Uninstalls, stat.ActiveDevices, trim(stat.Dimension, 12), trim(stat.DimValue, 24))
				}
			})
		},
	}
	installsCmd.Flags().StringVar(&installsBreakdown, "breakdown", "overview", "Install breakdown: overview, country, device, os_version, app_version, language, or carrier")

	var crashesBreakdown string
	crashesCmd := &cobra.Command{
		Use:   "crashes <app> [YYYYMM]",
		Short: "Download Play crash statistics from GCS",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appByAlias(args[0])
			if err != nil {
				return die(err)
			}
			if app.PlayPackage == "" {
				return die(fmt.Errorf("app %q has no Google Play package", args[0]))
			}
			month := ""
			if len(args) > 1 {
				month = args[1]
			}
			var stats []store.CrashStat
			if serverAddr != "" {
				query := url.Values{}
				query.Set("package", app.PlayPackage)
				query.Set("month", month)
				query.Set("breakdown", crashesBreakdown)
				if err := fetchServerJSON("/api/v1/reports/play/crashes", query, &stats); err != nil {
					return die(err)
				}
			} else {
				client, err := playReportClient()
				if err != nil {
					return die(err)
				}
				rows, err := client.DownloadCrashStats(app.PlayPackage, month, crashesBreakdown)
				if err != nil {
					return die(err)
				}
				stats = store.ParseCrashStats(rows, "android", args[0])
			}
			return render(stats, func() {
				if !noHeader {
					fmt.Printf("%-12s %-16s %-10s %-10s %-12s %s\n", "DATE", "APP", "CRASHES", "RATE", "DIMENSION", "VALUE")
				}
				for _, stat := range stats {
					fmt.Printf("%-12s %-16s %-10d %-10.4f %-12s %s\n", stat.Date, trim(stat.App, 16), stat.Crashes, stat.CrashRate, trim(stat.Dimension, 12), trim(stat.DimValue, 24))
				}
			})
		},
	}
	crashesCmd.Flags().StringVar(&crashesBreakdown, "breakdown", "overview", "Crash breakdown: overview, app_version, device, or os_version")

	var acquisitionType string
	acquisitionCmd := &cobra.Command{
		Use:   "acquisition <app> [YYYYMM]",
		Short: "Download Play acquisition statistics from GCS",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appByAlias(args[0])
			if err != nil {
				return die(err)
			}
			if app.PlayPackage == "" {
				return die(fmt.Errorf("app %q has no Google Play package", args[0]))
			}
			month := ""
			if len(args) > 1 {
				month = args[1]
			}
			var rows [][]string
			if serverAddr != "" {
				query := url.Values{}
				query.Set("package", app.PlayPackage)
				query.Set("month", month)
				query.Set("type", acquisitionType)
				if err := fetchServerJSON("/api/v1/reports/play/acquisition", query, &rows); err != nil {
					return die(err)
				}
			} else {
				client, err := playReportClient()
				if err != nil {
					return die(err)
				}
				rows, err = client.DownloadAcquisitionStats(app.PlayPackage, month, acquisitionType)
				if err != nil {
					return die(err)
				}
			}
			return render(rows, func() {
				for _, row := range rows {
					fmt.Println(strings.Join(row, "\t"))
				}
			})
		},
	}
	acquisitionCmd.Flags().StringVar(&acquisitionType, "type", "buyers_7d", "Acquisition type: buyers_7d or retained_installers")

	parent.AddCommand(filesCmd, earningsCmd, salesCmd, installsCmd, crashesCmd, acquisitionCmd)
	return parent
}

func playReportClient() (*play.Client, error) {
	creds := config.Load()
	return play.NewClient(creds.PlayKeyFile)
}
