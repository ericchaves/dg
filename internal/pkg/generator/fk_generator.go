package generator

import (
	"fmt"

	"github.com/codingconcepts/dg/internal/pkg/model"
	"github.com/samber/lo"
)

type ForeignKeyGenerator struct {
	Table       string `yaml:"table"`
	ReferenceAs string `yaml:"reference_as"`
	Column      string `yaml:"column"`
	repeat      string `yaml:"repeat"`
}

func (g ForeignKeyGenerator) Generate(t model.Table, files map[string]model.CSVFile) error {
	cols := lo.Filter(t.Columns, func(c model.Column, _ int) bool {
		return c.Type == "fk"
	})
	for _, c := range cols {
		var fk ForeignKeyGenerator
		if err := c.Generator.UnmarshalFunc(&fk); err != nil {
			return fmt.Errorf("parsing fk process for %s.%s: %w", t.Name, c.Name, err)
		}
		if err := fk.generate(t, c, files); err != nil {
			return fmt.Errorf("generating fk columns: %w", err)
		}
	}

	return nil
}

func (g ForeignKeyGenerator) generate(t model.Table, col model.Column, files map[string]model.CSVFile) error {
	refAs, refTable, refColumn := g.ReferenceAs, g.Table, g.Column
	if refAs == "" {
		refAs = "parent"
	}
	refFile, ok := files[refTable]
	if !ok {
		return fmt.Errorf("referenced table %s not found", refTable)
	}

	refValues := refFile.GetColumnValues(refColumn)
	if len(refValues) == 0 {
		return fmt.Errorf("no values found in referenced column %q of table %q", refColumn, refTable)
	}

	var lines []string
	rows := 0
	for i, val := range refValues {
		repeat := 1
		if g.repeat != "" {
			ec := &ExprContext{Files: files}
			env := ec.makeEnvFromLine(t.Name, i)
			parent := refFile.GetRecord(i)
			env[refAs] = parent
			output, err := ec.evaluate(g.repeat, env)
			if err != nil {
				return err
			}
			repeat, ok = output.(int)
			if !ok {
				return fmt.Errorf("cannot cast value to int: %s", output)
			}
		}
		for j := 0; j < repeat; j++ {
			lines = append(lines, val)
		}
		rows += repeat
		if t.Count > 0 && rows >= t.Count {
			break
		}
	}

	AddTable(t, col.Name, lines, files)
	return nil
}
