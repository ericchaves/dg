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
}

type LookupGenerator struct {
	MatchColumn   string        `yaml:"match_column"`
	LookupTables  []LookupTable `yaml:"tables"`
	IgnoreMissing bool          `yaml:"ignore_missing"`
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
	re := regexp.MustCompile(`value not found for \S+ in column \S+`)
	var lines []string
	for i := 0; i < t.Count; i++ {
		matchValue := baseTable.Lines[baseColumnIndex][i]
		value, err := g.generate(matchValue, g.LookupTables, files)
		if err != nil {
			if re.MatchString(err.Error()) && g.IgnoreMissing {
				lines = append(lines, value)
				continue
			}
			return err
		}
		lines = append(lines, value)
	}
	AddTable(t, c.Name, lines, files)
	return nil
}

func (g LookupGenerator) generate(matchValue string, lookupTables []LookupTable, files map[string]model.CSVFile) (string, error) {
	ec := &ExprContext{Files: files}
	value := matchValue
	var err error
	for _, lookup := range lookupTables {
		sourceFile, ok := files[lookup.SourceTable]
		if !ok {
			return "", fmt.Errorf("lookup table %s not found", lookup.SourceTable)
		}
		value, err = ec.searchFile(sourceFile, lookup.SourceColumn, value, lookup.SourceValue)
		if err != nil {
			return "", err
		}
	}
	return value, nil
}
