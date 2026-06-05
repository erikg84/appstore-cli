package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/dallaslabs/appctl/core/config"
)

type diagCheck struct {
	Name    string `json:"name"`
	Verdict string `json:"verdict"`
	Detail  string `json:"detail,omitempty"`
	Fix     string `json:"fix,omitempty"`
}

type diagResult struct {
	Command string      `json:"command"`
	Verdict string      `json:"verdict"`
	Checks  []diagCheck `json:"checks"`
	Summary map[string]int `json:"summary"`
}

func newDiagCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diag",
		Short: "Validate credentials and API connectivity",
		Long:  "Checks that all required credentials are present and that the API endpoints are reachable.",
		RunE: func(cmd *cobra.Command, args []string) error {
			creds := config.Load()
			var checks []diagCheck
			pass, warn, fail := 0, 0, 0

			// ASC key file
			if _, err := os.Stat(creds.ASCKeyFile); err == nil {
				checks = append(checks, diagCheck{"asc_key", "pass", creds.ASCKeyFile, ""})
				pass++
			} else {
				checks = append(checks, diagCheck{"asc_key", "fail", creds.ASCKeyFile,
					"Set ASC_KEY_FILE env var to path of your .p8 key"})
				fail++
			}

			// Play service account
			if _, err := os.Stat(creds.PlayKeyFile); err == nil {
				checks = append(checks, diagCheck{"play_key", "pass", creds.PlayKeyFile, ""})
				pass++
			} else {
				checks = append(checks, diagCheck{"play_key", "fail", creds.PlayKeyFile,
					"Set PLAY_KEY_FILE env var to path of your service account JSON"})
				fail++
			}

			// Firebase project ID
			if creds.FirebaseProjectID != "" {
				checks = append(checks, diagCheck{"firebase_project", "pass", creds.FirebaseProjectID, ""})
				pass++
			} else {
				checks = append(checks, diagCheck{"firebase_project", "warn", "(not set)",
					"Set FIREBASE_PROJECT_ID for firebase metrics/crashes commands"})
				warn++
			}

			// appctl-server (optional)
			if serverAddr != "" {
				_, err := exec.Command("curl", "-sf", serverAddr+"/healthz").Output()
				if err == nil {
					checks = append(checks, diagCheck{"server", "pass", serverAddr, ""})
					pass++
				} else {
					checks = append(checks, diagCheck{"server", "fail", serverAddr,
						"appctl-server not reachable at " + serverAddr})
					fail++
				}
			} else {
				checks = append(checks, diagCheck{"server", "warn", "(not configured)",
					"Pass --server http://host:port to proxy through appctl-server"})
				warn++
			}

			// Analytics key (optional)
			analyticsKeyFile := creds.ASCAnalyticsKeyFile
			if _, err := os.Stat(analyticsKeyFile); err == nil {
				checks = append(checks, diagCheck{"asc_analytics_key", "pass",
					filepath.Base(analyticsKeyFile), ""})
				pass++
			} else {
				checks = append(checks, diagCheck{"asc_analytics_key", "warn",
					"(not found)", "Optional: set ASC_ANALYTICS_KEY_FILE for richer installs data"})
				warn++
			}

			overall := "pass"
			if fail > 0 {
				overall = "fail"
			} else if warn > 0 {
				overall = "warn"
			}

			result := diagResult{
				Command: "diag",
				Verdict: overall,
				Checks:  checks,
				Summary: map[string]int{"pass": pass, "warn": warn, "fail": fail},
			}

			return render(result, func() {
				fmt.Println("appctl Diagnostics")
				fmt.Println("──────────────────")
				for _, c := range checks {
					verdict := c.Verdict
					if c.Fix != "" {
						fmt.Printf("  %-22s %-6s  %s\n", c.Name, verdict, c.Detail)
						fmt.Printf("  %-22s        Fix: %s\n", "", c.Fix)
					} else {
						fmt.Printf("  %-22s %-6s  %s\n", c.Name, verdict, c.Detail)
					}
				}
				fmt.Printf("\n%d checks: %d pass, %d warn, %d fail\n",
					pass+warn+fail, pass, warn, fail)
			})
		},
	}
}

func newCheatsheetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cheatsheet",
		Short: "Quick reference for all appctl commands",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(appctlCheatsheet)
		},
	}
}

const appctlCheatsheet = `# appctl cheatsheet

## Apps
appctl apps                        List all configured apps
appctl apps --output json          Machine-readable app list

## Store data
appctl versions --app iss-spotter  App Store versions
appctl builds   --app iss-spotter  TestFlight builds
appctl tracks   --app iss-spotter  Play release tracks
appctl reviews  --app iss-spotter  Recent reviews (iOS + Android)
appctl installs --app iss-spotter  Install and download stats

## IAP / Subscriptions
appctl iap            --app iss-spotter  One-time products
appctl subscriptions  --app iss-spotter  Subscription groups
appctl testflight     --app iss-spotter  Beta groups and testers

## Reports
appctl reports --app iss-spotter --type sales
appctl reports --app iss-spotter --type earnings

## Firebase (requires FIREBASE_PROJECT_ID + service account)
appctl firebase metrics  --app iss-spotter          DAU, MAU, crash-free, ARPU
appctl firebase crashes  --app iss-spotter --days 7 Top crash issues
appctl firebase revenue  --app iss-spotter          Revenue breakdown

## Output
--output json    Machine-readable JSON with schema_version envelope
--output table   Human-readable table (default)
--no-header      Suppress table header row

## Proxy
--server http://host:8080   Route all requests through appctl-server

## Diagnostics
appctl diag      Check credentials and connectivity
appctl help <command>        Detailed help for any command

## Exit codes
0 = success  1 = failure  2 = usage error
`
