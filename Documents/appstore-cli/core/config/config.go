package config

import (
	"os"
	"path/filepath"
)

type App struct {
	Name              string
	ASCAppID          string
	PlayPackage       string
	FirebaseProjectID string // GCP project ID for Firebase APIs (e.g. "dallaslabs-issspotter")
	GA4PropertyID     string // GA4 property ID (e.g. "123456789") linked to this Firebase app
}

var Apps = map[string]App{
	"venus":         {Name: "Venus Companion", ASCAppID: "6754580293", PlayPackage: "com.dallaslabs.venus"},
	"polity-now":    {Name: "Polity Now", ASCAppID: "6758637143", PlayPackage: "org.dallaslabs.estado"},
	"iss-spotter":   {Name: "ISSSpotterApp", ASCAppID: "6757094244", PlayPackage: "org.dallaslabs.issspotter"},
	"quake":         {Name: "Quake Companion", ASCAppID: "6758176095", PlayPackage: "org.dallaslabs.quakealert"},
	"cabinet-doors": {Name: "Cabinet Doors Calculator", ASCAppID: "6755406345", PlayPackage: "org.dallaslabs.cabinetdoors.cabinetdoors"},
	"snake":         {Name: "SnakeInTheGrass", ASCAppID: "6472907910", PlayPackage: "com.bluecodingriver.snake-game-ios"},
	"kmp-demo":      {Name: "KmpPushNotificationDemo", ASCAppID: "6759348156", PlayPackage: ""},
}

type Credentials struct {
	ASCKeyID             string
	ASCIssuerID          string
	ASCKeyFile           string
	ASCAnalyticsKeyID    string
	ASCAnalyticsIssuerID string
	ASCAnalyticsKeyFile  string
	ASCVendorID          string
	PlayKeyFile          string
	FirebaseProjectID    string // env: FIREBASE_PROJECT_ID
}

func Load() Credentials {
	home, _ := os.UserHomeDir()
	ascKeyID := getenv("ASC_KEY_ID", "746FSCD2PK")
	ascIssuerID := getenv("ASC_ISSUER_ID", "e68d90fb-c9b8-41dd-ad08-947afe7459ae")
	ascKeyFile := getenv("ASC_KEY_FILE", filepath.Join(home, "Downloads", "AuthKey_746FSCD2PK.p8"))

	analyticsKeyID := getenv("ASC_ANALYTICS_KEY_ID", "79PWD5WB3S")
	analyticsIssuerID := getenv("ASC_ANALYTICS_ISSUER_ID", ascIssuerID)
	analyticsKeyFile := getenv("ASC_ANALYTICS_KEY_FILE", filepath.Join(home, "Downloads", "AuthKey_79PWD5WB3S.p8"))

	return Credentials{
		ASCKeyID:             ascKeyID,
		ASCIssuerID:          ascIssuerID,
		ASCKeyFile:           ascKeyFile,
		ASCAnalyticsKeyID:    analyticsKeyID,
		ASCAnalyticsIssuerID: analyticsIssuerID,
		ASCAnalyticsKeyFile:  analyticsKeyFile,
		ASCVendorID:          getenv("ASC_VENDOR_ID", "92547182"),
		PlayKeyFile:          getenv("PLAY_KEY_FILE", filepath.Join(home, "Downloads", "play-service-account.json")),
		FirebaseProjectID:    getenv("FIREBASE_PROJECT_ID", ""),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
