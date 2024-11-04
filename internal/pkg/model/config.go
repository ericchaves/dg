package model

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
)

// Config represents the entire contents of a config file.
type Config struct {
	Tables  []Table  `yaml:"tables"`
	Inputs  []Input  `yaml:"inputs"`
	Extends []string `yaml:"extends"`
}

// Table represents the instructions to create one CSV file.
type Table struct {
	Name          string   `yaml:"name"`
	Count         int      `yaml:"count"`
	Suppress      bool     `yaml:"suppress"`
	UniqueColumns []string `yaml:"unique_columns"`
	Columns       []Column `yaml:"columns"`
}

// Column represents the instructions to populate one CSV file column.
type Column struct {
	Name      string     `yaml:"name"`
	Type      string     `yaml:"type"`
	Suppress  bool       `yaml:"suppress"`
	Generator RawMessage `yaml:"processor"`
}

// Input represents a data source provided by the user.
type Input struct {
	Name   string     `yaml:"name"`
	Type   string     `yaml:"type"`
	Source RawMessage `yaml:"source"`
}

// Load config from a file
func LoadConfig(r io.Reader, baseDir string) (Config, error) {
	var c Config
	if err := yaml.NewDecoder(r).Decode(&c); err != nil {
		return Config{}, fmt.Errorf("parsing file: %w", err)
	}

	// Process extends section if it exists
	if len(c.Extends) > 0 {
		var mergedConfig Config
		for _, extendFile := range c.Extends {
			fullPath := path.Join(baseDir, extendFile)
			extFile, err := os.Open(fullPath)
			if err != nil {
				return Config{}, fmt.Errorf("opening extended file %s: %w", fullPath, err)
			}
			defer extFile.Close()

			// Recursively load extended config
			extConfig, err := LoadConfig(extFile, path.Dir(fullPath))
			if err != nil {
				return Config{}, fmt.Errorf("loading extended config from %s: %w", fullPath, err)
			}

			// Merge the extended config with current merged config
			mergedConfig = MergeConfig(mergedConfig, extConfig)
		}
		// Finally merge with the current config
		c = MergeConfig(mergedConfig, c)
	}

	return c, nil
}

func MergeConfig(current Config, partial Config) Config {
	result := current

	// Merge Inputs
	for _, newInput := range partial.Inputs {
		found := false
		for i, existingInput := range result.Inputs {
			if existingInput.Name == newInput.Name {
				// Replace the entire Input if names match
				result.Inputs[i] = newInput
				found = true
				break
			}
		}
		if !found {
			// Add new Input if no matching name was found
			result.Inputs = append(result.Inputs, newInput)
		}
	}

	// Merge Tables
	for _, overrideTable := range partial.Tables {
		tableFound := false
		for i, currentTable := range result.Tables {
			if overrideTable.Name == currentTable.Name {
				tableFound = true
				// Rule: the new count always overrides previous table count.
				result.Tables[i].Count = overrideTable.Count
				// Rule: the new suppress always overrides previous table suppress flag.
				result.Tables[i].Suppress = overrideTable.Suppress

				// Rule: if new columns exist, replace the entire column spec
				if len(overrideTable.Columns) > 0 {
					// Use append to replace current columns
					result.Tables[i].Columns = append(result.Tables[i].Columns[:0], overrideTable.Columns...)
				}

				// Rule: merge UniqueColumns Filtering columns that dont exists in final result
				if overrideTable.UniqueColumns != nil {
					combined := append(result.Tables[i].UniqueColumns, overrideTable.UniqueColumns...)
					result.Tables[i].UniqueColumns = lo.Uniq(combined)
				}
				break
			}
		}
		// Rule: new tables are added to the config
		if !tableFound {
			result.Tables = append(result.Tables, overrideTable)
		}
	}

	return result
}
