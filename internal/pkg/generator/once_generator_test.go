package generator

import (
	"testing"

	"github.com/codingconcepts/dg/internal/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestOnceGenerator_Generate_Unique(t *testing.T) {
	sourceTable := model.CSVFile{
		Name:   "source",
		Header: []string{"id", "name", "value"},
		Lines: [][]string{
			{"1", "2", "3", "4"},
			{"Alice", "Bob", "Alice", "Bob"},
			{"A", "B", "C", "D"},
		},
	}

	targetTable := model.CSVFile{
		Name:   "target",
		Header: []string{"id", "ref"},
		Lines: [][]string{
			{"1", "2", "3"},
			{"Alice", "Bob", "Alice"},
		},
	}

	files := map[string]model.CSVFile{
		"source": sourceTable,
		"target": targetTable,
	}

	generator := OnceGenerator{
		SourceTable:  "source",
		SourceColumn: "name",
		SourceValue:  "value",
		MatchColumn:  "ref",
		Unique:       true,
	}

	table := model.Table{
		Name:    "target",
		Count:   3,
		Columns: []model.Column{{Name: "ref"}, {Name: "new_value"}},
	}

	column := model.Column{Name: "new_value"}

	err := generator.Generate(table, column, files)

	assert.NoError(t, err)

	generatedTable := files["target"]
	assert.Equal(t, []string{"id", "ref", "new_value"}, generatedTable.Header)
	assert.Equal(t, 3, len(generatedTable.Lines))
	assert.Equal(t, []string{"1", "2", "3"}, generatedTable.Lines[0])
	assert.Equal(t, []string{"Alice", "Bob", "Alice"}, generatedTable.Lines[1])
	assert.Equal(t, []string{"A", "B", "C"}, generatedTable.Lines[2])
}
func TestOnceGenerator_Generate_NonUnique(t *testing.T) {
	sourceTable := model.CSVFile{
		Name:   "source",
		Header: []string{"id", "name", "value"},
		Lines: [][]string{
			{"1", "2", "3"},
			{"Alice", "Bob", "Alice"},
			{"A", "B", "C"},
		},
	}

	targetTable := model.CSVFile{
		Name:   "target",
		Header: []string{"id", "ref"},
		Lines: [][]string{
			{"1", "2", "3", "4"},
			{"Alice", "Bob", "Alice", "Bob"},
		},
	}

	files := map[string]model.CSVFile{
		"source": sourceTable,
		"target": targetTable,
	}

	generator := OnceGenerator{
		SourceTable:  "source",
		SourceColumn: "name",
		SourceValue:  "value",
		MatchColumn:  "ref",
		Unique:       false,
	}

	table := model.Table{
		Name:    "target",
		Columns: []model.Column{{Name: "ref"}, {Name: "new_value"}},
		Count:   6,
	}

	column := model.Column{Name: "new_value"}

	err := generator.Generate(table, column, files)

	assert.NoError(t, err)

	generatedTable := files["target"]
	assert.Equal(t, []string{"A", "B", "C", "B", "A", "B"}, generatedTable.Lines[2])
}

func TestOnceGenerator_Generate_ErrorCases(t *testing.T) {
	generator := OnceGenerator{
		SourceTable:  "source",
		SourceColumn: "name",
		SourceValue:  "value",
		MatchColumn:  "ref",
	}

	t.Run("Source table not found", func(t *testing.T) {
		files := map[string]model.CSVFile{}
		err := generator.Generate(model.Table{}, model.Column{}, files)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "source table source not found")
	})

	t.Run("Source column not found", func(t *testing.T) {
		files := map[string]model.CSVFile{
			"source": {Header: []string{"id", "wrong_name", "value"}},
		}
		err := generator.Generate(model.Table{}, model.Column{}, files)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "source column or value column not found in table source")
	})

	t.Run("Missing destination table", func(t *testing.T) {
		files := map[string]model.CSVFile{
			"source": {Header: []string{"id", "name", "value"}},
		}
		table := model.Table{Name: "target", Columns: []model.Column{{Name: "wrong_ref"}}}
		err := generator.Generate(table, model.Column{}, files)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing destination table")
	})

	t.Run("Match column not found", func(t *testing.T) {
		files := map[string]model.CSVFile{
			"source": {Name: "source", Header: []string{"id", "name", "value"}},
			"target": {Name: "target", Header: []string{"id"}},
		}
		table := model.Table{Name: "target", Columns: []model.Column{{Name: "wrong_ref"}}}
		err := generator.Generate(table, model.Column{}, files)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing match column")
	})

}

func TestOnceGenerator_findUnusedValue(t *testing.T) {
	generator := OnceGenerator{}
	sourceFile := model.CSVFile{
		Lines: [][]string{
			{"1", "2", "3", "4"},
			{"Alice", "Bob", "Alice", "Bob"},
			{"A", "B", "C", "D"},
		},
	}

	t.Run("Find unused value", func(t *testing.T) {
		usedValues := map[string]bool{"A": true}
		value, err := generator.findUnusedValue(sourceFile, 1, 2, "Alice", usedValues)
		assert.NoError(t, err)
		assert.Equal(t, "C", value)
	})

	t.Run("No unused value found", func(t *testing.T) {
		usedValues := map[string]bool{"A": true, "B": true, "C": true}
		_, err := generator.findUnusedValue(sourceFile, 1, 2, "Alice", usedValues)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no unused value found for match value Alice")
	})

	t.Run("No match found", func(t *testing.T) {
		usedValues := map[string]bool{"A": true, "B": true, "C": true}
		_, err := generator.findUnusedValue(sourceFile, 1, 2, "David", usedValues)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no match found for David")
	})
}
