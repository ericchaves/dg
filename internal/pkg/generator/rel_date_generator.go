package generator

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/codingconcepts/dg/internal/pkg/model"
	"github.com/samber/lo"
)

const (
	day   = "day"
	month = "month"
	year  = "year"
)

type RelDateGenerator struct {
	Date   string      `yaml:"date"`
	Unit   string      `yaml:"unit"`
	After  interface{} `yaml:"after"`
	Before interface{} `yaml:"before"`
	Format string      `yaml:"format"`
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
	if g.Before == nil || g.Before == "" {
		g.Before = 0
	}
	if g.After == nil || g.After == "" {
		g.After = 0
	}
	if g.Date == "" || g.Date == "now" {
		g.Date = "now()"
	}
	ec := &ExprContext{Files: files, Format: g.Format}
	var lines []string
	for i := 0; i < t.Count; i++ {
		rec := model.GetRecord(t.Name, i, files)
		env := ec.makeEnv(rec)

		reference, ok := model.ParseDate(g.Date, g.Format)
		if !ok {
			result, err := ec.evaluate(g.Date, env)
			if err != nil {
				return err
			}
			if reference, ok = result.(time.Time); !ok {
				return fmt.Errorf("date does not evaluate to valid time.Time: %s", g.Date)
			}
		}

		after, ok := g.After.(int)
		if !ok {
			if str, ok := g.After.(string); ok {
				afterVal, err := ec.evaluate(str, env)
				if err != nil {
					return err
				}
				if after, ok = afterVal.(int); !ok {
					return fmt.Errorf("after is not valid int: %s", g.After)
				}
			}
		}

		before, ok := g.Before.(int)
		if !ok {
			if str, ok := g.Before.(string); ok {
				beforeVal, err := ec.evaluate(str, env)
				if err != nil {
					return err
				}
				if before, ok = beforeVal.(int); !ok {
					return fmt.Errorf("before is not valid int: %s", g.Before)
				}
			}
		}

		s := g.generate(reference, before, after)
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
