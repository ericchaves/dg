package generator

import (
	"testing"

	"github.com/codingconcepts/dg/internal/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestOnceGenerator_Generate(t *testing.T) {
	// Setup
	generator := OnceGenerator{
		Table:  "source_table",
		Column: "unique_ids",
	}

	table := model.Table{Name: "test_table"}
	column := model.Column{
		Name: "test_column",
		Generator: model.RawMessage{
			UnmarshalFunc: func(v interface{}) error {
				*(v.(*OnceGenerator)) = generator
				return nil
			},
		},
	}

	files := map[string]model.CSVFile{
		"source_table": {
			Name:   "source_table",
			Header: []string{"unique_ids"},
			Lines:  [][]string{{"A", "B", "C", "D", "E"}},
		},
		"test_table": {
			Name:   "test_table",
			Header: []string{"test_column"},
			Lines:  [][]string{{}},
		},
	}

	// Execute
	err := generator.Generate(table, column, files)

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, files, "test_table")
	assert.Len(t, files["test_table"].Lines, 2) // Original column + new column

	newColumn := files["test_table"].Lines[1]
	assert.Len(t, newColumn, 5) // Should have 5 unique values

	// Check that all values from source are used exactly once
	usedValues := make(map[string]bool)
	for _, value := range newColumn {
		assert.False(t, usedValues[value], "Value %s was used more than once", value)
		usedValues[value] = true
	}
	assert.Len(t, usedValues, 5)
}

func TestOnceGenerator_Generate_InsufficientValues(t *testing.T) {
	// Setup
	generator := OnceGenerator{
		Table:  "source_table",
		Column: "unique_ids",
	}

	table := model.Table{Name: "test_table", Count: 10}
	column := model.Column{
		Name: "test_column",
		Generator: model.RawMessage{
			UnmarshalFunc: func(v interface{}) error {
				*(v.(*OnceGenerator)) = generator
				return nil
			},
		},
	}

	files := map[string]model.CSVFile{
		"source_table": {
			Name:   "source_table",
			Header: []string{"unique_ids"},
			Lines:  [][]string{{"A", "B", "C"}},
		},
		"test_table": {
			Name:   "test_table",
			Header: []string{"test_column"},
			Lines:  [][]string{{}},
		},
	}

	// Execute
	err := generator.Generate(table, column, files)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not enough unique values in the pool to generate")
}

func TestOnceGenerator_Generate_EmptySourceTable(t *testing.T) {
	// Setup
	generator := OnceGenerator{
		Table:  "source_table",
		Column: "unique_ids",
	}

	table := model.Table{Name: "test_table"}
	column := model.Column{
		Name: "test_column",
		Generator: model.RawMessage{
			UnmarshalFunc: func(v interface{}) error {
				*(v.(*OnceGenerator)) = generator
				return nil
			},
		},
	}

	files := map[string]model.CSVFile{
		"source_table": {
			Name:   "source_table",
			Header: []string{"unique_ids"},
			Lines:  [][]string{{}},
		},
		"test_table": {
			Name:   "test_table",
			Header: []string{"test_column"},
			Lines:  [][]string{{}},
		},
	}

	// Execute
	err := generator.Generate(table, column, files)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no values found in column")
}
