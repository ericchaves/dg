package generator

import (
	"testing"

	"github.com/codingconcepts/dg/internal/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestFKGenerator_Generate(t *testing.T) {
	tests := []struct {
		name           string
		fkGenerator    ForeignKeyGenerator
		table          model.Table
		column         model.Column
		files          map[string]model.CSVFile
		expectedError  string
		expectedResult map[string]model.CSVFile
	}{
		{
			name: "Valid FK generation",
			fkGenerator: ForeignKeyGenerator{
				Reference: "refTable",
				Column:    "refColumn",
			},
			table: model.Table{
				Name:  "testTable",
				Count: 3,
			},
			column: model.Column{
				Name: "fkColumn",
			},
			files: map[string]model.CSVFile{
				"refTable": {
					Header: []string{"refColumn"},
					Lines:  [][]string{{"1", "2", "3", "4"}},
				},
			},
			expectedError: "",
			expectedResult: map[string]model.CSVFile{
				"refTable": {
					Header: []string{"refColumn"},
					Lines:  [][]string{{"1", "2", "3", "4"}},
				},
				"testTable": {
					Name:   "testTable",
					Header: []string{"fkColumn"},
					Lines:  [][]string{{"1", "2", "3"}},
					Output: true,
				},
			},
		},
		{
			name: "Referenced table not found",
			fkGenerator: ForeignKeyGenerator{
				Reference: "nonExistentTable",
				Column:    "refColumn",
			},
			table: model.Table{
				Name:  "testTable",
				Count: 3,
			},
			column: model.Column{
				Name: "fkColumn",
			},
			files:          map[string]model.CSVFile{},
			expectedError:  "referenced table nonExistentTable not found",
			expectedResult: map[string]model.CSVFile{},
		},
		{
			name: "No values in referenced column",
			fkGenerator: ForeignKeyGenerator{
				Reference: "refTable",
				Column:    "emptyColumn",
			},
			table: model.Table{
				Name:  "testTable",
				Count: 3,
			},
			column: model.Column{
				Name: "fkColumn",
			},
			files: map[string]model.CSVFile{
				"refTable": {
					Header: []string{"emptyColumn"},
					Lines:  [][]string{},
				},
			},
			expectedError: "no values found in referenced column \"emptyColumn\" of table \"refTable\"",
			expectedResult: map[string]model.CSVFile{
				"refTable": {
					Header: []string{"emptyColumn"},
					Lines:  [][]string{},
				},
			},
		},
		{
			name: "FK generation with cardinality",
			fkGenerator: ForeignKeyGenerator{
				Reference: "orders",
				Column:    "order_id",
				repeat:    "parent.item_count",
			},
			table: model.Table{
				Name: "order_items",
			},
			column: model.Column{
				Name: "order_id",
			},
			files: map[string]model.CSVFile{
				"orders": {
					Header: []string{"order_id", "item_count"},
					Lines: [][]string{
						{"A", "B", "C"},
						{"2", "3", "1"},
					},
				},
			},
			expectedError: "",
			expectedResult: map[string]model.CSVFile{
				"orders": {
					Header: []string{"order_id", "item_count"},
					Lines: [][]string{
						{"A", "B", "C"},
						{"2", "3", "1"},
					},
				},
				"order_items": {
					Name:   "order_items",
					Header: []string{"order_id"},
					Lines: [][]string{
						{"A", "A", "B", "B", "B", "C"},
					},
					Output: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fkGenerator.generate(tt.table, tt.column, tt.files)

			if tt.expectedError != "" {
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, tt.files)
			}
		})
	}
}