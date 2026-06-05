package store

type AppSummary struct {
	Alias       string `json:"alias"`
	Name        string `json:"name"`
	ASCID       string `json:"ascId,omitempty"`
	PlayPackage string `json:"playPackage,omitempty"`
}

type Version struct {
	Platform      string `json:"platform"`
	VersionString string `json:"versionString"`
	State         string `json:"state"`
	CreatedDate   string `json:"createdDate"`
}

type Build struct {
	Version      string `json:"version"`
	State        string `json:"state"`
	UploadedDate string `json:"uploadedDate"`
	Platform     string `json:"platform"`
}

type Track struct {
	Name         string  `json:"name"`
	Status       string  `json:"status"`
	VersionCodes []int64 `json:"versionCodes"`
	VersionName  string  `json:"versionName"`
}

type Review struct {
	Store        string `json:"store"`
	Rating       int    `json:"rating"`
	Title        string `json:"title,omitempty"`
	Body         string `json:"body"`
	ReviewerName string `json:"reviewerName,omitempty"`
	Date         string `json:"date"`
	AppVersion   string `json:"appVersion,omitempty"`
}

type IAP struct {
	ID          string `json:"id"`
	ProductID   string `json:"productId"`
	ProductType string `json:"productType"`
	Name        string `json:"name"`
	State       string `json:"state"`
	Price       string `json:"price,omitempty"`
}

type SubscriptionGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Subscription struct {
	ID         string `json:"id"`
	ProductID  string `json:"productId"`
	GroupID    string `json:"groupId,omitempty"`
	Name       string `json:"name"`
	State      string `json:"state"`
	ReviewNote string `json:"reviewNote,omitempty"`
}

type BetaGroup struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	IsInternal  bool   `json:"isInternal"`
	TesterCount int    `json:"testerCount"`
}

type BetaTester struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	State     string `json:"state"`
}

type SalesReport struct {
	Date        string  `json:"date"`
	Units       int     `json:"units"`
	Revenue     float64 `json:"revenue"`
	ProductID   string  `json:"productId"`
	CountryCode string  `json:"countryCode,omitempty"`
	Currency    string  `json:"currency,omitempty"`
}

type InstallStat struct {
	Source        string `json:"source"`
	Date          string `json:"date"`
	App           string `json:"app"`
	Installs      int64  `json:"installs"`
	Uninstalls    int64  `json:"uninstalls"`
	ActiveDevices int64  `json:"activeDevices"`
	Dimension     string `json:"dimension"`
	DimValue      string `json:"dimValue"`
}

type CrashStat struct {
	Source    string  `json:"source"`
	Date      string  `json:"date"`
	App       string  `json:"app"`
	CrashRate float64 `json:"crashRate"`
	Crashes   int64   `json:"crashes"`
	Dimension string  `json:"dimension"`
	DimValue  string  `json:"dimValue"`
}

type AppUser struct {
	ID             string   `json:"id"`
	Email          string   `json:"email"`
	Roles          []string `json:"roles"`
	AllAppsVisible bool     `json:"allAppsVisible"`
}

// ── Firebase types ────────────────────────────────────────────────────────────

// AppMetrics holds aggregate health metrics from Firebase Analytics (GA4).
type AppMetrics struct {
	AppID              string  `json:"app_id"`
	PeriodDays         int     `json:"period_days"`
	DAU                int     `json:"dau"`
	MAU                int     `json:"mau"`
	RetentionD1        float64 `json:"retention_d1"`
	RetentionD7        float64 `json:"retention_d7"`
	RetentionD30       float64 `json:"retention_d30"`
	AvgSessionDuration float64 `json:"avg_session_duration_s"`
	CrashFreeRate      float64 `json:"crash_free_rate"`
	RevenueUSD         float64 `json:"revenue_usd"`
	ARPUUSD            float64 `json:"arpu_usd"`
}

// CrashReport summarizes crash data from Firebase Crashlytics.
type CrashReport struct {
	AppID         string           `json:"app_id"`
	Period        string           `json:"period"`
	TotalCrashes  int              `json:"total_crashes"`
	AffectedUsers int              `json:"affected_users"`
	CrashFreeRate float64          `json:"crash_free_rate"`
	TopIssues     []map[string]any `json:"top_issues"`
}

// RevenueData holds revenue breakdown from Play / ASC reports.
type RevenueData struct {
	AppID            string           `json:"app_id"`
	Period           string           `json:"period"`
	TotalUSD         float64          `json:"total_usd"`
	IAPUSD           float64          `json:"iap_usd"`
	AdsUSD           float64          `json:"ads_usd"`
	SubscriptionsUSD float64          `json:"subscriptions_usd"`
	Trend7d          float64          `json:"trend_7d"`
	Trend30d         float64          `json:"trend_30d"`
	ByCountry        []map[string]any `json:"by_country"`
}
