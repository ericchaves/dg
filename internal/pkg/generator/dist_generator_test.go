package generator

import (
	"testing"

	"github.com/codingconcepts/dg/internal/pkg/model"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestDistGenerator(t *testing.T) {
	tests := []struct {
		name           string
		values         []string
		weights        []int
		count          int
		expectedCounts map[string]int
	}{
		{
			name:    "Basic distribution",
			values:  []string{"dog", "cat", "bird"},
			weights: []int{7, 2, 1},
			count:   100,
			expectedCounts: map[string]int{
				"dog":  70,
				"cat":  20,
				"bird": 10,
			},
		},
		{
			name:    "Uneven weights",
			values:  []string{"A", "B", "C", "D"},
			weights: []int{40, 30, 20, 10},
			count:   1000,
			expectedCounts: map[string]int{
				"A": 400,
				"B": 300,
				"C": 200,
				"D": 100,
			},
		},
		{
			name:    "Missing weights",
			values:  []string{"X", "Y", "Z"},
			weights: []int{50, 50},
			count:   200,
			expectedCounts: map[string]int{
				"X": 100,
				"Y": 100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := &DistGenerator{
				Values:  tt.values,
				Weights: tt.weights,
			}

			table := model.Table{
				Name:  "test_table",
				Count: tt.count,
			}

			column := model.Column{
				Name: "test_column",
			}

			files := map[string]model.CSVFile{
				"test_table": {
					Name:   "test_table",
					Header: []string{"test_column"},
					Lines:  [][]string{},
				},
			}

			err := generator.Generate(table, column, files)
			assert.NoError(t, err)

			actualCounts := lo.CountValues(files["test_table"].Lines[0])
			assert.Equal(t, tt.expectedCounts, actualCounts)
		})
	}
}

func TestDistGeneratorWithExpression(t *testing.T) {
	t.Run("Simple array expression", func(t *testing.T) {
		generator := &DistGenerator{
			Expression: `["apple", "banana", "orange"]`,
		}

		table := model.Table{
			Name:  "test_table",
			Count: 90,
		}

		column := model.Column{
			Name: "test_column",
		}

		files := map[string]model.CSVFile{
			"test_table": {
				Name:   "test_table",
				Header: []string{"test_column"},
				Lines:  [][]string{},
			},
		}

		err := generator.Generate(table, column, files)
		assert.NoError(t, err)

		// Verify that the values from expression were added to g.Values
		assert.Equal(t, []string{"apple", "banana", "orange"}, generator.Values)

		// Verify the distribution (should be roughly equal since no weights were provided)
		actualCounts := lo.CountValues(files["test_table"].Lines[0])
		assert.Equal(t, 30, actualCounts["apple"])
		assert.Equal(t, 30, actualCounts["banana"])
		assert.Equal(t, 30, actualCounts["orange"])
	})

	t.Run("Array expression with repeated values", func(t *testing.T) {
		generator := &DistGenerator{
			Expression: `["apple", "banana", "apple", "orange", "apple", "banana"]`,
		}

		table := model.Table{
			Name:  "test_table",
			Count: 60,
		}

		column := model.Column{
			Name: "test_column",
		}

		files := map[string]model.CSVFile{
			"test_table": {
				Name:   "test_table",
				Header: []string{"test_column"},
				Lines:  [][]string{},
			},
		}

		err := generator.Generate(table, column, files)
		assert.NoError(t, err)

		// Verify that unique values were added to g.Values
		assert.ElementsMatch(t, []string{"apple", "banana", "orange"}, generator.Values)

		// Verify that counts were used as weights
		assert.ElementsMatch(t, []int{3, 2, 1}, generator.Weights)

		// Verify the distribution matches the weights
		actualCounts := lo.CountValues(files["test_table"].Lines[0])
		assert.Equal(t, 30, actualCounts["apple"])  // 3/6 * 60 = 30
		assert.Equal(t, 20, actualCounts["banana"]) // 2/6 * 60 = 20
		assert.Equal(t, 10, actualCounts["orange"]) // 1/6 * 60 = 10
	})

}

func TestDistGeneratorErrors(t *testing.T) {
	t.Run("Empty values and expression", func(t *testing.T) {
		generator := &DistGenerator{
			Values:  []string{},
			Weights: []int{1},
		}

		err := generator.Generate(model.Table{}, model.Column{}, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "either values or expression must be provided")
	})

	t.Run("Empty values but with expression", func(t *testing.T) {
		generator := &DistGenerator{
			Expression: `["A", "B"]`,
		}

		table := model.Table{
			Name:  "test_table",
			Count: 100,
		}

		column := model.Column{
			Name: "test_column",
		}

		files := map[string]model.CSVFile{
			"test_table": {
				Name:   "test_table",
				Header: []string{"test_column"},
				Lines:  [][]string{},
			},
		}

		err := generator.Generate(table, column, files)
		assert.NoError(t, err)

		actualCounts := lo.CountValues(files["test_table"].Lines[0])
		assert.Equal(t, 50, actualCounts["A"])
		assert.Equal(t, 50, actualCounts["B"])
	})

	t.Run("Zero total weight should use equal distribution", func(t *testing.T) {
		generator := &DistGenerator{
			Values:  []string{"A", "B"},
			Weights: []int{0, 0},
		}

		table := model.Table{
			Name:  "test_table",
			Count: 100,
		}

		column := model.Column{
			Name: "test_column",
		}

		files := map[string]model.CSVFile{
			"test_table": {
				Name:   "test_table",
				Header: []string{"test_column"},
				Lines:  [][]string{},
			},
		}

		err := generator.Generate(table, column, files)
		assert.NoError(t, err)

		actualCounts := lo.CountValues(files["test_table"].Lines[0])
		assert.Equal(t, 50, actualCounts["A"])
		assert.Equal(t, 50, actualCounts["B"])
	})
}
