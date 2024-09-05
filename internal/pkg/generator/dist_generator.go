package generator

import (
	"fmt"
	"math/rand"

	"github.com/codingconcepts/dg/internal/pkg/model"
	"github.com/samber/lo"
)

type DistGenerator struct {
	Values  []string
	Weights []int
}

func (g *DistGenerator) Generate(t model.Table, c model.Column, files map[string]model.CSVFile) error {
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
		return fmt.Errorf("total weight is zero")
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
