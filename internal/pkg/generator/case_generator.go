package generator

import (
	"fmt"

	"github.com/codingconcepts/dg/internal/pkg/model"
	"github.com/samber/lo"
)

type CaseCondition struct {
	When string `yaml:"when"`
	Then string `yaml:"then"`
}

// ConstGenerator provides additional context to a case column.
type CaseGenerator []CaseCondition

// Generate values for a column based on a series of provided conditions.
func (g CaseGenerator) Generate(t model.Table, c model.Column, files map[string]model.CSVFile) error {
	if len(g) == 0 {
		return fmt.Errorf("no values provided for case generator")
	}
	if t.Count == 0 {
		t.Count = len(lo.MaxBy(files[t.Name].Lines, func(a, b []string) bool {
			return len(a) > len(b)
		}))
	}
	var lines []string
	for i := 0; i < t.Count; i++ {
		for _, cond := range g {
			ec := &ExprContext{Files: files, Format: ""}
			rec := model.GetRecord(t.Name, i, files)
			env := ec.makeEnv(rec)
			result, err := ec.evaluate(cond.When, env)
			if err != nil {
				return fmt.Errorf("error parsing When: %s (%w)", cond.When, err)
			}
			if result.(bool) {
				value, err := ec.evaluate(cond.Then, env)
				if err != nil {
					return fmt.Errorf("error evaluating Value: %s (%w)", cond.Then, err)
				}
				lines = append(lines, value.(string))
				break
			}
		}
	}
	AddTable(t, c.Name, lines, files)
	return nil
}
