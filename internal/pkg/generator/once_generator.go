package generator

import (
	"fmt"
	"math/rand"

	"github.com/codingconcepts/dg/internal/pkg/model"
)

type OnceGenerator struct {
	Table  string `yaml:"table"`
	Column string `yaml:"column"`
}

func (g OnceGenerator) Generate(t model.Table, col model.Column, files map[string]model.CSVFile) error {
	refFile, ok := files[g.Table]
	if !ok {
		return fmt.Errorf("referenced table %s not found", g.Table)
	}

	columnValues := refFile.GetColumnValues(g.Column)
	if len(columnValues) == 0 {
		return fmt.Errorf("no values found in column %s of table %s", g.Column, g.Table)
	}

	if t.Count == 0 {
		t.Count = len(columnValues)
	}

	if t.Count > len(columnValues) {
		return fmt.Errorf("not enough unique values in the pool to generate %d rows", t.Count)
	}

	lines := make([]string, t.Count)
	copy(lines, columnValues[:t.Count])

	rand.Shuffle(len(lines), func(i, j int) {
		lines[i], lines[j] = lines[j], lines[i]
	})

	AddTable(t, col.Name, lines, files)
	return nil
}
