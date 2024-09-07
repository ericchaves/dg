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
				Table:  "refTable",
				Column: "refColumn",
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
				Table:  "nonExistentTable",
				Column: "refColumn",
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
				Table:  "refTable",
				Column: "emptyColumn",
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
				Table:  "orders",
				Column: "order_id",
				Repeat: "int(parent.item_count)",
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
		{
			name: "FK generation with repeat expression using parent column",
			fkGenerator: ForeignKeyGenerator{
				Table:  "products",
				Column: "product_id",
				Repeat: "int(parent.quantity)",
			},
			table: model.Table{
				Name: "order_details",
			},
			column: model.Column{
				Name: "product_id",
			},
			files: map[string]model.CSVFile{
				"products": {
					Header: []string{"product_id", "quantity"},
					Lines: [][]string{
						{"P1", "P2", "P3"},
						{"2", "1", "3"},
					},
				},
			},
			expectedError: "",
			expectedResult: map[string]model.CSVFile{
				"products": {
					Header: []string{"product_id", "quantity"},
					Lines: [][]string{
						{"P1", "P2", "P3"},
						{"2", "1", "3"},
					},
				},
				"order_details": {
					Name:   "order_details",
					Header: []string{"product_id"},
					Lines: [][]string{
						{"P1", "P1", "P2", "P3", "P3", "P3"},
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
