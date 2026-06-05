package rest

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/dallaslabs/appctl/core/asc"
	"github.com/dallaslabs/appctl/core/config"
	"github.com/dallaslabs/appctl/core/play"
	"github.com/dallaslabs/appctl/core/store"
)

type Handler struct{}

func ascClient() *asc.Client {
	creds := config.Load()
	return asc.NewClient(creds.ASCKeyID, creds.ASCIssuerID, creds.ASCKeyFile)
}

func ascAnalyticsClient() *asc.Client {
	creds := config.Load()
	return asc.NewClient(creds.ASCAnalyticsKeyID, creds.ASCAnalyticsIssuerID, creds.ASCAnalyticsKeyFile)
}

func playClient() (*play.Client, error) {
	creds := config.Load()
	return play.NewClient(creds.PlayKeyFile)
}

func (h Handler) Apps(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, appSummaries())
}

func (h Handler) Versions(w http.ResponseWriter, r *http.Request) {
	app, err := appByAlias(chi.URLParam(r, "alias"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	versions, err := ascClient().ListVersions(app.ASCAppID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, versions)
}

func (h Handler) Builds(w http.ResponseWriter, r *http.Request) {
	app, err := appByAlias(chi.URLParam(r, "alias"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	builds, err := ascClient().ListBuilds(app.ASCAppID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, builds)
}

func (h Handler) Tracks(w http.ResponseWriter, r *http.Request) {
	app, err := appByAlias(chi.URLParam(r, "alias"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if app.PlayPackage == "" {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("app %q has no Google Play package", chi.URLParam(r, "alias")))
		return
	}
	client, err := playClient()
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	tracks, err := client.ListTracks(app.PlayPackage)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tracks)
}

func (h Handler) Reviews(w http.ResponseWriter, r *http.Request) {
	app, err := appByAlias(chi.URLParam(r, "alias"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	storeName, err := normalizeStore(r.URL.Query().Get("store"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var reviews []store.Review
	both := storeName == "both"
	if wantsIOS(storeName) {
		iosReviews, err := ascClient().ListReviews(app.ASCAppID)
		if err != nil {
			if !both {
				writeError(w, http.StatusBadGateway, err.Error())
				return
			}
			log.Printf("reviews: ios fetch for %s: %v", chi.URLParam(r, "alias"), err)
		} else {
			reviews = append(reviews, iosReviews...)
		}
	}
	if wantsAndroid(storeName) {
		if app.PlayPackage == "" {
			if !both {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("app %q has no Google Play package", chi.URLParam(r, "alias")))
				return
			}
		} else {
			client, err := playClient()
			if err == nil {
				androidReviews, err := client.ListReviews(app.PlayPackage)
				if err != nil {
					if !both {
						writeError(w, http.StatusBadGateway, err.Error())
						return
					}
					log.Printf("reviews: android fetch for %s: %v", chi.URLParam(r, "alias"), err)
				} else {
					reviews = append(reviews, androidReviews...)
				}
			} else if !both {
				writeError(w, http.StatusBadGateway, err.Error())
				return
			}
		}
	}
	writeJSON(w, http.StatusOK, reviews)
}

func (h Handler) Installs(w http.ResponseWriter, r *http.Request) {
	alias := chi.URLParam(r, "alias")
	app, err := appByAlias(alias)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	storeName, err := normalizeStore(r.URL.Query().Get("store"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	month := strings.TrimSpace(r.URL.Query().Get("month"))
	breakdown := strings.TrimSpace(r.URL.Query().Get("breakdown"))
	if breakdown == "" {
		breakdown = "overview"
	}

	var stats []store.InstallStat
	if wantsIOS(storeName) {
		iosStats, err := loadAppInstallStats(app.ASCAppID, alias, month, breakdown)
		if err != nil {
			if !(storeName == "both" && app.PlayPackage != "") {
				writeError(w, http.StatusBadGateway, err.Error())
				return
			}
		} else {
			stats = append(stats, iosStats...)
		}
	}
	if wantsAndroid(storeName) {
		if app.PlayPackage == "" {
			if storeName == "android" {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("app %q has no Google Play package", alias))
				return
			}
		} else {
			client, err := playClient()
			if err == nil {
				rows, err := client.DownloadInstallStats(app.PlayPackage, month, breakdown)
				if err != nil {
					if storeName == "android" {
						writeError(w, http.StatusBadGateway, err.Error())
						return
					}
					log.Printf("installs: android fetch for %s: %v", alias, err)
				} else {
					stats = append(stats, store.ParseInstallStats(rows, "android", alias)...)
				}
			} else if storeName == "android" {
				writeError(w, http.StatusBadGateway, err.Error())
				return
			}
		}
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h Handler) IAP(w http.ResponseWriter, r *http.Request) {
	app, err := appByAlias(chi.URLParam(r, "alias"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	storeName, err := normalizeStore(r.URL.Query().Get("store"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var items []store.IAP
	if wantsIOS(storeName) {
		iosItems, err := ascClient().ListIAPs(app.ASCAppID)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		items = append(items, iosItems...)
	}
	if wantsAndroid(storeName) {
		if app.PlayPackage == "" {
			if storeName == "android" {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("app %q has no Google Play package", chi.URLParam(r, "alias")))
				return
			}
		} else {
			client, err := playClient()
			if err != nil {
				writeError(w, http.StatusBadGateway, err.Error())
				return
			}
			androidItems, err := client.ListIAPs(app.PlayPackage)
			if err != nil {
				writeError(w, http.StatusBadGateway, err.Error())
				return
			}
			items = append(items, androidItems...)
		}
	}
	writeJSON(w, http.StatusOK, items)
}

func (h Handler) Subscriptions(w http.ResponseWriter, r *http.Request) {
	app, err := appByAlias(chi.URLParam(r, "alias"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	storeName, err := normalizeStore(r.URL.Query().Get("store"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var items []store.Subscription
	if wantsIOS(storeName) {
		ascSvc := ascClient()
		groups, err := ascSvc.ListSubscriptionGroups(app.ASCAppID)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		for _, group := range groups {
			subs, err := ascSvc.ListSubscriptions(group.ID)
			if err != nil {
				writeError(w, http.StatusBadGateway, err.Error())
				return
			}
			items = append(items, subs...)
		}
	}
	if wantsAndroid(storeName) {
		if app.PlayPackage == "" {
			if storeName == "android" {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("app %q has no Google Play package", chi.URLParam(r, "alias")))
				return
			}
		} else {
			client, err := playClient()
			if err != nil {
				writeError(w, http.StatusBadGateway, err.Error())
				return
			}
			subs, err := client.ListSubscriptions(app.PlayPackage)
			if err != nil {
				writeError(w, http.StatusBadGateway, err.Error())
				return
			}
			items = append(items, subs...)
		}
	}
	writeJSON(w, http.StatusOK, items)
}

func (h Handler) TestFlightGroups(w http.ResponseWriter, r *http.Request) {
	app, err := appByAlias(chi.URLParam(r, "alias"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	groups, err := ascClient().ListBetaGroups(app.ASCAppID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, groups)
}

func (h Handler) TestFlightTesters(w http.ResponseWriter, r *http.Request) {
	app, err := appByAlias(chi.URLParam(r, "alias"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	testers, err := ascClient().ListBetaTesters(app.ASCAppID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, testers)
}

func (h Handler) SalesReport(w http.ResponseWriter, r *http.Request) {
	vendor := strings.TrimSpace(r.URL.Query().Get("vendor"))
	date := strings.TrimSpace(r.URL.Query().Get("date"))
	frequency := strings.TrimSpace(r.URL.Query().Get("frequency"))
	if vendor == "" || date == "" {
		writeError(w, http.StatusBadRequest, "vendor and date are required")
		return
	}
	reports, err := ascClient().GetSalesReport(vendor, date, frequency)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, reports)
}

func (h Handler) PlayReportFiles(w http.ResponseWriter, r *http.Request) {
	client, err := playClient()
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	category := r.URL.Query().Get("category") // optional filter: earnings, sales, reviews, stats/crashes, etc.
	files, err := client.ListReportFiles(category)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, files)
}

func (h Handler) PlayEarnings(w http.ResponseWriter, r *http.Request) {
	client, err := playClient()
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	yearMonth := r.URL.Query().Get("month") // e.g. "202512", optional
	rows, err := client.DownloadEarnings(yearMonth)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (h Handler) PlaySales(w http.ResponseWriter, r *http.Request) {
	client, err := playClient()
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	yearMonth := r.URL.Query().Get("month")
	rows, err := client.DownloadSales(yearMonth)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (h Handler) PlayInstalls(w http.ResponseWriter, r *http.Request) {
	pkg := strings.TrimSpace(r.URL.Query().Get("package"))
	if pkg == "" {
		writeError(w, http.StatusBadRequest, "package is required")
		return
	}
	client, err := playClient()
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	rows, err := client.DownloadInstallStats(pkg, strings.TrimSpace(r.URL.Query().Get("month")), strings.TrimSpace(r.URL.Query().Get("breakdown")))
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, store.ParseInstallStats(rows, "android", pkg))
}

func (h Handler) PlayCrashes(w http.ResponseWriter, r *http.Request) {
	pkg := strings.TrimSpace(r.URL.Query().Get("package"))
	if pkg == "" {
		writeError(w, http.StatusBadRequest, "package is required")
		return
	}
	client, err := playClient()
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	rows, err := client.DownloadCrashStats(pkg, strings.TrimSpace(r.URL.Query().Get("month")), strings.TrimSpace(r.URL.Query().Get("breakdown")))
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, store.ParseCrashStats(rows, "android", pkg))
}

func (h Handler) PlayAcquisition(w http.ResponseWriter, r *http.Request) {
	pkg := strings.TrimSpace(r.URL.Query().Get("package"))
	if pkg == "" {
		writeError(w, http.StatusBadRequest, "package is required")
		return
	}
	client, err := playClient()
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	rows, err := client.DownloadAcquisitionStats(pkg, strings.TrimSpace(r.URL.Query().Get("month")), strings.TrimSpace(r.URL.Query().Get("type")))
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (h Handler) Users(w http.ResponseWriter, r *http.Request) {
	users, err := ascClient().ListUsers()
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if strings.TrimSpace(os.Getenv("PLAY_DEVELOPER_ACCOUNT")) != "" {
		client, err := playClient()
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		playUsers, err := client.ListUsers("")
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		users = append(users, playUsers...)
	}
	writeJSON(w, http.StatusOK, users)
}

func (h Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func normalizeStore(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		value = "both"
	}
	switch value {
	case "ios", "android", "both":
		return value, nil
	default:
		return "", fmt.Errorf("invalid store %q", value)
	}
}

func wantsIOS(storeName string) bool {
	return storeName == "ios" || storeName == "both"
}

func wantsAndroid(storeName string) bool {
	return storeName == "android" || storeName == "both"
}

func appByAlias(alias string) (config.App, error) {
	app, ok := config.Apps[alias]
	if !ok {
		return config.App{}, fmt.Errorf("unknown app %q", alias)
	}
	return app, nil
}

func appSummaries() []store.AppSummary {
	aliases := make([]string, 0, len(config.Apps))
	for alias := range config.Apps {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)

	apps := make([]store.AppSummary, 0, len(aliases))
	for _, alias := range aliases {
		app := config.Apps[alias]
		apps = append(apps, store.AppSummary{
			Alias:       alias,
			Name:        app.Name,
			ASCID:       app.ASCAppID,
			PlayPackage: app.PlayPackage,
		})
	}
	return apps
}

func loadAppInstallStats(appID, appName, yearMonth, breakdown string) ([]store.InstallStat, error) {
	if strings.TrimSpace(appID) == "" {
		return nil, fmt.Errorf("app %q has no App Store Connect ID", appName)
	}
	analyticsClient := ascAnalyticsClient()
	rows, err := analyticsClient.GetDownloadStats(appID)
	if err != nil {
		rows, err = ascClient().GetDownloadStats(appID)
		if err != nil {
			return nil, fmt.Errorf("ios download stats unavailable: %w", err)
		}
	}
	stats := store.ParseInstallStats(rows, "ios", appName)

	deletionRows, err := analyticsClient.GetInstallDeletionStats(appID)
	if err != nil {
		deletionRows, err = ascClient().GetInstallDeletionStats(appID)
	}
	if err == nil {
		stats = store.MergeInstallStats(stats, store.ParseInstallStats(deletionRows, "ios", appName))
	}

	stats = store.FilterInstallStatsMonth(stats, yearMonth)
	if strings.TrimSpace(breakdown) != "" && breakdown != "overview" {
		filtered := make([]store.InstallStat, 0, len(stats))
		for _, stat := range stats {
			if stat.Dimension == breakdown {
				filtered = append(filtered, stat)
			}
		}
		stats = filtered
	}
	return stats, nil
}
