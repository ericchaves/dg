package generator

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"strconv"
	"time"

	"github.com/codingconcepts/dg/internal/pkg/model"
	"github.com/samber/lo"
)

const (
	day   = "Day"
	month = "Month"
	year  = "Year"
)

type RelDateGenerator struct {
	Date   string      `yaml:"date"`
	Unit   string      `yaml:"unit"`
	After  interface{} `yaml:"after"`
	Before interface{} `yaml:"before"`
	Format string      `yaml:"format"`
}

func findColumnIndex(t model.Table, name string) int {
	for i, column := range t.Columns {
		if column.Name == name {
			return i
		}
	}
	return -1
}

func resolveIntValue(source interface{}, t model.Table, i int, files map[string]model.CSVFile, fieldName string) (int, error) {
	if intValue, ok := source.(int); ok {
		return intValue, nil
	} else if col, ok := source.(string); ok {
		idx := findColumnIndex(t, col)
		if idx == -1 {
			return 0, fmt.Errorf("%s column not found: %s", fieldName, col)
		}
		val, err := strconv.Atoi(files[t.Name].Lines[idx][i])
		if err != nil {
			return 0, fmt.Errorf("error parsing %s column value: %w", fieldName, err)
		}
		return val, nil
	} else {
		return 0, fmt.Errorf("error parsing %s value: %v", fieldName, source)
	}
}

func resolveDateValue(source string, format string, t model.Table, i int, files map[string]model.CSVFile) (*time.Time, error) {
	var err error
	result := time.Now()
	if source == "" || source == "now" {
		return &result, nil
	}
	matched, _ := regexp.MatchString(`^match`, source)
	if matched {
		// match value from other table
		ctx := &ExpressionContext{Files: files, Table: t, Format: format}
		expression, err := ctx.NewEvaluableExpression(source)
		if err != nil {
			return nil, fmt.Errorf("error parsing expression: %w", err)
		}
		result, err := ctx.EvaluateExpression(expression, i)
		if err != nil {
			return nil, fmt.Errorf("error evaluating expression %w", err)
		}
		if getType(result) == "time.Time" {
			return result.(*time.Time), nil
		} else {
			return nil, fmt.Errorf("expression %s is not a date: %v", source, result)
		}
	}

	matched, _ = regexp.MatchString(`^[a-zA-Z]\w+$`, source)
	if matched {
		ref_column := findColumnIndex(t, source)
		if ref_column == -1 {
			return nil, fmt.Errorf("column not found: %s", source)
		}
		source = files[t.Name].Lines[ref_column][i]
	}
	result, err = time.Parse(format, source)
	if err != nil {
		return nil, fmt.Errorf("error parsing date: %w", err)
	}
	return &result, nil
}

func (g RelDateGenerator) Generate(t model.Table, c model.Column, files map[string]model.CSVFile) error {

	if g.Format == "" {
		g.Format = "2006-01-02"
	}

	if g.Unit != day && g.Unit != month && g.Unit != year {
		g.Unit = day
	}

	if t.Count == 0 {
		t.Count = len(lo.MaxBy(files[t.Name].Lines, func(a, b []string) bool {
			return len(a) > len(b)
		}))
	}
	var lines []string
	for i := 0; i < t.Count; i++ {
		reference, err := resolveDateValue(g.Date, g.Format, t, i, files)
		if err != nil {
			return err
		}
		after, err := resolveIntValue(g.After, t, i, files, "After")
		if err != nil {
			return err
		}
		before, err := resolveIntValue(g.Before, t, i, files, "Before")
		if err != nil {
			return err
		}
		s := g.generate(*reference, before, after)
		lines = append(lines, s)
	}
	AddTable(t, c.Name, lines, files)
	return nil
}

func (g RelDateGenerator) generate(reference time.Time, before int, after int) string {
	if after > before {
		after, before = before, after
	}
	offset := rand.IntN(before-after+1) + after
	switch g.Unit {
	case day:
		return reference.AddDate(0, 0, offset).Format(g.Format)
	case month:
		return reference.AddDate(0, offset, 0).Format(g.Format)
	case year:
		return reference.AddDate(offset, 0, 0).Format(g.Format)
	}
	return fmt.Errorf("invalid unit %s. unit must be 'day', 'month' or 'year'", g.Unit).Error()
}
