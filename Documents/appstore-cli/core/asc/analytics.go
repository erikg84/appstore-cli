package asc

import (
	"bytes"
	"compress/gzip"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// GetDownloadStats fetches "App Downloads Standard" analytics data for an app.
// Returns raw CSV rows from the most recent available instance.
func (c *Client) GetDownloadStats(appID string) ([][]string, error) {
	return c.getAnalyticsReportRows(appID, "App Downloads Standard")
}

// GetInstallDeletionStats fetches "App Store Installation and Deletion Standard"
// analytics data for an app and returns raw CSV rows.
func (c *Client) GetInstallDeletionStats(appID string) ([][]string, error) {
	return c.getAnalyticsReportRows(appID, "App Store Installation and Deletion Standard")
}

func (c *Client) getAnalyticsReportRows(appID, reportName string) ([][]string, error) {
	requestID, err := c.ensureOngoingAnalyticsRequest(appID)
	if err != nil {
		return nil, fmt.Errorf("ensure analytics request: %w", err)
	}

	reportID, err := c.findAnalyticsReportID(requestID, reportName)
	if err != nil {
		return nil, fmt.Errorf("find analytics report %q: %w", reportName, err)
	}

	instanceID, err := c.latestAnalyticsInstanceID(reportID)
	if err != nil {
		return nil, fmt.Errorf("find latest analytics instance: %w", err)
	}

	segmentURLs, err := c.analyticsSegmentURLs(instanceID)
	if err != nil {
		return nil, fmt.Errorf("list analytics segments: %w", err)
	}
	if len(segmentURLs) == 0 {
		return nil, fmt.Errorf("no analytics segments available for %q", reportName)
	}

	var allRows [][]string
	var header []string
	for _, segmentURL := range segmentURLs {
		rows, err := c.downloadAnalyticsSegment(segmentURL)
		if err != nil {
			return nil, fmt.Errorf("download analytics segment: %w", err)
		}
		for _, row := range rows {
			if len(row) == 0 {
				continue
			}
			if header == nil {
				header = row
				allRows = append(allRows, row)
				continue
			}
			if csvRowsEqual(row, header) {
				continue
			}
			allRows = append(allRows, row)
		}
	}
	return allRows, nil
}

func (c *Client) ensureOngoingAnalyticsRequest(appID string) (string, error) {
	// App-scoped listing works for more roles than the global collection endpoint.
	if requestID, err := c.findAppScopedAnalyticsRequest(appID); err == nil && requestID != "" {
		return requestID, nil
	}

	// Backward-compatible fallback for accounts/roles where global listing is allowed.
	query := url.Values{}
	query.Set("filter[app]", appID)
	query.Set("filter[accessType]", "ONGOING")
	query.Set("limit", "1")

	var payload map[string]any
	if err := c.getJSON("/v1/analyticsReportRequests", query, &payload); err != nil {
		return "", err
	}
	if items := dataItems(payload); len(items) > 0 {
		return str(items[0], "id"), nil
	}

	request := map[string]any{
		"data": map[string]any{
			"type": "analyticsReportRequests",
			"attributes": map[string]any{
				"accessType": "ONGOING",
			},
			"relationships": map[string]any{
				"app": map[string]any{
					"data": map[string]any{
						"type": "apps",
						"id":   appID,
					},
				},
			},
		},
	}
	payload = map[string]any{}
	if err := c.postJSON("/v1/analyticsReportRequests", request, &payload); err != nil {
		return "", err
	}
	if items := dataItems(payload); len(items) > 0 {
		if id := str(items[0], "id"); id != "" {
			return id, nil
		}
	}
	if id := str(payload, "id"); id != "" {
		return id, nil
	}
	return "", fmt.Errorf("analytics request creation returned no id")
}

func (c *Client) findAppScopedAnalyticsRequest(appID string) (string, error) {
	query := url.Values{}
	query.Set("filter[accessType]", "ONGOING")
	query.Set("limit", "1")

	var payload map[string]any
	if err := c.getJSON(fmt.Sprintf("/v1/apps/%s/analyticsReportRequests", url.PathEscape(appID)), query, &payload); err != nil {
		return "", err
	}
	if items := dataItems(payload); len(items) > 0 {
		return str(items[0], "id"), nil
	}
	return "", nil
}

func (c *Client) findAnalyticsReportID(requestID, reportName string) (string, error) {
	query := url.Values{}
	query.Set("filter[name]", reportName)

	var payload map[string]any
	if err := c.getJSON(fmt.Sprintf("/v1/analyticsReportRequests/%s/reports", url.PathEscape(requestID)), query, &payload); err != nil {
		return "", err
	}
	for _, item := range dataItems(payload) {
		if strings.EqualFold(strings.TrimSpace(firstNonEmpty(str(attrs(item), "name"), str(item, "name"))), reportName) {
			return str(item, "id"), nil
		}
	}
	return "", fmt.Errorf("report not found")
}

func (c *Client) latestAnalyticsInstanceID(reportID string) (string, error) {
	query := url.Values{}
	query.Set("limit", "1")

	var payload map[string]any
	if err := c.getJSON(fmt.Sprintf("/v1/analyticsReports/%s/instances", url.PathEscape(reportID)), query, &payload); err != nil {
		return "", err
	}
	if items := dataItems(payload); len(items) > 0 {
		return str(items[0], "id"), nil
	}
	return "", fmt.Errorf("no analytics instances found")
}

func (c *Client) analyticsSegmentURLs(instanceID string) ([]string, error) {
	var payload map[string]any
	if err := c.getJSON(fmt.Sprintf("/v1/analyticsReportInstances/%s/segments", url.PathEscape(instanceID)), nil, &payload); err != nil {
		return nil, err
	}

	urls := make([]string, 0, len(dataItems(payload)))
	for _, item := range dataItems(payload) {
		segmentURL := firstNonEmpty(str(item, "url"), str(attrs(item), "url"))
		if strings.TrimSpace(segmentURL) != "" {
			urls = append(urls, segmentURL)
		}
	}
	return urls, nil
}

func (c *Client) downloadAnalyticsSegment(segmentURL string) ([][]string, error) {
	body, headers, err := c.externalGet(segmentURL, true)
	if err != nil {
		body, headers, err = c.externalGet(segmentURL, false)
		if err != nil {
			return nil, err
		}
	}

	reader := io.Reader(bytes.NewReader(body))
	if strings.EqualFold(headers.Get("Content-Encoding"), "gzip") || (len(body) >= 2 && body[0] == 0x1f && body[1] == 0x8b) {
		gz, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("decompress analytics segment: %w", err)
		}
		defer gz.Close()
		reader = gz
	}

	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = -1
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (c *Client) externalGet(rawURL string, withAuth bool) ([]byte, http.Header, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, nil, err
	}
	if withAuth {
		token, err := c.bearerToken()
		if err != nil {
			return nil, nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.Header, err
	}
	if resp.StatusCode >= 300 {
		return nil, resp.Header, fmt.Errorf("analytics segment GET %s: %s", rawURL, truncate(body))
	}
	return body, resp.Header, nil
}

func (c *Client) postJSON(path string, payload any, target any) error {
	token, err := c.bearerToken()
	if err != nil {
		return err
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("app store connect %s %s: %s", req.Method, path, truncate(responseBody))
	}
	if target == nil || len(responseBody) == 0 {
		return nil
	}
	return jsonUnmarshal(responseBody, target)
}

func csvRowsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if strings.TrimSpace(a[i]) != strings.TrimSpace(b[i]) {
			return false
		}
	}
	return true
}
