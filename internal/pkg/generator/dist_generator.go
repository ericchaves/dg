package generator

import (
	"fmt"
	"math/rand"
	"reflect"

	"github.com/codingconcepts/dg/internal/pkg/model"
	"github.com/samber/lo"
)

type DistGenerator struct {
	Values     []string
	Weights    []int
	Expression string
}

func (g *DistGenerator) Generate(t model.Table, c model.Column, files map[string]model.CSVFile) error {

	if g.Expression != "" {
		ec := &ExprContext{Files: files}
		env := ec.makeEnv()
		result, err := ec.evaluate(g.Expression, env)
		if err != nil {
			return fmt.Errorf("error evaluating expression %w", err)
		}
		v := reflect.ValueOf(result)
		if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
			return fmt.Errorf("expression must return array or slice but returned %s instead", v.Kind().String())
		}

		// Convert slice to []string for counting
		values := make([]string, v.Len())
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			if elem.Kind() == reflect.String || elem.Kind() == reflect.Interface {
				values[i] = elem.Interface().(string)
			} else {
				return fmt.Errorf("expression must return an array or slice of strings, but element %d is %s", i, elem.Kind().String())
			}
		}

		// Count occurrences and use them as weights
		counts := lo.CountValues(values)
		for value, count := range counts {
			g.Values = append(g.Values, value)
			g.Weights = append(g.Weights, count)
		}
	}

	if len(g.Values) == 0 {
		return fmt.Errorf("values slice is empty")
	}

	if retFile, ok := files[t.Name]; t.Count <= 0 && ok {
		t.Count = len(lo.MaxBy(retFile.Lines, func(a, b []string) bool {
			return len(a) > len(b)
		}))
	}

	lines := make([]string, 0, t.Count)

	totalWeight := 0
	normalizedWeights := make([]int, len(g.Values))
	for i := 0; i < len(g.Values); i++ {
		weight := 0
		if i < len(g.Weights) {
			weight = g.Weights[i]
		}
		normalizedWeights[i] = weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		// When no weights are provided, use equal weights of 1 for all values
		for i := range normalizedWeights {
			normalizedWeights[i] = 1
		}
		totalWeight = len(normalizedWeights)
	}

	for i, value := range g.Values {
		numberOfItems := (normalizedWeights[i] * t.Count) / totalWeight
		for j := 0; j < numberOfItems; j++ {
			lines = append(lines, value)
		}
	}

	// fill array if needed
	for len(lines) < t.Count {
		for _, value := range g.Values {
			if len(lines) < t.Count {
				lines = append(lines, value)
			}
		}
	}

	rand.Shuffle(len(lines), func(i, j int) {
		lines[i], lines[j] = lines[j], lines[i]
	})

	AddTable(t, c.Name, lines, files)
	return nil
}
