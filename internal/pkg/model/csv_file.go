package model

import (
	"time"

	"github.com/samber/lo"
)

const ROW_NUMBER = "row_number"
const ROWS_SKIPPED = "rows_skipped"
const ROW_VALUE = "value"
const VALUE_INDEX = "index"
const VALUE_COUNT = "count"

// CSVFile represents the content of a CSV file.
type CSVFile struct {
	Name          string
	Header        []string
	Lines         [][]string
	UniqueColumns []string
	Output        bool
}

// Unique removes any duplicates from the CSVFile's lines.
func (c *CSVFile) Unique() [][]string {
	uniqueColumnIndexes := uniqueIndexes(c.Header, c.UniqueColumns)

	uniqueValues := map[string]struct{}{}
	var uniqueLines [][]string

	for i := 0; i < len(c.Lines); i++ {
		key := uniqueKey(uniqueColumnIndexes, c.Lines[i])

		if _, ok := uniqueValues[key]; !ok {
			uniqueLines = append(uniqueLines, c.Lines[i])
			uniqueValues[key] = struct{}{}
		}
	}

	return uniqueLines
}

func uniqueIndexes(header, uniqueColumns []string) []int {
	indexes := []int{}

	for i, h := range header {
		if lo.Contains(uniqueColumns, h) {
			indexes = append(indexes, i)
		}
	}

	return indexes
}

func uniqueKey(indexes []int, line []string) string {
	output := ""

	for i, col := range line {
		if lo.Contains(indexes, i) {
			output += col
		} else {
			output += "-"
		}
	}

	return output
}

func GetRecord(table string, lineNumber int, files map[string]CSVFile) map[string]any {
	refFile, ok := files[table]
	if !ok {
		return map[string]any{}
	}
	return refFile.GetRecord(lineNumber)
}

func GetColumnValues(table string, columnName string, files map[string]CSVFile) []string {
	refFile, ok := files[table]
	if !ok {
		return []string{}
	}
	return refFile.GetColumnValues(columnName)
}

func (c *CSVFile) GetColumnValues(columnName string) []string {
	columnIndex := -1
	for i, header := range c.Header {
		if header == columnName {
			columnIndex = i
			break
		}
	}
	if columnIndex == -1 || columnIndex >= len(c.Lines) {
		return []string{}
	}
	return c.Lines[columnIndex]
}

func (c *CSVFile) GetRecord(lineNumber int) map[string]any {
	record := make(map[string]any)

	if lineNumber < 0 || len(c.Lines) == 0 {
		return map[string]any{}
	}

	empty := true
	for i, header := range c.Header {
		if i < len(c.Lines) && lineNumber < len(c.Lines[i]) {
			record[header] = c.Lines[i][lineNumber]
			empty = false
		} else {
			record[header] = nil
		}
	}
	record[ROW_NUMBER] = lineNumber
	record[ROWS_SKIPPED] = 0
	if empty {
		return map[string]any{}
	}
	return record
}

func ParseDate(value string, dateFormat string) (time.Time, bool) {
	// Try to convert to a time.Time
	// Define a list of date formats to try
	dateFormats := []string{
		time.RFC3339,
		"2006/01/02",                // ISO date variation (YYYY/MM/DD)
		"2006-01-02",                // ISO date format
		"02/01/2006",                // European date format (DD/MM/YYYY)
		"01/02/2006",                // US date format (MM/DD/YYYY)
		"2006-01-02 15:04:05",       // ISO datetime format without timezone
		"2006/01/02T15:04:05Z07:00", // ISO datetime variation
	}
	if dateFormat != "" {
		dateFormats = append([]string{dateFormat}, dateFormats...)
	}

	for _, format := range dateFormats {
		if dateVal, err := time.Parse(format, value); err == nil {
			return dateVal, true
		}
	}
	return time.Now(), false
}
