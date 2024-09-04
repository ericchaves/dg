package generator

import (
	"fmt"
	"reflect"

	"github.com/codingconcepts/dg/internal/pkg/model"
	"github.com/samber/lo"
)

type ExprGenerator struct {
	Expression string `yaml:"expression"`
	Format     string `yaml:"format"`
}

func (g ExprGenerator) Generate(t model.Table, c model.Column, files map[string]model.CSVFile) error {
	if g.Expression == "" {
		return fmt.Errorf("expression cannot be empty")
	}

	if t.Count == 0 {
		t.Count = len(lo.MaxBy(files[t.Name].Lines, func(a, b []string) bool {
			return len(a) > len(b)
		}))
	}
	ec := &ExprContext{Files: files, Format: g.Format}
	var lines []string
	for i := 0; i < t.Count; i++ {
		if len(lines) == t.Count {
			break
		}
		env := ec.makeEnvFromLine(t.Name, i)
		result, err := ec.evaluate(g.Expression, env)
		if err != nil {
			return fmt.Errorf("error evaluating expression %w", err)
		}
		items := reflect.ValueOf(result)
		if items.Kind() == reflect.Array || items.Kind() == reflect.Slice {
			for j := 0; j < items.Len(); j++ {
				item := items.Index(j)
				line := ec.AnyToString(item.Interface())
				lines = append(lines, line)
				if len(lines) == t.Count {
					break
				}
			}
		} else {
			line := ec.AnyToString(result)
			lines = append(lines, line)
		}
	}
	AddTable(t, c.Name, lines, files)
	return nil
}
