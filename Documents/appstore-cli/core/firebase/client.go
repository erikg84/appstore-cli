// Package firebase provides clients for Firebase Analytics, Crashlytics,
// and Performance Monitoring via Google service account credentials.
package firebase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/dallaslabs/appctl/core/store"
)

const (
	analyticsScope   = "https://www.googleapis.com/auth/analytics.readonly"
	cloudScope       = "https://www.googleapis.com/auth/cloud-platform"
)

// Client wraps Firebase API access using a Google service account.
type Client struct {
	projectID string
	http      *http.Client
}

// NewClient creates a Firebase client for the given project.
func NewClient(serviceAccountFile, projectID string) (*Client, error) {
	if projectID == "" {
		return nil, fmt.Errorf("FIREBASE_PROJECT_ID not set — required for Firebase commands")
	}
	keyData, err := os.ReadFile(serviceAccountFile)
	if err != nil {
		return nil, fmt.Errorf("read service account key: %w", err)
	}
	ctx := context.Background()
	creds, err := google.CredentialsFromJSON(ctx, keyData, analyticsScope, cloudScope)
	if err != nil {
		return nil, fmt.Errorf("parse credentials: %w", err)
	}
	return &Client{projectID: projectID, http: oauth2.NewClient(ctx, creds.TokenSource)}, nil
}

// GetMetrics fetches app metrics from the GA4 Data API.
// ga4PropertyID must be the numeric property ID (e.g. "123456789").
func (c *Client) GetMetrics(ga4PropertyID string, days int) (store.AppMetrics, error) {
	if ga4PropertyID == "" {
		return store.AppMetrics{}, fmt.Errorf("GA4_PROPERTY_ID not set for this app — add to app registry")
	}
	end := time.Now().UTC().Format("2006-01-02")
	start := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")

	url := fmt.Sprintf("https://analyticsdata.googleapis.com/v1beta/properties/%s:runReport", ga4PropertyID)
	body := map[string]any{
		"dateRanges": []map[string]string{{"startDate": start, "endDate": end}},
		"metrics": []map[string]string{
			{"name": "activeUsers"},
			{"name": "active28DayUsers"},
			{"name": "averageSessionDuration"},
			{"name": "crashFreeUsersRate"},
			{"name": "totalRevenue"},
		},
	}
	data, err := c.postJSON(url, body)
	if err != nil {
		return store.AppMetrics{}, fmt.Errorf("GA4 metrics: %w", err)
	}
	return parseMetrics(data, days), nil
}

func parseMetrics(data map[string]any, days int) store.AppMetrics {
	m := store.AppMetrics{PeriodDays: days}
	rows, _ := data["rows"].([]any)
	headers, _ := data["metricHeaders"].([]any)
	if len(rows) == 0 {
		return m
	}
	idx := map[string]int{}
	for i, h := range headers {
		hm, _ := h.(map[string]any)
		if name, _ := hm["name"].(string); name != "" {
			idx[name] = i
		}
	}
	rm, _ := rows[0].(map[string]any)
	vals, _ := rm["metricValues"].([]any)
	get := func(name string) float64 {
		i, ok := idx[name]
		if !ok || i >= len(vals) {
			return 0
		}
		vm, _ := vals[i].(map[string]any)
		v, _ := vm["value"].(string)
		var f float64
		fmt.Sscanf(v, "%f", &f)
		return f
	}
	m.DAU = int(get("activeUsers"))
	m.MAU = int(get("active28DayUsers"))
	m.CrashFreeRate = get("crashFreeUsersRate")
	m.AvgSessionDuration = get("averageSessionDuration")
	m.RevenueUSD = get("totalRevenue")
	if m.MAU > 0 {
		m.ARPUUSD = m.RevenueUSD / float64(m.MAU)
	}
	return m
}

// GetCrashReport fetches crash issues from Firebase Crashlytics.
func (c *Client) GetCrashReport(androidPackage string, days int) (store.CrashReport, error) {
	url := fmt.Sprintf("https://firebasecrashlytics.googleapis.com/v1beta/projects/%s:queryCrashlytics", c.projectID)
	end := time.Now().UTC()
	start := end.AddDate(0, 0, -days)
	body := map[string]any{
		"filter": fmt.Sprintf(`package_name = "%s"`, androidPackage),
		"timeRange": map[string]string{
			"startTime": start.Format(time.RFC3339),
			"endTime":   end.Format(time.RFC3339),
		},
	}
	data, err := c.postJSON(url, body)
	if err != nil {
		return store.CrashReport{}, fmt.Errorf("Crashlytics: %w (ensure service account has Firebase Crashlytics Viewer)", err)
	}
	return parseCrashes(data, days), nil
}

func parseCrashes(data map[string]any, days int) store.CrashReport {
	r := store.CrashReport{Period: fmt.Sprintf("%dd", days)}
	issues, _ := data["issues"].([]any)
	for _, issue := range issues {
		im, _ := issue.(map[string]any)
		ev, _ := im["events"].(map[string]any)
		var count, users int
		fmt.Sscanf(fmt.Sprint(ev["count"]), "%d", &count)
		fmt.Sscanf(fmt.Sprint(ev["affectedUsers"]), "%d", &users)
		r.TotalCrashes += count
		r.AffectedUsers += users
		r.TopIssues = append(r.TopIssues, map[string]any{
			"id":             im["issueId"],
			"title":          im["title"],
			"count":          count,
			"affected_users": users,
		})
	}
	return r
}

// ── HTTP helper ───────────────────────────────────────────────────────────────

func (c *Client) postJSON(url string, body map[string]any) (map[string]any, error) {
	payload, _ := json.Marshal(body)
	resp, err := c.http.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(raw))
	}
	var result map[string]any
	return result, json.Unmarshal(raw, &result)
}
