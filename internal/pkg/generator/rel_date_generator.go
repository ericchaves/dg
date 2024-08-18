package generator

import (
	"fmt"
	"math/rand/v2"
	"regexp"
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

func findColumnIndex(table model.Table, columnName string) int {
	for i, column := range table.Columns {
		if column.Name == columnName {
			return i
		}
	}
	return -1
}

func resolveIntValue(source interface{}, t model.Table, cursor int, files map[string]model.CSVFile) (*int, error) {
	if val, ok := source.(int); ok {
		return &val, nil
	}
	if str, ok := source.(string); ok {
		ctx := &ExpressionContext{Files: files, Table: t, Format: ""}
		expression, err := ctx.NewEvaluableTableExpression(str)
		if err != nil {
			return nil, fmt.Errorf("error parsing expression: %w", err)
		}
		result, err := ctx.EvaluateTableExpression(expression, cursor)
		if err != nil {
			return nil, fmt.Errorf("error evaluating expression %w", err)
		}
		switch v := result.(type) {
		case int:
			return &v, nil
		case float32:
		case float64:
			i := int(v)
			return &i, nil
		default:
			return nil, fmt.Errorf("expression result %s is not of type int: %v", source, result)
		}
	}
	return nil, fmt.Errorf("invalid expression or column not found: %s", source)
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
		expression, err := ctx.NewEvaluableTableExpression(source)
		if err != nil {
			return nil, fmt.Errorf("error parsing expression: %w", err)
		}
		result, err := ctx.EvaluateTableExpression(expression, i)
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
		after, err := resolveIntValue(g.After, t, i, files)
		if err != nil {
			return err
		}
		before, err := resolveIntValue(g.Before, t, i, files)
		if err != nil {
			return err
		}
		s := g.generate(*reference, *before, *after)
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
