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
	ec := &ExprContext{Files: files, Format: g.Format}
	columnValues := refFile.GetColumnValues(g.Column)
	countValues := lo.CountValues(columnValues)
	indexValues := make(map[string]int, len(countValues))
	var lines []string
	for row, value := range columnValues {
		if _, ok := indexValues[value]; !ok {
			indexValues[value] = 1
		}
		record := model.GetRecord(t.Name, row, files)
		env := ec.makeEnv()
		if err := ec.mergeEnv(env, record); err != nil {
			return err
		}
		env["index"] = indexValues[value]
		env["count"] = countValues[value]
		env["value"] = value
		result, err := ec.evaluate(g.Expression, env)
		if err != nil {
			return fmt.Errorf("error evaluating expression for row %d of value %s", row, value)
		}
		line := ec.AnyToString(result)
		lines = append(lines, line)
		indexValues[value] = indexValues[value] + 1
		if t.Count > 0 && row >= t.Count {
			break
		}
	}

	AddTable(t, col.Name, lines, files)
	return nil
}
