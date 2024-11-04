package model

import (
	"strings"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func cloneColumnsWithoutPointers(t Table) []Column {
	clone := make([]Column, len(t.Columns))
	for i, col := range t.Columns {
		clone[i] = Column{
			Name:     col.Name,
			Type:     col.Type,
			Suppress: col.Suppress,
			Generator: RawMessage{
				UnmarshalFunc: nil,
			},
		}
	}

	return clone
}

func TestLoadConfig(t *testing.T) {
	y := `
inputs:
  - name: my_data
    type: csv
    source:
      file_name: my_data.csv

tables:
  - name: person
    count: 100
    columns:
      - name: id
        type: inc
        processor:
          start: 1
          format: "P%03d"
`

	config, err := LoadConfig(strings.NewReader(y), ".")
	assert.Nil(t, err)

	exp := Config{
		Inputs: []Input{
			{
				Name: "my_data",
				Type: "csv",
				Source: ToRawMessage(t, SourceCSV{
					FileName: "my_data.csv",
				}),
			},
		},
		Tables: []Table{
			{
				Name:  "person",
				Count: 100,
				Columns: []Column{
					{
						Name: "id",
						Type: "inc",
						Generator: ToRawMessage(t, map[string]any{
							"start":  1,
							"format": "P%03d",
						}),
					},
				},
			},
		},
	}

	assert.Equal(t, exp.Inputs[0].Name, config.Inputs[0].Name)
	assert.Equal(t, exp.Inputs[0].Type, config.Inputs[0].Type)

	var expSource SourceCSV
	assert.Nil(t, exp.Inputs[0].Source.UnmarshalFunc(&expSource))

	var actSource SourceCSV
	assert.Nil(t, config.Inputs[0].Source.UnmarshalFunc(&actSource))

	assert.Equal(t, expSource, actSource)

	assert.Equal(t, exp.Tables[0].Name, config.Tables[0].Name)
	assert.Equal(t, exp.Tables[0].Count, config.Tables[0].Count)
	assert.Equal(t, exp.Tables[0].Columns[0].Name, config.Tables[0].Columns[0].Name)
	assert.Equal(t, exp.Tables[0].Columns[0].Type, config.Tables[0].Columns[0].Type)

	var expProcessor map[string]any
	assert.Nil(t, exp.Tables[0].Columns[0].Generator.UnmarshalFunc(&expProcessor))

	var actProcessor map[string]any
	assert.Nil(t, config.Tables[0].Columns[0].Generator.UnmarshalFunc(&actProcessor))

	assert.Equal(t, expProcessor, actProcessor)
}

func TestMergeConfigsWithExamples(t *testing.T) {
	// Load config1.yaml
	config1, err := LoadConfig(strings.NewReader(`
inputs:
  - name: significant_event
    type: csv
    source:
      file_name: significant_dates.csv

tables:
  - name: events
    columns:
      - name: timeline_date
        type: range
        processor:
          type: date
          from: 1885-01-01
          to: 1985-10-26
          format: 2006-01-02
          step: 24h
      - name: timeline_event
        type: match
        processor:
          source_table: significant_event
          source_column: date
          source_value: events
          match_column: timeline_date
  
  - name: one
    columns:
      - name: c1
        type: const
        processor:
          values: [a, b, c]
`), ".")
	assert.NoError(t, err)

	// Load config2.yaml
	config2, err := LoadConfig(strings.NewReader(`
inputs:
  - name: market
    type: csv
    source:
      file_name: invalid_market.csv

tables:
  - name: events
    count: 30

  - name: one
    columns:
      - name: c2
        type: const
        processor:
          values: [d, e, f ]

  - name: market_product
    count: 10
    columns:
      - name: id
        type: gen
        processor:
          value: ${uuid}
      - name: market
        type: set
        processor:
          values: ["us", "in"]
      - name: region
        type: match
        processor:
          source_table: market
          source_column: code
          source_value: region
          match_column: market

`), ".")
	assert.NoError(t, err)

	// Load config3.yaml
	config3, err := LoadConfig(strings.NewReader(`
inputs:
  - name: market
    suppress: true
    type: csv
    source:
      file_name: market.csv
tables:
  - name: market_product
    suppress: true

`), ".")
	assert.NoError(t, err)

	// Load expected.yaml
	expectedConfig, err := LoadConfig(strings.NewReader(`
inputs:
  - name: significant_event
    type: csv
    source:
      file_name: significant_dates.csv
  - name: market
    type: csv
    source:
      file_name: market.csv
tables:
  - name: events
    count: 30
    columns:
      - name: timeline_date
        type: range
        processor:
          type: date
          from: 1885-01-01
          to: 1985-10-26
          format: 2006-01-02
          step: 24h
      - name: timeline_event
        type: match
        processor:
          source_table: significant_event
          source_column: date
          source_value: events
          match_column: timeline_date
  - name: one
    columns:
      - name: c2
        type: const
        processor:
          values:
            - d
            - e
            - f
  - name: market_product
    suppress: true
    columns:
      - name: id
        type: gen
        processor:
          value: ${uuid}
      - name: market
        type: set
        processor:
          values:
            - us
            - in
      - name: region
        type: match
        processor:
          source_table: market
          source_column: code
          source_value: region
          match_column: market

`), ".")
	assert.NoError(t, err)

	// Merge configs
	mergedConfig := MergeConfig(config1, config2)
	mergedConfig = MergeConfig(mergedConfig, config3)

	assert.NoError(t, err)
	assert.Equal(t, len(expectedConfig.Inputs), len(mergedConfig.Inputs))
	assert.Equal(t, expectedConfig.Inputs[0].Name, mergedConfig.Inputs[0].Name)
	assert.Equal(t, expectedConfig.Inputs[0].Type, mergedConfig.Inputs[0].Type)
	assert.Equal(t, expectedConfig.Inputs[1].Name, mergedConfig.Inputs[1].Name)
	assert.Equal(t, expectedConfig.Inputs[1].Type, mergedConfig.Inputs[1].Type)

	assert.Equal(t, len(expectedConfig.Tables), len(mergedConfig.Tables))

	for _, expectedTable := range expectedConfig.Tables {
		actualTable := lo.Filter(mergedConfig.Tables, func(t Table, _ int) bool {
			return t.Name == expectedTable.Name
		})[0]
		assert.NotNil(t, actualTable)
		assert.Equal(t, expectedTable.Name, actualTable.Name)
		expectedColumns := cloneColumnsWithoutPointers(expectedTable)
		actualColumns := cloneColumnsWithoutPointers(expectedTable)
		assert.Equal(t, expectedColumns, actualColumns)
	}
}
