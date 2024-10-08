package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnique(t *testing.T) {
	cases := []struct {
		name          string
		uniqueColumns []string
		exp           [][]string
	}{
		{
			name:          "1 column unique",
			uniqueColumns: []string{"col_1"},
			exp: [][]string{
				{"a", "d", "g"},
				{"b", "d", "g"},
				{"c", "d", "g"},
			},
		},
		{
			name:          "2 column unique",
			uniqueColumns: []string{"col_1", "col_2"},
			exp: [][]string{
				{"a", "d", "g"},
				{"b", "d", "g"},
				{"c", "d", "g"},
				{"a", "e", "g"},
				{"b", "e", "g"},
				{"c", "e", "g"},
				{"a", "f", "g"},
				{"b", "f", "g"},
				{"c", "f", "g"},
			},
		},
		{
			name:          "3 column unique",
			uniqueColumns: []string{"col_1", "col_2", "col_3"},
			exp: [][]string{
				{"a", "d", "g"},
				{"b", "d", "g"},
				{"c", "d", "g"},
				{"a", "e", "g"},
				{"b", "e", "g"},
				{"c", "e", "g"},
				{"a", "f", "g"},
				{"b", "f", "g"},
				{"c", "f", "g"},
				{"a", "d", "h"},
				{"b", "d", "h"},
				{"c", "d", "h"},
				{"a", "e", "h"},
				{"b", "e", "h"},
				{"c", "e", "h"},
				{"a", "f", "h"},
				{"b", "f", "h"},
				{"c", "f", "h"},
				{"a", "d", "i"},
				{"b", "d", "i"},
				{"c", "d", "i"},
				{"a", "e", "i"},
				{"b", "e", "i"},
				{"c", "e", "i"},
				{"a", "f", "i"},
				{"b", "f", "i"},
				{"c", "f", "i"},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			file := CSVFile{
				Header:        []string{"col_1", "col_2", "col_3"},
				UniqueColumns: c.uniqueColumns,
				Lines: [][]string{
					{"a", "d", "g"},
					{"b", "d", "g"},
					{"c", "d", "g"},
					{"a", "e", "g"},
					{"b", "e", "g"},
					{"c", "e", "g"},
					{"a", "f", "g"},
					{"b", "f", "g"},
					{"c", "f", "g"},
					{"a", "d", "h"},
					{"b", "d", "h"},
					{"c", "d", "h"},
					{"a", "e", "h"},
					{"b", "e", "h"},
					{"c", "e", "h"},
					{"a", "f", "h"},
					{"b", "f", "h"},
					{"c", "f", "h"},
					{"a", "d", "i"},
					{"b", "d", "i"},
					{"c", "d", "i"},
					{"a", "e", "i"},
					{"b", "e", "i"},
					{"c", "e", "i"},
					{"a", "f", "i"},
					{"b", "f", "i"},
					{"c", "f", "i"},
				},
			}

			act := file.Unique()

			assert.Equal(t, c.exp, act)
		})
	}
}

func TestGetLineValues(t *testing.T) {
	file := CSVFile{
		Header: []string{"col_1", "col_2", "col_3"},
		Lines: [][]string{
			{"a", "d", "g"},
			{"b", "e", "h"},
			{"c", "f", "i"},
		},
	}

	tests := []struct {
		name       string
		lineNumber int
		expected   map[string]any
	}{
		{"Valid line 0", 0, map[string]any{"col_1": "a", "col_2": "b", "col_3": "c", "row_number": 0, "rows_skipped": 0}},
		{"Valid line 1", 1, map[string]any{"col_1": "d", "col_2": "e", "col_3": "f", "row_number": 1, "rows_skipped": 0}},
		{"Valid line 2", 2, map[string]any{"col_1": "g", "col_2": "h", "col_3": "i", "row_number": 2, "rows_skipped": 0}},
		{"Invalid line negative", -1, map[string]any{}},
		{"Invalid line too high", 3, map[string]any{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := file.GetRecord(tt.lineNumber)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetColumnValues(t *testing.T) {
	file := CSVFile{
		Header: []string{"col_1", "col_2", "col_3", "col_4"},
		Lines: [][]string{
			{"a", "d", "g"},
			{"b", "e", "h"},
			{"c", "f", "i"},
			{},
		},
	}

	tests := []struct {
		name       string
		columnName string
		expected   []string
	}{
		{"Valid column col_1", "col_1", []string{"a", "d", "g"}},
		{"Valid column col_2", "col_2", []string{"b", "e", "h"}},
		{"Valid column col_3", "col_3", []string{"c", "f", "i"}},
		{"Empty column", "col_4", []string{}},
		{"Invalid column", "col_5", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := file.GetColumnValues(tt.columnName)
			assert.Equal(t, tt.expected, result)
		})
	}
}
