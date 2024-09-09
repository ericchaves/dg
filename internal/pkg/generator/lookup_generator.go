package generator

import (
	"fmt"
	"regexp"

	"github.com/codingconcepts/dg/internal/pkg/model"
	"github.com/samber/lo"
)

type LookupTable struct {
	SourceTable  string `yaml:"source_table"`
	SourceColumn string `yaml:"source_column"`
	SourceValue  string `yaml:"source_value"`
	Format       string `yaml:"format"`
	Expression   string `yaml:"expression"`
	Predicate    string `yaml:"predicate"`
}

type LookupGenerator struct {
	MatchColumn   string        `yaml:"match_column"`
	LookupTables  []LookupTable `yaml:"tables"`
	IgnoreMissing bool          `yaml:"ignore_missing"`
	Repeat        string        `yaml:"repeat"`
}

func (g LookupGenerator) Generate(t model.Table, c model.Column, files map[string]model.CSVFile) error {
	if g.MatchColumn == "" {
		return fmt.Errorf("required match column missing")
	}
	baseTable, ok := files[t.Name]
	if !ok {
		return fmt.Errorf("base table %s not found", t.Name)
	}
	baseColumnIndex := lo.IndexOf(baseTable.Header, g.MatchColumn)
	if baseColumnIndex < 0 {
		return fmt.Errorf("match column %s not found in base table", g.MatchColumn)
	}
	if t.Count == 0 {
		t.Count = len(baseTable.Lines[baseColumnIndex])
	}

	if count := len(baseTable.Lines[baseColumnIndex]); t.Count > count {
		return fmt.Errorf("not enough values in base table: %d values, need %d", count, t.Count)
	}
	re := regexp.MustCompile(`value \S+ not found in column \S+`)
	var lines []string
	rows := 0
	for rows < t.Count {
		matchValue := baseTable.Lines[baseColumnIndex][rows]
		values, err := g.generate(matchValue, g.LookupTables, files)
		if err == nil || (re.MatchString(err.Error()) && g.IgnoreMissing) {
			if len(values) == 0 {
				values = []string{""}
			}
			lines = append(lines, values...)
			rows += len(values)
		} else {
			return err
		}
	}
	AddTable(t, c.Name, lines, files)
	return nil
}

func (g LookupGenerator) generate(matchValue string, lookupTables []LookupTable, files map[string]model.CSVFile) ([]string, error) {
	if matchValue == "" {
		return []string{}, fmt.Errorf("match_column is required")
	}
	ec := &ExprContext{Files: files}
	values := []string{}
	value := matchValue
	env := make(map[string]any)
	repeat := 1
	for _, lookup := range lookupTables {
		ec.Format = lookup.Format
		sourceFile, ok := files[lookup.SourceTable]
		if !ok {
			return []string{}, fmt.Errorf("lookup table %s not found", lookup.SourceTable)
		}
		if lookup.SourceColumn == "" {
			return []string{}, fmt.Errorf("source_column is required")
		}
		if lookup.SourceValue == "" && lookup.Expression == "" {
			return []string{}, fmt.Errorf("either source_value or expression is required")
		}
		record, err := ec.searchRecord(sourceFile, lookup.SourceColumn, value, lookup.Predicate)
		if err != nil {
			return []string{}, err
		}
		if lookup.Expression != "" {
			env = ec.makeEnv()
			if err := ec.mergeEnv(env, record); err != nil {
				return []string{}, err
			}
			anyValue, err := ec.evaluate(lookup.Expression, env)
			if err != nil {
				return []string{}, err
			}
			value = ec.AnyToString(anyValue)
		} else {
			value = ec.AnyToString(record[lookup.SourceValue])
		}
	}

	if g.Repeat != "" {
		output, err := ec.evaluate(g.Repeat, env)
		if err != nil {
			return []string{}, err
		}
		var ok bool
		repeat, ok = output.(int)
		if !ok {
			return []string{}, fmt.Errorf("cannot cast value to int: %s", output)
		}
	}
	for j := 0; j < repeat; j++ {
		values = append(values, value)
	}
	return values, nil
}
