package generator

import (
	"fmt"

	"github.com/codingconcepts/dg/internal/pkg/model"
	"github.com/samber/lo"
)

type ForeignKeyGenerator struct {
	Table       string `yaml:"table"`
	ReferenceAs string `yaml:"reference_as"`
	SkippedAs   string `yaml:"skipped_as"`
	Column      string `yaml:"column"`
	Repeat      string `yaml:"repeat"`
	Filter      string `yaml:"filter"`
}

func (g ForeignKeyGenerator) Generate(t model.Table, files map[string]model.CSVFile) error {
	cols := lo.Filter(t.Columns, func(c model.Column, _ int) bool {
		return c.Type == "fk"
	})
	for _, c := range cols {
		var fkCol ForeignKeyGenerator
		if err := c.Generator.UnmarshalFunc(&fkCol); err != nil {
			return fmt.Errorf("parsing fk process for %s.%s: %w", t.Name, c.Name, err)
		}
		if err := fkCol.generate(t, c, files); err != nil {
			return fmt.Errorf("generating fk columns: %w", err)
		}
	}

	return nil
}

func (g ForeignKeyGenerator) generate(t model.Table, col model.Column, files map[string]model.CSVFile) error {
	skipAs, refAs, refTable, refColumn := g.SkippedAs, g.ReferenceAs, g.Table, g.Column
	if refAs == "" {
		refAs = "parent"
	}
	if skipAs == "" {
		skipAs = "skipped"
	}
	refFile, ok := files[refTable]
	if !ok {
		return fmt.Errorf("referenced table %s not found", refTable)
	}
	if lo.Contains(refFile.Header, refAs) {
		return fmt.Errorf("current table has a column named %s. use reference_as to set another variable name for the referenced table", refAs)
	}
	if lo.Contains(refFile.Header, skipAs) {
		return fmt.Errorf("current table has a column named %s. use skipped_as to set another variable name for the skipped count", skipAs)
	}

	refValues := refFile.GetColumnValues(refColumn)
	if len(refValues) == 0 {
		return fmt.Errorf("no values found in referenced column %q of table %q", refColumn, refTable)
	}

	var lines []string
	rows := 0
	skipped := 0
	for i, val := range refValues {
		repeat := 1
		ec := &ExprContext{Files: files}
		record := model.GetRecord(t.Name, i, files)
		env := ec.makeEnv()
		if err := ec.mergeEnv(env, record); err != nil {
			return err
		}
		parent := refFile.GetRecord(i)
		env[refAs] = parent

		if g.Filter != "" {
			env[skipAs] = skipped
			output, err := ec.evaluate(g.Filter, env)
			if err != nil {
				return err
			}
			skip := !ec.AnyToBool(output)
			if skip {
				skipped++
				continue
			}
		}

		if g.Repeat != "" {
			output, err := ec.evaluate(g.Repeat, env)
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
