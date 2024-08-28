package generator

import (
	"fmt"
	"time"

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
		env := ec.makeEnvFromLine(t.Name, i)
		result, err := ec.evaluate(g.Expression, env)
		if err != nil {
			return fmt.Errorf("error evaluating expression %w", err)
		}
		switch getType(result) {
		case "time.Time":
			if g.Format == "" {
				g.Format = "2006-01-02"
			}
			lines = append(lines, result.(time.Time).Format(g.Format))
		case "float64":
			if g.Format == "" {
				g.Format = "%g"
			}
			lines = append(lines, fmt.Sprintf(g.Format, result.(float64)))
		case "int":
			if g.Format == "" {
				g.Format = "%d"
			}
			lines = append(lines, fmt.Sprintf(g.Format, result.(int)))
		case "bool":
			if g.Format == "" {
				g.Format = "%t"
			}
			lines = append(lines, fmt.Sprintf(g.Format, result.(bool)))
		case "string":
			lines = append(lines, result.(string))
		default:
			lines = append(lines, fmt.Sprintf("%v", result))
		}
	}
	AddTable(t, c.Name, lines, files)
	return nil
}
