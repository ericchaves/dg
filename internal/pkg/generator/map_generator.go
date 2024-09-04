package generator

import (
	"fmt"

	"github.com/codingconcepts/dg/internal/pkg/model"
	"github.com/samber/lo"
)

type MapGenerator struct {
	Table      string `yaml:"table"`
	Column     string `yaml:"column"`
	Expression string `yaml:"expression"`
	Format     string `yaml:"format"`
}

func (g MapGenerator) Generate(t model.Table, col model.Column, files map[string]model.CSVFile) error {
	if g.Table == "" {
		g.Table = t.Name
	}
	refFile, ok := files[g.Table]
	if !ok {
		return fmt.Errorf("referenced table %s not found", g.Table)
	}
	ec := &ExprContext{Files: files}
	columnValues := refFile.GetColumnValues(g.Column)
	countValues := lo.CountValues(columnValues)
	var lines []string
	j := 0
	for value, count := range countValues {
		for i := 1; i <= count; i++ {
			env := ec.makeEnvFromLine(t.Name, j)
			env["index"] = i
			env["count"] = count
			env["value"] = value
			result, err := ec.evaluate(g.Expression, env)
			if err != nil {
				return fmt.Errorf("error evaluating expression for count %d of value %s", count, value)
			}
			line := ec.AnyToString(result)
			lines = append(lines, line)
		}
	}

	AddTable(t, col.Name, lines, files)
	return nil
}
