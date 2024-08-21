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
		Values: []CaseCondition{
			{
				When:  "age <= 10 && gender == 'male'",
				Value: "'little guy!'",
			},
			{
				When:  "age <= 10 && gender == 'female'",
				Value: "'little girl!'",
			},
			{
				When:  "age > 10 && age <= 20 && gender == 'male'",
				Value: "'Hey, young man!'",
			},
			{
				When:  "age > 10 && age <= 20 && gender == 'female'",
				Value: "'Hey, young lady!'",
			},
			{
				When:  "age > 20 && age <= 50 && gender == 'male'",
				Value: "'Hello, sir!'",
			},
			{
				When:  "age > 20 && age <= 50 && gender == 'female'",
				Value: "'Hello, ma'am!'",
			},
			{
				When:  "age > 50 && gender == 'male'",
				Value: "'Good day, sir!'",
			},
			{
				When:  "age > 50 && gender == 'female'",
				Value: "'Good day, ma'am!'",
			},
			{
				When:  "true",
				Value: "'Yo, Stranger!'",
			},
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
		"Hello, ma'am!",
		"Good day, sir!",
		"Good day, ma'am!",
	}

	assert.Equal(t, len(expectedGreetings), len(generatedGreetings), "Number of generated greetings should match expected")

	for i, greeting := range generatedGreetings {
		assert.Equal(t, expectedGreetings[i], greeting, "Greeting at index %d should match expected", i)
	}
}
