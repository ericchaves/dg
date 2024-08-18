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

	_after  int
	_before int
}

func findColumnIndex(t model.Table, name string) int {
	for i, column := range t.Columns {
		if column.Name == name {
			return i
		}
	}
	return -1
}

func resolveIntValue(value interface{}, t model.Table, i int, files map[string]model.CSVFile, fieldName string) (int, error) {
	if intValue, ok := value.(int); ok {
		return intValue, nil
	} else if col, ok := value.(string); ok {
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
		return 0, fmt.Errorf("error parsing %s value: %v", fieldName, value)
	}
}

func (g RelDateGenerator) Generate(t model.Table, c model.Column, files map[string]model.CSVFile) error {

	if g.Format == "" {
		g.Format = "2006-01-02"
	}
	ref_date := time.Now()
	ref_column := -1
	if g.Date != "" && g.Date != "now" {
		var err error
		matched, _ := regexp.MatchString(`^[a-zA-Z]\w+$`, g.Date)
		if matched {
			ref_column = findColumnIndex(t, g.Date)
		} else {
			ref_date, err = time.Parse(g.Format, g.Date)
			if err != nil {
				return fmt.Errorf("error parsing date: %w", err)
			}
		}
	}

	if g.Unit != day && g.Unit != month && g.Unit != year {
		g.Unit = day
	}

	if t.Count == 0 {
		t.Count = len(lo.MaxBy(files[t.Name].Lines, func(a, b []string) bool {
			return len(a) > len(b)
		}))
	}
	var err error
	var lines []string
	for i := 0; i < t.Count; i++ {
		if ref_column > -1 {
			ref_date, err = time.Parse(g.Format, files[t.Name].Lines[ref_column][i])
			if err != nil {
				return fmt.Errorf("error parsing date: %w", err)
			}
		}
		g._after, err = resolveIntValue(g.After, t, i, files, "After")
		if err != nil {
			return err
		}
		g._before, err = resolveIntValue(g.Before, t, i, files, "Before")
		if err != nil {
			return err
		}
		s := g.generate(ref_date)
		lines = append(lines, s)
	}
	AddTable(t, c.Name, lines, files)
	return nil
}

func (g RelDateGenerator) generate(reference time.Time) string {
	if g._after > g._before {
		g._after, g._before = g._before, g._after
	}
	offset := rand.IntN(g._before-g._after+1) + g._after
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
