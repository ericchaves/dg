package model

import (
	"strconv"
	"time"

	"github.com/samber/lo"
)

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
	return refFile.GetRecord((lineNumber))
}

func (c *CSVFile) GetLineValues(lineNumber int) []string {
	if lineNumber < 0 || lineNumber >= len(c.Lines[0]) {
		return []string{}
	}
	var values []string
	for _, line := range c.Lines {
		if lineNumber < len(line) {
			values = append(values, line[lineNumber])
		}
	}
	return values
}

func (c *CSVFile) GetColumnValues(columnName string) []string {
	columnIndex := c.GetColumnIndex(columnName)
	if columnIndex == -1 {
		return []string{}
	}
	if len(c.Lines) < columnIndex+1 {
		return []string{}
	}
	return c.Lines[columnIndex]
}

func (c *CSVFile) GetColumnIndex(columnName string) int {
	columnIndex := -1
	for i, header := range c.Header {
		if header == columnName {
			columnIndex = i
			break
		}
	}
	return columnIndex
}

func (c *CSVFile) GetRecord(lineNumber int) map[string]any {
	record := make(map[string]any)
	if lineNumber < 0 || len(c.Lines) == 0 || lineNumber > len(c.Lines[0]) {
		return record
	}
	values := c.GetLineValues(lineNumber)
	for i, header := range c.Header {
		record[header] = CoerceType(values[i], "")
	}
	return record
}

func CoerceType(value string, dateFormat string) interface{} {
	// Try to convert to an integer
	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal
	}

	// Try to convert to a float64
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}

	// Try to convert to a boolean
	if boolVal, err := strconv.ParseBool(value); err == nil {
		return boolVal
	}

	// Try to convert to a time.Time
	if dateVal, ok := ParseDate(value, dateFormat); ok {
		return dateVal
	}

	// If all conversions fail, return the original string
	return value
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
