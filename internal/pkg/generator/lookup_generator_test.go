package generator

import (
	"testing"

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
