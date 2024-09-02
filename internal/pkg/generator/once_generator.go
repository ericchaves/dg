package generator

import (
	"fmt"

	"github.com/codingconcepts/dg/internal/pkg/model"
	"github.com/samber/lo"
)

type OnceGenerator struct {
	SourceTable  string `yaml:"source_table"`
	SourceColumn string `yaml:"source_column"`
	SourceValue  string `yaml:"source_value"`
	MatchColumn  string `yaml:"match_column"`
	Unique       bool   `yaml:"unique"`
}

func (g OnceGenerator) Generate(t model.Table, col model.Column, files map[string]model.CSVFile) error {
	sourceFile, ok := files[g.SourceTable]
	if !ok {
		return fmt.Errorf("source table %s not found", g.SourceTable)
	}

	sourceColumnIndex := lo.IndexOf(sourceFile.Header, g.SourceColumn)
	sourceValueIndex := lo.IndexOf(sourceFile.Header, g.SourceValue)
	if sourceColumnIndex == -1 || sourceValueIndex == -1 {
		return fmt.Errorf("source column or value column not found in table %s", g.SourceTable)
	}
	matchTable, ok := files[t.Name]
	if !ok {
		return fmt.Errorf("missing destination table %q for match lookup", t.Name)
	}
	_, matchColumnIndex, ok := lo.FindIndexOf(matchTable.Header, func(c string) bool {
		return c == g.MatchColumn
	})
	if !ok {
		return fmt.Errorf("missing match column %q in current table", g.MatchColumn)
	}

	matchColumn := matchTable.Lines[matchColumnIndex]

	if t.Count == 0 {
		t.Count = len(matchColumn)
	}

	var lines []string
	usedValues := make(map[string]bool)
	lastIndexes := make(map[string]int)
	lastMatchIndex := 0
	for i := 0; i < t.Count; i++ {
		currentMatchIndex := (lastMatchIndex + i) % len(matchColumn)
		matchValue := matchColumn[currentMatchIndex]
		var value string
		var err error
		if g.Unique {
			value, err = g.findUnusedValue(sourceFile, sourceColumnIndex, sourceValueIndex, matchValue, usedValues)
		} else {
			lastIndex, ok := lastIndexes[matchValue]
			if !ok {
				lastIndex = 0
			}
			value, lastIndex, err = g.findNextValue(sourceFile, sourceColumnIndex, sourceValueIndex, matchValue, lastIndex)
			if err == nil {
				lastIndexes[matchValue] = lastIndex
			}
		}
		if err != nil {
			return err
		}
		lines = append(lines, value)
		usedValues[value] = true
	}

	AddTable(t, col.Name, lines, files)
	return nil
}

func (g OnceGenerator) findUnusedValue(sourceFile model.CSVFile, sourceColumnIndex, sourceValueIndex int, matchValue string, usedValues map[string]bool) (string, error) {
	matched := false
	for i, sourceValue := range sourceFile.Lines[sourceColumnIndex] {
		if sourceValue == matchValue {
			matched = true
			value := sourceFile.Lines[sourceValueIndex][i]
			if !usedValues[value] {
				return value, nil
			}
		}
	}
	if matched {
		return "", fmt.Errorf("no unused value found for match value %s", matchValue)
	}
	return "", fmt.Errorf("no match found for %s", matchValue)
}

func (g OnceGenerator) findNextValue(sourceFile model.CSVFile, sourceColumnIndex, sourceValueIndex int, matchValue string, lastIndex int) (string, int, error) {
	sourceColumnValues := sourceFile.Lines[sourceColumnIndex]
	sourceValueValues := sourceFile.Lines[sourceValueIndex]

	for i := 0; i < len(sourceColumnValues); i++ {
		currentIndex := (lastIndex + i) % len(sourceColumnValues)
		if sourceColumnValues[currentIndex] == matchValue {
			return sourceValueValues[currentIndex], currentIndex + 1, nil
		}
	}

	return "", lastIndex, fmt.Errorf("no match found for %s", matchValue)
}
