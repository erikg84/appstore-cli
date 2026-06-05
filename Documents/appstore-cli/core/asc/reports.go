package asc

import (
    "bytes"
    "compress/gzip"
    "encoding/csv"
    "fmt"
    "io"
    "net/url"
    "strconv"
    "strings"

    "github.com/dallaslabs/appctl/core/store"
)

func (c *Client) GetSalesReport(vendorNumber, date, frequency string) ([]store.SalesReport, error) {
    query := url.Values{}
    query.Set("filter[reportType]", "SALES")
    query.Set("filter[reportSubType]", "SUMMARY")
    query.Set("filter[frequency]", strings.ToUpper(firstNonEmpty(frequency, "MONTHLY")))
    query.Set("filter[reportDate]", date)
    query.Set("filter[vendorNumber]", vendorNumber)

    body, headers, err := c.getWithAccept("/v1/salesReports", query, "application/a-gzip")
    if err != nil {
        return nil, err
    }

    reader := io.Reader(bytes.NewReader(body))
    if headers.Get("Content-Encoding") == "gzip" || (len(body) >= 2 && body[0] == 0x1f && body[1] == 0x8b) {
        gz, err := gzip.NewReader(bytes.NewReader(body))
        if err != nil {
            return nil, fmt.Errorf("decompress sales report: %w", err)
        }
        defer gz.Close()
        reader = gz
    }

    csvReader := csv.NewReader(reader)
    csvReader.Comma = '\t'
    csvReader.FieldsPerRecord = -1
    records, err := csvReader.ReadAll()
    if err != nil {
        return nil, err
    }
    if len(records) == 0 {
        return nil, nil
    }

    index := map[string]int{}
    for i, header := range records[0] {
        index[strings.TrimSpace(header)] = i
    }

    reports := make([]store.SalesReport, 0, len(records)-1)
    for _, row := range records[1:] {
        if len(strings.TrimSpace(strings.Join(row, ""))) == 0 {
            continue
        }
        reports = append(reports, store.SalesReport{
            Date:        column(row, index, "Begin Date", "Date"),
            Units:       parseInt(column(row, index, "Units")),
            Revenue:     parseFloat(column(row, index, "Developer Proceeds", "Proceeds", "Customer Price")),
            ProductID:   column(row, index, "SKU", "Product Type Identifier", "Apple Identifier"),
            CountryCode: column(row, index, "Country Code", "Country"),
            Currency:    column(row, index, "Currency of Proceeds", "Currency"),
        })
    }
    return reports, nil
}

func column(row []string, index map[string]int, names ...string) string {
    for _, name := range names {
        if i, ok := index[name]; ok && i < len(row) {
            return strings.TrimSpace(row[i])
        }
    }
    return ""
}

func parseInt(value string) int {
    value = strings.TrimSpace(value)
    if value == "" {
        return 0
    }
    n, _ := strconv.Atoi(value)
    return n
}

func parseFloat(value string) float64 {
    value = strings.TrimSpace(strings.ReplaceAll(value, ",", ""))
    if value == "" {
        return 0
    }
    n, _ := strconv.ParseFloat(value, 64)
    return n
}
