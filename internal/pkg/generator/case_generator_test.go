package generator

import (
	"testing"

	"github.com/codingconcepts/dg/internal/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestCaseGenerator(t *testing.T) {
	table := model.Table{
		Name:    "table",
		Columns: []model.Column{{Name: "person_id"}, {Name: "age"}, {Name: "gender"}},
	}
	column := model.Column{
		Name: "greetings",
	}
	files := map[string]model.CSVFile{
		"table": {
			Name:   "table",
			Header: []string{"person_id", "age", "gender"},
			Lines: [][]string{
				{"1", "2", "3", "4", "5", "6", "7", "8"},
				{"5", "5", "15", "18", "25", "35", "50", "60"},
				{"male", "female", "male", "female", "male", "female", "male", "female"},
			},
		},
	}
	g := CaseGenerator{
		{
			When: "int(age) <= 10 && gender == 'male'",
			Then: "'little guy!'",
		},
		{
			When: "int(age) <= 10 && gender == 'female'",
			Then: "'little girl!'",
		},
		{
			When: "int(age) > 10 && int(age) <= 20 && gender == 'male'",
			Then: "'Hey, young man!'",
		},
		{
			When: "int(age) > 10 && int(age) <= 20 && gender == 'female'",
			Then: "'Hey, young lady!'",
		},
		{
			When: "int(age) > 20 && int(age) <= 50 && gender == 'male'",
			Then: "'Hello, sir!'",
		},
		{
			When: "int(age) > 20 && int(age) <= 50 && gender == 'female'",
			Then: "'Hello, madam!'",
		},
		{
			When: "int(age) > 50 && gender == 'male'",
			Then: "'Good day, sir!'",
		},
		{
			When: "int(age) > 50 && gender == 'female'",
			Then: "'Good day, madam!'",
		},
		{
			When: "true",
			Then: "'Yo, Stranger!'",
		},
	}
	err := g.Generate(table, column, files)
	assert.Nil(t, err)

	// Verify the generated greetings
	generatedGreetings := files["table"].Lines[3]
	expectedGreetings := []string{
		"little guy!",
		"little girl!",
		"Hey, young man!",
		"Hey, young lady!",
		"Hello, sir!",
		"Hello, madam!",
		"Hello, sir!",
		"Good day, madam!",
	}

	assert.Equal(t, len(expectedGreetings), len(generatedGreetings), "Number of generated greetings should match expected")

	for i, greeting := range generatedGreetings {
		assert.Equal(t, expectedGreetings[i], greeting, "Greeting at index %d should match expected", i)
	}
}
