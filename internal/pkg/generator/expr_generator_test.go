package generator

import (
	"testing"

	"github.com/codingconcepts/dg/internal/pkg/model"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestGeneratorExpMatchColumn(t *testing.T) {

	table := model.Table{
		Name:  "table",
		Count: 1,
	}

	column := model.Column{
		Name: "column",
	}

	files := map[string]model.CSVFile{
		"products": {
			Name:   "products",
			Header: []string{"product_id", "product_name", "product_price"},
			Lines: [][]string{
				{"1", "2", "3"},
				{"Apple", "bananas", "carrots"},
				{"3.00", "5.00", "2.0"},
			},
		},
		"table": {
			Name:   "table",
			Header: []string{"id"},
			Lines: [][]string{
				{"2", "1", "3"},
			},
		},
	}
	g := ExprGenerator{
		Expression: "float(match('products','product_id', id,'product_price')) / 2.0",
	}
	err := g.Generate(table, column, files)
	assert.Nil(t, err)
	assert.Equal(t, files["table"].Lines[1][0], "2.5")
}

func TestGeneratorExprColumnValues(t *testing.T) {

	table := model.Table{
		Name:  "table",
		Count: 3,
	}

	column := model.Column{
		Name: "column",
	}

	files := map[string]model.CSVFile{
		"table": {
			Name:   "table",
			Header: []string{"name", "rate", "months"},
			Lines: [][]string{
				{"jhon", "jack", "joe"},
				{"3.00", "5.00", "2.0"},
				{"2", "3", "5"},
			},
		},
	}
	g := ExprGenerator{
		Expression: "float(rate) * int(months)",
		Format:     "%.4f",
	}
	err := g.Generate(table, column, files)
	assert.Nil(t, err)
	assert.Equal(t, files["table"].Lines[3][0], "6.0000")
}

func TestGeneratorExprDateFunctions(t *testing.T) {

	table := model.Table{
		Name:  "table",
		Count: 1,
	}

	column := model.Column{
		Name: "column",
	}

	files := map[string]model.CSVFile{
		"table": {
			Name:   "table",
			Header: []string{"name", "rate", "months"},
			Lines: [][]string{
				{"jhon", "jack", "joe"},
				{"3.00", "5.00", "2.0"},
				{"2", "3", "5"},
			},
		},
	}
	g := ExprGenerator{
		Expression: "add_date(1, 1, 1, '2024-12-25')",
	}
	err := g.Generate(table, column, files)
	assert.Nil(t, err)
	assert.Equal(t, files["table"].Lines[3][0], "2026-01-26")
}

func TestGeneratorExprDateFunctionsFormatted(t *testing.T) {

	table := model.Table{
		Name:  "table",
		Count: 3,
	}

	column := model.Column{
		Name: "column",
	}

	files := map[string]model.CSVFile{
		"table": {
			Name:   "table",
			Header: []string{"name", "rate", "months"},
			Lines: [][]string{
				{"jhon", "jack", "joe"},
				{"3.00", "5.00", "2.0"},
				{"2", "3", "5"},
			},
		},
	}
	g := ExprGenerator{
		Expression: "add_date(1, 1, 1, '2024/12/25')",
		Format:     "2006/01/02",
	}
	err := g.Generate(table, column, files)
	assert.Nil(t, err)
	assert.Equal(t, files["table"].Lines[3][0], "2026/01/26")
}

func TestGeneratorExprRandFunctions(t *testing.T) {
	cases := []struct {
		name       string
		expression string
	}{
		{
			name:       "rand",
			expression: "rand()",
		},
		{
			name:       "rand with max value",
			expression: "rand(10)",
		},
		{
			name:       "rand with negative number",
			expression: "randn(-10)",
		},
		{
			name:       "rand_range",
			expression: "rand_range(-10, int(parameter))",
		},
		{
			name:       "rand_range with all negatives",
			expression: "rand_range(-30, int(parameter) * -1)",
		},
		{
			name:       "rand with value from other cell",
			expression: "rand(int(parameter))",
		},
		{
			name:       "rand_float64",
			expression: "rand_float64()",
		},
		{
			name:       "rand_perm",
			expression: "rand_perm(int(parameter))",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			table := model.Table{
				Name:    "table",
				Columns: []model.Column{{Name: "id"}, {Name: "case"}, {Name: "parameter"}},
				Count:   1,
			}
			column := model.Column{
				Name: "random_value",
			}
			files := map[string]model.CSVFile{
				"table": {
					Name:   "table",
					Header: []string{"id", "parameter"},
					Lines: [][]string{
						{"0"},
						{"10"},
					},
				},
			}
			g := ExprGenerator{
				Expression: c.expression,
			}
			err := g.Generate(table, column, files)
			assert.Nil(t, err)
			last_line, ok := lo.Last(files["table"].Lines)
			assert.True(t, ok)
			last_value, ok := lo.Last(last_line)
			assert.True(t, ok)
			assert.NotEmpty(t, last_value)
		})
	}
}

func TestGeneratorExprMinMaxFunctions(t *testing.T) {
	cases := []struct {
		name       string
		expression string
		expected   string
	}{
		{
			name:       "min integers",
			expression: "min(3,5,0)",
			expected:   "0",
		},
		{
			name:       "min with float",
			expression: "min(5,0.3,2, 21.2345)",
			expected:   "0.3",
		},
		{
			name:       "min with negatives",
			expression: "min(5,-3,0.3,7,-0.5)",
			expected:   "-3",
		},
		{
			name:       "min with references",
			expression: "min(5,-3,0.3,7,-0.5, int(parameter))",
			expected:   "-3",
		},
		{
			name:       "max integers",
			expression: "max(3,5,0)",
			expected:   "5",
		},
		{
			name:       "max with float",
			expression: "max(5,0.3,2, 21.2345)",
			expected:   "21.2345",
		},
		{
			name:       "max with negatives",
			expression: "max(5,-3,0.3,7,-0.5)",
			expected:   "7",
		},
		{
			name:       "max with references",
			expression: "max(5,-3,0.3,7,-0.5, int(parameter))",
			expected:   "10",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			table := model.Table{
				Name:    "table",
				Columns: []model.Column{{Name: "id"}, {Name: "case"}, {Name: "parameter"}},
				Count:   1,
			}
			column := model.Column{
				Name: "random_value",
			}
			files := map[string]model.CSVFile{
				"table": {
					Name:   "table",
					Header: []string{"id", "parameter"},
					Lines: [][]string{
						{"0"},
						{"10"},
					},
				},
			}
			g := ExprGenerator{
				Expression: c.expression,
			}
			err := g.Generate(table, column, files)
			assert.Nil(t, err)
			last_line, ok := lo.Last(files["table"].Lines)
			assert.True(t, ok)
			last_value, ok := lo.Last(last_line)
			assert.True(t, ok)
			assert.Equal(t, last_value, c.expected)
		})
	}
}

func TestGeneratorExprMatchFunction(t *testing.T) {
	cases := []struct {
		name       string
		expression string
		format     string
		expected   []string
	}{
		{
			name:       "match integer field",
			expression: "match('contracts','id', row_id , 'months')",
			expected:   []string{"2", "3", "5"},
		},
		{
			name:       "match integer field using LN cursor",
			expression: "match('contracts','id', LN + 1, 'months')",
			expected:   []string{"2", "3", "5"},
		},
		{
			name:       "match with format float64 using LN cursor",
			expression: "int(match('contracts','id', LN + 1, 'months')) + 0.000001",
			format:     "%.2f",
			expected:   []string{"2.00", "3.00", "5.00"},
		},
		{
			name:       "match with date column",
			expression: "date(match('contracts','id', LN + 1, 'enroll'), '2006-01-02')",
			format:     "02/01/2006",
			expected:   []string{"14/08/2023", "15/08/2023", "16/08/2023"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {

			table := model.Table{
				Name:  "table",
				Count: 3,
			}

			column := model.Column{
				Name: "column",
			}

			files := map[string]model.CSVFile{
				"table": {
					Name:   "table",
					Header: []string{"row_id"},
					Lines: [][]string{
						{"1", "2", "3"},
					},
				},
				"contracts": {
					Name:   "contracts",
					Header: []string{"id", "name", "enroll", "months"},
					Lines: [][]string{
						{"1", "2", "3"},
						{"jhon", "jack", "joe"},
						{"2023-08-14", "2023-08-15", "2023-08-16"},
						{"2", "3", "5"},
					},
				},
			}

			g := ExprGenerator{
				Expression: c.expression,
				Format:     c.format,
			}
			err := g.Generate(table, column, files)
			assert.Nil(t, err)
			assert.Equal(t, files["table"].Lines[1], c.expected)
		})
	}
}
