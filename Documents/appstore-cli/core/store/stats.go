package store

import (
	"strconv"
	"strings"
)

func ParseInstallStats(rows [][]string, source, app string) []InstallStat {
	if len(rows) == 0 {
		return nil
	}

	headers := rows[0]
	normalized := normalizeHeaders(headers)
	dimIndex, dimension := detectDimension(headers, normalized)

	stats := make([]InstallStat, 0, max(0, len(rows)-1))
	for _, row := range rows[1:] {
		if rowEmpty(row) {
			continue
		}
		stat := InstallStat{
			Source:        source,
			Date:          csvValue(row, normalized, "date", "processingdate"),
			App:           firstValue(app, csvValue(row, normalized, "app", "appname", "packagename", "package", "bundleid")),
			Installs:      parseInt64Value(metricValue(row, normalized, []string{"dailydeviceinstalls", "dailyuserinstalls", "installs", "appdownloads", "downloads", "totaldownloads", "firsttimedownloads"}, []string{"downloads", "install"})),
			Uninstalls:    parseInt64Value(metricValue(row, normalized, []string{"dailydeviceuninstalls", "dailyuseruninstalls", "uninstalls", "deletions", "deletion"}, []string{"uninstall", "deletion", "delete"})),
			ActiveDevices: parseInt64Value(metricValue(row, normalized, []string{"activedeviceinstalls", "activedevices", "activeinstalls", "devicesactive"}, []string{"activedevice", "activedevices", "active devices"})),
			Dimension:     dimension,
		}
		if dimIndex >= 0 && dimIndex < len(row) {
			stat.DimValue = strings.TrimSpace(row[dimIndex])
		}
		if stat.DimValue == "" {
			stat.Dimension = ""
		}
		stats = append(stats, stat)
	}
	return stats
}

func ParseCrashStats(rows [][]string, source, app string) []CrashStat {
	if len(rows) == 0 {
		return nil
	}

	headers := rows[0]
	normalized := normalizeHeaders(headers)
	dimIndex, dimension := detectDimension(headers, normalized)

	stats := make([]CrashStat, 0, max(0, len(rows)-1))
	for _, row := range rows[1:] {
		if rowEmpty(row) {
			continue
		}
		stat := CrashStat{
			Source:    source,
			Date:      csvValue(row, normalized, "date", "processingdate"),
			App:       firstValue(app, csvValue(row, normalized, "app", "appname", "packagename", "package", "bundleid")),
			CrashRate: parseFloatValue(metricValue(row, normalized, []string{"crashrate", "dailycrashrate"}, []string{"crashrate", "crash rate"})),
			Crashes:   parseInt64Value(metricValue(row, normalized, []string{"dailycrashes", "crashes"}, []string{"crashes", "crash"})),
			Dimension: dimension,
		}
		if dimIndex >= 0 && dimIndex < len(row) {
			stat.DimValue = strings.TrimSpace(row[dimIndex])
		}
		if stat.DimValue == "" {
			stat.Dimension = ""
		}
		stats = append(stats, stat)
	}
	return stats
}

func MergeInstallStats(groups ...[]InstallStat) []InstallStat {
	merged := make([]InstallStat, 0)
	index := map[string]int{}

	for _, group := range groups {
		for _, stat := range group {
			key := strings.Join([]string{stat.Source, stat.Date, stat.App, stat.Dimension, stat.DimValue}, "\x1f")
			if i, ok := index[key]; ok {
				current := merged[i]
				if stat.Installs != 0 {
					current.Installs = stat.Installs
				}
				if stat.Uninstalls != 0 {
					current.Uninstalls = stat.Uninstalls
				}
				if stat.ActiveDevices != 0 {
					current.ActiveDevices = stat.ActiveDevices
				}
				merged[i] = current
				continue
			}
			index[key] = len(merged)
			merged = append(merged, stat)
		}
	}
	return merged
}

func FilterInstallStatsMonth(stats []InstallStat, yearMonth string) []InstallStat {
	if strings.TrimSpace(yearMonth) == "" {
		return stats
	}
	filtered := make([]InstallStat, 0, len(stats))
	for _, stat := range stats {
		if matchesYearMonth(stat.Date, yearMonth) {
			filtered = append(filtered, stat)
		}
	}
	return filtered
}

func normalizeHeaders(headers []string) map[string]int {
	index := make(map[string]int, len(headers))
	for i, header := range headers {
		index[normalizeKey(header)] = i
	}
	return index
}

func detectDimension(headers []string, normalized map[string]int) (int, string) {
	for i, header := range headers {
		key := normalizeKey(header)
		if key == "" || isMetricHeader(key) {
			continue
		}
		if dimension := mapDimension(header, key); dimension != "" {
			return i, dimension
		}
	}
	return -1, ""
}

func isMetricHeader(key string) bool {
	switch key {
	case "date", "processingdate", "app", "appname", "packagename", "package", "bundleid",
		"dailydeviceinstalls", "dailydeviceuninstalls", "dailydeviceupgrades", "totaluserinstalls",
		"dailyuserinstalls", "dailyuseruninstalls", "activedeviceinstalls", "installevents",
		"updateevents", "uninstallevents", "dailycrashes", "dailyanrs", "crashes", "anrs",
		"crashrate", "dailycrashrate", "downloads", "appdownloads", "firsttimedownloads",
		"deletions", "deletion", "activeinstalls", "activedevices", "devicesactive":
		return true
	}
	return strings.Contains(key, "install") || strings.Contains(key, "uninstall") || strings.Contains(key, "crash") || strings.Contains(key, "download") || strings.Contains(key, "active") || strings.Contains(key, "delete") || strings.Contains(key, "anr")
}

func mapDimension(header, key string) string {
	switch {
	case strings.Contains(key, "country") || strings.Contains(key, "territory"):
		return "country"
	case strings.Contains(key, "device"):
		return "device"
	case strings.Contains(key, "osversion") || strings.Contains(key, "platformversion"):
		return "os_version"
	case strings.Contains(key, "appversion") || key == "version":
		return "app_version"
	case strings.Contains(key, "language") || strings.Contains(key, "locale"):
		return "language"
	case strings.Contains(key, "carrier"):
		return "carrier"
	case strings.Contains(key, "channel"):
		return "channel"
	}
	normalized := strings.Trim(strings.Join(strings.FieldsFunc(strings.ToLower(header), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	}), "_"), "_")
	if normalized == "" {
		return ""
	}
	return normalized
}

func csvValue(row []string, normalized map[string]int, names ...string) string {
	for _, name := range names {
		if i, ok := normalized[normalizeKey(name)]; ok && i < len(row) {
			return strings.TrimSpace(row[i])
		}
	}
	return ""
}

func metricValue(row []string, normalized map[string]int, exact []string, contains []string) string {
	if value := csvValue(row, normalized, exact...); value != "" {
		return value
	}
	for key, i := range normalized {
		if i >= len(row) {
			continue
		}
		for _, part := range contains {
			if strings.Contains(key, normalizeKey(part)) {
				return strings.TrimSpace(row[i])
			}
		}
	}
	return ""
}

func normalizeKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func rowEmpty(row []string) bool {
	for _, value := range row {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func matchesYearMonth(date, yearMonth string) bool {
	return strings.HasPrefix(strings.ReplaceAll(strings.TrimSpace(date), "-", ""), strings.TrimSpace(yearMonth))
}

func parseInt64Value(value string) int64 {
	value = strings.TrimSpace(strings.ReplaceAll(value, ",", ""))
	if value == "" {
		return 0
	}
	n, _ := strconv.ParseInt(value, 10, 64)
	return n
}

func parseFloatValue(value string) float64 {
	value = strings.TrimSpace(strings.ReplaceAll(value, ",", ""))
	if value == "" {
		return 0
	}
	n, _ := strconv.ParseFloat(value, 64)
	return n
}

func firstValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
