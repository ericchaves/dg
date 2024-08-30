package generator

import (
	"testing"

	"github.com/codingconcepts/dg/internal/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestCountValuesGenerator_Generate(t *testing.T) {
	// Setup
	generator := CountValuesGenerator{
		Table:      "test_table",
		Column:     "test_column",
		Expression: "string(ITN) + ':' + string(VALUE) + ':' + string(COUNT)",
	}

	table := model.Table{Name: "test_table"}
	column := model.Column{
		Name: "test_column",
		Generator: model.RawMessage{
			UnmarshalFunc: func(v interface{}) error {
				*(v.(*CountValuesGenerator)) = generator
				return nil
			},
		},
	}

	files := map[string]model.CSVFile{
		"test_table": {
			Name:   "test_table",
			Header: []string{"test_column"},
			Lines:  [][]string{{"A", "B", "A", "C", "B", "A"}},
		},
	}

	// Execute
	err := generator.Generate(table, column, files)

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, files, "test_table")
	assert.Len(t, files["test_table"].Lines, 2) // Original column + new column

	newColumn := files["test_table"].Lines[1]
	assert.Len(t, newColumn, 6) // 3 unique values, each repeated its count times

	expectedValues := []string{"1:A:3", "2:A:3", "3:A:3", "1:B:2", "2:B:2", "1:C:1"}
	assert.ElementsMatch(t, expectedValues, newColumn)
}

func TestCountValuesGenerator_Generate_EmptyTable(t *testing.T) {
	// Setup
	generator := CountValuesGenerator{
		Table:      "empty_table",
		Column:     "test_column",
		Expression: "{VALUE}:{COUNT}",
	}

	table := model.Table{Name: "empty_table"}
	column := model.Column{
		Name: "test_column",
		Generator: model.RawMessage{
			UnmarshalFunc: func(v interface{}) error {
				*(v.(*CountValuesGenerator)) = generator
				return nil
			},
		},
	}

	files := map[string]model.CSVFile{
		"empty_table": {
			Name:   "empty_table",
			Header: []string{"test_column"},
			Lines:  [][]string{{}},
		},
	}

	// Execute
	err := generator.Generate(table, column, files)

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, files, "empty_table")
	assert.Len(t, files["empty_table"].Lines, 2)   // Original column + new column
	assert.Empty(t, files["empty_table"].Lines[1]) // New column should be empty
}
