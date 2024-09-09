package generator

import (
	"testing"
	"time"

	"github.com/codingconcepts/dg/internal/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestLookupGenerator_Generate(t *testing.T) {
	files := map[string]model.CSVFile{
		"base": {
			Name:   "base",
			Header: []string{"id", "name"},
			Lines: [][]string{
				{"1", "2", "3"},
				{"Alice", "Bob", "Charlie"},
			},
		},
		"lookup1": {
			Name:   "lookup1",
			Header: []string{"name", "age"},
			Lines: [][]string{
				{"Alice", "Bob", "Charlie"},
				{"25", "30", "35"},
			},
		},
		"lookup2": {
			Name:   "lookup2",
			Header: []string{"age", "city"},
			Lines: [][]string{
				{"25", "30", "35"},
				{"New York", "London", "Paris"},
			},
		},
		"lookup3": {
			Name:   "lookup3",
			Header: []string{"city", "country"},
			Lines: [][]string{
				{"New York", "London", "Paris"},
				{"USA", "UK", "France"},
			},
		},
	}

	generator := LookupGenerator{
		MatchColumn: "name",
		LookupTables: []LookupTable{
			{SourceTable: "lookup1", SourceColumn: "name", SourceValue: "age"},
			{SourceTable: "lookup2", SourceColumn: "age", SourceValue: "city"},
			{SourceTable: "lookup3", SourceColumn: "city", SourceValue: "country"},
		},
	}

	table := model.Table{Name: "base", Count: 3}
	column := model.Column{Name: "country"}

	err := generator.Generate(table, column, files)

	assert.NoError(t, err)
	assert.Contains(t, files, "base")
	resultFile := files["base"]
	assert.Equal(t, "country", resultFile.Header[2])
	assert.Equal(t, []string{"USA", "UK", "France"}, resultFile.Lines[2])
}

func TestLookupGenerator_Generate_Errors(t *testing.T) {
	files := map[string]model.CSVFile{
		"base": {
			Name:   "base",
			Header: []string{"id"},
			Lines:  [][]string{{"1"}},
		},
	}

	tests := []struct {
		name      string
		generator LookupGenerator
		table     model.Table
		column    model.Column
		wantErr   string
	}{
		{
			name:      "Missing match column",
			generator: LookupGenerator{},
			table:     model.Table{Name: "base"},
			column:    model.Column{Name: "result"},
			wantErr:   "required match column missing",
		},
		{
			name:      "Base table not found",
			generator: LookupGenerator{MatchColumn: "id"},
			table:     model.Table{Name: "nonexistent"},
			column:    model.Column{Name: "result"},
			wantErr:   "base table nonexistent not found",
		},
		{
			name:      "Match column not found",
			generator: LookupGenerator{MatchColumn: "nonexistent"},
			table:     model.Table{Name: "base"},
			column:    model.Column{Name: "result"},
			wantErr:   "match column nonexistent not found in base table",
		},
		{
			name:      "Not enough values in base table",
			generator: LookupGenerator{MatchColumn: "id"},
			table:     model.Table{Name: "base", Count: 2},
			column:    model.Column{Name: "result"},
			wantErr:   "not enough values in base table: 1 values, need 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.generator.Generate(tt.table, tt.column, files)
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestLookupGenerator_Generate_IgnoreMissing(t *testing.T) {
	files := map[string]model.CSVFile{
		"base": {
			Name:   "base",
			Header: []string{"id", "name"},
			Lines: [][]string{
				{"1", "2", "3"},
				{"Alice", "Bob", "Charlie"},
			},
		},
		"lookup1": {
			Name:   "lookup1",
			Header: []string{"name", "age"},
			Lines: [][]string{
				{"Alice", "Charlie"},
				{"25", "35"},
			},
		},
		"lookup2": {
			Name:   "lookup2",
			Header: []string{"age", "city"},
			Lines: [][]string{
				{"25", "35"},
				{"New York", "Paris"},
			},
		},
	}

	generator := LookupGenerator{
		MatchColumn: "name",
		LookupTables: []LookupTable{
			{SourceTable: "lookup1", SourceColumn: "name", SourceValue: "age"},
			{SourceTable: "lookup2", SourceColumn: "age", SourceValue: "city"},
		},
		IgnoreMissing: true,
	}

	table := model.Table{Name: "base", Count: 3}
	column := model.Column{Name: "city"}

	err := generator.Generate(table, column, files)

	assert.NoError(t, err)
	assert.Contains(t, files, "base")
	resultFile := files["base"]
	assert.Equal(t, "city", resultFile.Header[2])
	assert.Equal(t, []string{"New York", "", "Paris"}, resultFile.Lines[2])
}

func TestExprContext_SearchRecord(t *testing.T) {
	files := map[string]model.CSVFile{
		"test": {
			Name:   "test",
			Header: []string{"id", "name", "age"},
			Lines: [][]string{
				{"1", "2", "3", "4"},
				{"Alice", "Bob", "Charlie", "Bob"},
				{"25", "30", "35", "35"},
			},
		},
	}

	ec := &ExprContext{Files: files}

	t.Run("Valid search without predicate", func(t *testing.T) {
		record, err := ec.searchRecord(files["test"], "name", "Bob", "")
		assert.NoError(t, err)
		assert.Equal(t, map[string]any{"id": "2", "name": "Bob", "age": "30", "row_number": 1, "rows_skipped": 0}, record)
	})

	t.Run("Invalid source column", func(t *testing.T) {
		_, err := ec.searchRecord(files["test"], "invalid", "Bob", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "column not found")
	})

	t.Run("Value not found", func(t *testing.T) {
		_, err := ec.searchRecord(files["test"], "name", "David", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "value David not found")
	})

	t.Run("With Predicate", func(t *testing.T) {
		record, err := ec.searchRecord(files["test"], "name", "Bob", "int(age) > 30")
		assert.NoError(t, err)
		assert.Equal(t, map[string]any{"id": "4", "name": "Bob", "age": "35", "row_number": 3, "rows_skipped": 1}, record)
	})

	t.Run("Predicate not satisfied", func(t *testing.T) {
		_, err := ec.searchRecord(files["test"], "name", "Bob", "int(age) > 35")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "value Bob not found")
	})
}

func TestExprContext_Evaluate(t *testing.T) {
	ec := &ExprContext{}

	t.Run("Simple arithmetic", func(t *testing.T) {
		result, err := ec.evaluate("2 + 2", nil)
		assert.NoError(t, err)
		assert.Equal(t, 4, result)
	})

	t.Run("With environment variables", func(t *testing.T) {
		env := map[string]any{"x": 5, "y": 3}
		result, err := ec.evaluate("x * y", env)
		assert.NoError(t, err)
		assert.Equal(t, 15, result)
	})

	t.Run("Invalid expression", func(t *testing.T) {
		_, err := ec.evaluate("2 +", nil)
		assert.Error(t, err)
	})

	t.Run("Function call", func(t *testing.T) {
		env := map[string]any{
			"double": func(x int) int { return x * 2 },
		}
		result, err := ec.evaluate("double(5)", env)
		assert.NoError(t, err)
		assert.Equal(t, 10, result)
	})
}

func TestExprContext_AnyToString(t *testing.T) {
	ec := &ExprContext{}

	t.Run("Integer", func(t *testing.T) {
		ec.Format = ""
		result := ec.AnyToString(42)
		assert.Equal(t, "42", result)
	})

	t.Run("Float", func(t *testing.T) {
		ec.Format = ""
		result := ec.AnyToString(3.14)
		assert.Equal(t, "3.14", result)
	})

	t.Run("String", func(t *testing.T) {
		ec.Format = ""
		result := ec.AnyToString("hello")
		assert.Equal(t, "hello", result)
	})

	t.Run("Boolean", func(t *testing.T) {
		ec.Format = ""
		result := ec.AnyToString(true)
		assert.Equal(t, "true", result)
	})

	t.Run("Time", func(t *testing.T) {
		ec.Format = "2006-01-02"
		time := time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC)
		result := ec.AnyToString(time)
		assert.Equal(t, "2023-05-15", result)
	})
}

func TestExprContext_AnyToBool(t *testing.T) {
	ec := &ExprContext{}

	t.Run("Boolean true", func(t *testing.T) {
		assert.True(t, ec.AnyToBool(true))
	})

	t.Run("Boolean false", func(t *testing.T) {
		assert.False(t, ec.AnyToBool(false))
	})

	t.Run("Integer non-zero", func(t *testing.T) {
		assert.True(t, ec.AnyToBool(1))
	})

	t.Run("Integer zero", func(t *testing.T) {
		assert.False(t, ec.AnyToBool(0))
	})

	t.Run("String non-empty", func(t *testing.T) {
		assert.True(t, ec.AnyToBool("hello"))
	})

	t.Run("String empty", func(t *testing.T) {
		assert.False(t, ec.AnyToBool(""))
	})

	t.Run("Nil", func(t *testing.T) {
		assert.False(t, ec.AnyToBool(nil))
	})

	t.Run("Slice non-empty", func(t *testing.T) {
		assert.True(t, ec.AnyToBool([]interface{}{1, 2, 3}))
	})

	t.Run("Slice empty", func(t *testing.T) {
		assert.False(t, ec.AnyToBool([]interface{}{}))
	})

	t.Run("Map non-empty", func(t *testing.T) {
		assert.True(t, ec.AnyToBool(map[interface{}]interface{}{"key": "value"}))
	})

	t.Run("Map empty", func(t *testing.T) {
		assert.False(t, ec.AnyToBool(map[interface{}]interface{}{}))
	})
}
