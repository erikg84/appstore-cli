package play

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"strings"
	"unicode/utf16"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/storage/v1"
)

const pubsiteBucket = "pubsite_prod_5992512819868906729"

// ReportFile describes a file available in the GCS pubsite bucket.
type ReportFile struct {
	Category string `json:"category"` // earnings, sales, reviews, stats/crashes, stats/installs, etc.
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Object   string `json:"object"` // full GCS object path
}

// ListReportFiles returns available report files, optionally filtered by category prefix
// (e.g. "earnings", "sales", "reviews", "stats/crashes", "stats/installs").
func (c *Client) ListReportFiles(categoryPrefix string) ([]ReportFile, error) {
	gcs, err := c.gcsService()
	if err != nil {
		return nil, err
	}

	var files []ReportFile
	prefix := categoryPrefix
	call := gcs.Objects.List(pubsiteBucket).Prefix(prefix)
	if err := call.Pages(context.Background(), func(page *storage.Objects) error {
		if page == nil || len(page.Items) == 0 {
			return nil
		}
		for _, obj := range page.Items {
			cat := path.Dir(obj.Name)
			if cat == "." {
				cat = ""
			}
			files = append(files, ReportFile{
				Category: cat,
				Name:     path.Base(obj.Name),
				Size:     int64(obj.Size),
				Object:   obj.Name,
			})
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return files, nil
}

// DownloadReportCSV downloads a report object from GCS and returns its rows as
// [][]string. Handles both UTF-16 LE/BE encoded CSVs (Play format) and plain
// UTF-8. ZIP archives are automatically extracted — the first .csv inside is used.
func (c *Client) DownloadReportCSV(objectPath string) ([][]string, error) {
	gcs, err := c.gcsService()
	if err != nil {
		return nil, err
	}

	resp, err := gcs.Objects.Get(pubsiteBucket, objectPath).Download()
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", objectPath, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// If ZIP, extract the first CSV inside
	if strings.HasSuffix(strings.ToLower(objectPath), ".zip") {
		data, err = extractFirstCSVFromZip(data)
		if err != nil {
			return nil, err
		}
	}

	// Decode UTF-16 if BOM is present
	text, err := decodeUTF16(data)
	if err != nil {
		return nil, err
	}

	return parseCSV(text), nil
}

// DownloadEarnings downloads the latest earnings ZIP and returns rows.
func (c *Client) DownloadEarnings(yearMonth string) ([][]string, error) {
	files, err := c.ListReportFiles("earnings/")
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if yearMonth == "" || strings.Contains(f.Name, yearMonth) {
			return c.DownloadReportCSV(f.Object)
		}
	}
	if yearMonth != "" {
		return nil, fmt.Errorf("no earnings report found for %s", yearMonth)
	}
	return nil, fmt.Errorf("no earnings reports found")
}

// DownloadSales downloads the latest sales ZIP and returns rows.
func (c *Client) DownloadSales(yearMonth string) ([][]string, error) {
	files, err := c.ListReportFiles("sales/")
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if yearMonth == "" || strings.Contains(f.Name, yearMonth) {
			return c.DownloadReportCSV(f.Object)
		}
	}
	if yearMonth != "" {
		return nil, fmt.Errorf("no sales report found for %s", yearMonth)
	}
	return nil, fmt.Errorf("no sales reports found")
}

// DownloadInstallStats downloads install stats for a package from GCS.
func (c *Client) DownloadInstallStats(packageName, yearMonth, breakdown string) ([][]string, error) {
	if strings.TrimSpace(breakdown) == "" {
		breakdown = "overview"
	}
	files, err := c.ListReportFiles("stats/installs/")
	if err != nil {
		return nil, err
	}
	namePrefix := fmt.Sprintf("installs_%s_", packageName)
	nameSuffix := "_" + breakdown + ".csv"
	file := latestMatchingFile(files, namePrefix, nameSuffix, yearMonth)
	if file == nil {
		return nil, fmt.Errorf("no install stats found for %s (%s)", packageName, firstNonEmpty(yearMonth, breakdown))
	}
	return c.DownloadReportCSV(file.Object)
}

// DownloadCrashStats downloads crash stats for a package from GCS.
func (c *Client) DownloadCrashStats(packageName, yearMonth, breakdown string) ([][]string, error) {
	if strings.TrimSpace(breakdown) == "" {
		breakdown = "overview"
	}
	files, err := c.ListReportFiles("stats/crashes/")
	if err != nil {
		return nil, err
	}
	namePrefix := fmt.Sprintf("crashes_%s_", packageName)
	nameSuffix := "_" + breakdown + ".csv"
	file := latestMatchingFile(files, namePrefix, nameSuffix, yearMonth)
	if file == nil {
		return nil, fmt.Errorf("no crash stats found for %s (%s)", packageName, firstNonEmpty(yearMonth, breakdown))
	}
	return c.DownloadReportCSV(file.Object)
}

// DownloadAcquisitionStats downloads acquisition stats from GCS.
func (c *Client) DownloadAcquisitionStats(packageName, yearMonth, category string) ([][]string, error) {
	if strings.TrimSpace(category) == "" {
		category = "buyers_7d"
	}
	files, err := c.ListReportFiles("acquisition/" + category + "/")
	if err != nil {
		return nil, err
	}
	prefix := category + "_" + packageName + "_"
	var best *ReportFile
	bestMonth := ""
	bestRank := -1
	for _, f := range files {
		if !strings.HasPrefix(f.Name, prefix) || !strings.HasSuffix(strings.ToLower(f.Name), ".csv") {
			continue
		}
		month, dimension := acquisitionMonthAndDimension(f.Name)
		if yearMonth != "" && month != yearMonth {
			continue
		}
		rank := acquisitionDimensionRank(dimension)
		if best == nil || month > bestMonth || (month == bestMonth && rank > bestRank) || (month == bestMonth && rank == bestRank && f.Name > best.Name) {
			candidate := f
			best = &candidate
			bestMonth = month
			bestRank = rank
		}
	}
	if best == nil {
		return nil, fmt.Errorf("no acquisition stats found for %s (%s)", packageName, firstNonEmpty(yearMonth, category))
	}
	return c.DownloadReportCSV(best.Object)
}

// gcsService builds a storage.Service using the same SA credentials.
func (c *Client) gcsService() (*storage.Service, error) {
	// Extract the token source from the reporting client's transport
	ctx := context.Background()

	// We need a token source — re-read from the reporting service's HTTP client
	// by grabbing the underlying token source stored at client construction.
	ts := c.tokenSource
	if ts == nil {
		return nil, fmt.Errorf("no token source available for GCS")
	}
	return storage.NewService(ctx, option.WithTokenSource(ts))
}

// --- internal helpers ---

func extractFirstCSVFromZip(data []byte) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("zip open: %w", err)
	}
	for _, f := range r.File {
		if strings.HasSuffix(strings.ToLower(f.Name), ".csv") {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("no CSV found inside zip")
}

func decodeUTF16(data []byte) (string, error) {
	if len(data) < 2 {
		return string(data), nil
	}
	// UTF-16 LE BOM: FF FE
	if data[0] == 0xFF && data[1] == 0xFE {
		return decodeUTF16LE(data[2:]), nil
	}
	// UTF-16 BE BOM: FE FF
	if data[0] == 0xFE && data[1] == 0xFF {
		return decodeUTF16BE(data[2:]), nil
	}
	return string(data), nil
}

func decodeUTF16LE(b []byte) string {
	u16 := make([]uint16, len(b)/2)
	for i := range u16 {
		u16[i] = uint16(b[2*i]) | uint16(b[2*i+1])<<8
	}
	return string(utf16.Decode(u16))
}

func decodeUTF16BE(b []byte) string {
	u16 := make([]uint16, len(b)/2)
	for i := range u16 {
		u16[i] = uint16(b[2*i])<<8 | uint16(b[2*i+1])
	}
	return string(utf16.Decode(u16))
}

// parseCSV splits a CSV text into rows/columns. Handles quoted fields.
func parseCSV(text string) [][]string {
	var rows [][]string
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		rows = append(rows, splitCSVLine(line))
	}
	return rows
}

func splitCSVLine(line string) []string {
	var fields []string
	var cur strings.Builder
	inQuote := false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		switch {
		case ch == '"' && !inQuote:
			inQuote = true
		case ch == '"' && inQuote:
			if i+1 < len(line) && line[i+1] == '"' {
				cur.WriteByte('"')
				i++
			} else {
				inQuote = false
			}
		case ch == ',' && !inQuote:
			fields = append(fields, cur.String())
			cur.Reset()
		default:
			cur.WriteByte(ch)
		}
	}
	fields = append(fields, cur.String())
	return fields
}

func latestMatchingFile(files []ReportFile, namePrefix, nameSuffix, yearMonth string) *ReportFile {
	var best *ReportFile
	for _, f := range files {
		if !strings.HasPrefix(f.Name, namePrefix) || !strings.HasSuffix(f.Name, nameSuffix) {
			continue
		}
		if yearMonth != "" && !strings.Contains(f.Name, "_"+yearMonth+"_") {
			continue
		}
		if best == nil || f.Name > best.Name {
			candidate := f
			best = &candidate
		}
	}
	return best
}

func acquisitionMonthAndDimension(name string) (string, string) {
	base := strings.TrimSuffix(name, path.Ext(name))
	lastUnderscore := strings.LastIndex(base, "_")
	if lastUnderscore < 0 {
		return "", ""
	}
	dimension := base[lastUnderscore+1:]
	rest := base[:lastUnderscore]
	prevUnderscore := strings.LastIndex(rest, "_")
	if prevUnderscore < 0 {
		return "", dimension
	}
	return rest[prevUnderscore+1:], dimension
}

func acquisitionDimensionRank(dimension string) int {
	switch dimension {
	case "country":
		return 3
	case "play_country":
		return 2
	case "channel":
		return 1
	default:
		return 0
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// tokenSourceHolder is used so the client can pass its token source to GCS.
type tokenSourceHolder struct {
	ts oauth2.TokenSource
}
