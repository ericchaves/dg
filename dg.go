package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"runtime/pprof"
	"strings"
	"text/template"
	"time"

	"github.com/codingconcepts/dg/internal/pkg/generator"
	"github.com/codingconcepts/dg/internal/pkg/model"
	"github.com/codingconcepts/dg/internal/pkg/source"
	"github.com/codingconcepts/dg/internal/pkg/ui"
	"github.com/codingconcepts/dg/internal/pkg/web"
	"github.com/samber/lo"
)

var (
	version string
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return strings.Join(*i, ", ")
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	log.SetFlags(0)

	var configPaths arrayFlags
	flag.Var(&configPaths, "c", "the absolute or relative path to the config file (can be used multiple times)")
	outputDir := flag.String("o", ".", "the absolute or relative path to the output dir")
	createImports := flag.String("i", "", "write import statements to file")
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	versionFlag := flag.Bool("version", false, "display the current version number")
	port := flag.Int("p", 0, "port to serve files from (omit to generate without serving)")
	flag.Parse()

	if *cpuprofile != "" {
		defer launchProfiler(*cpuprofile)()
	}

	if *versionFlag {
		fmt.Println(version)
		return
	}

	if len(configPaths) == 0 {
		flag.Usage()
		os.Exit(2)
	}

	tt := ui.TimeTracker(os.Stdout, realClock{}, 40)
	defer tt(time.Now(), "done")

	c, err := loadConfigs(configPaths, tt)
	if err != nil {
		log.Fatalf("error loading configs: %v", err)
	}

	files := make(map[string]model.CSVFile)

	if err = loadInputs(c, path.Dir(configPaths[0]), tt, files); err != nil {
		log.Fatalf("error loading inputs: %v", err)
	}

	if err = generateTables(c, tt, files); err != nil {
		log.Fatalf("error generating tables: %v", err)
	}

	if err = removeSuppressedColumns(c, tt, files); err != nil {
		log.Fatalf("error removing supressed columns: %v", err)
	}

	if err = reorderColumns(c, tt, files); err != nil {
		log.Fatalf("error validating files: %v", err)
	}

	if err := writeFiles(*outputDir, files, tt); err != nil {
		log.Fatalf("error writing csv files: %v", err)
	}

	if *createImports != "" {
		if err := writeImports(*outputDir, *createImports, c, files, tt); err != nil {
			log.Fatalf("error writing import statements: %v", err)
		}
	}

	if *port == 0 {
		return
	}

	log.Fatal(web.Serve(*outputDir, *port))
}

func loadConfigs(filenames []string, tt ui.TimerFunc) (model.Config, error) {
	defer tt(time.Now(), "loaded config files")

	var mergedConfig model.Config

	for _, filename := range filenames {
		file, err := os.Open(filename)
		if err != nil {
			return model.Config{}, fmt.Errorf("opening file %s: %w", filename, err)
		}
		defer file.Close()

		config, err := model.LoadConfig(file, path.Dir(filename))
		if err != nil {
			return model.Config{}, fmt.Errorf("loading config from %s: %w", filename, err)
		}

		mergedConfig = model.MergeConfig(mergedConfig, config)
	}

	return mergedConfig, nil
}

func loadInputs(c model.Config, configDir string, tt ui.TimerFunc, files map[string]model.CSVFile) error {
	defer tt(time.Now(), "loaded data sources")

	for _, input := range c.Inputs {
		if err := loadInput(input, configDir, tt, files); err != nil {
			return fmt.Errorf("loading input for %q: %w", input.Name, err)
		}
	}

	return nil
}

func loadInput(input model.Input, configDir string, tt ui.TimerFunc, files map[string]model.CSVFile) error {
	defer tt(time.Now(), fmt.Sprintf("loaded data source: %s", input.Name))

	switch input.Type {
	case "csv":
		var s model.SourceCSV
		if err := input.Source.UnmarshalFunc(&s); err != nil {
			return fmt.Errorf("parsing csv source for %s: %w", input.Name, err)
		}

		if err := source.LoadCSVSource(input.Name, configDir, s, files); err != nil {
			return fmt.Errorf("loading csv for %s: %w", input.Name, err)
		}
	}

	return nil
}

func generateTables(c model.Config, tt ui.TimerFunc, files map[string]model.CSVFile) error {
	defer tt(time.Now(), "generated all tables")

	for _, table := range c.Tables {
		if err := generateTable(table, files, tt); err != nil {
			return fmt.Errorf("generating csv file for %q: %w", table.Name, err)
		}
	}

	return nil
}

func reorderColumns(c model.Config, tt ui.TimerFunc, files map[string]model.CSVFile) error {
	defer tt(time.Now(), "reorder all table columns")

	for _, model := range c.Tables {
		file, ok := files[model.Name]
		if !ok {
			continue // Skip if the file doesn't exist
		}
		newHeader := make([]string, len(file.Header))
		newLines := make([][]string, len(file.Lines))
		i := 0
		for _, col := range model.Columns {
			if col.Suppress {
				continue
			}
			currentIndex := lo.IndexOf(file.Header, col.Name)
			if currentIndex < 0 || currentIndex > len(file.Lines) {
				return fmt.Errorf("column %s not found in file %s", col.Name, file.Name)
			}
			newHeader[i] = col.Name
			newLines[i] = file.Lines[currentIndex]
			i++
		}
		file.Header = newHeader
		file.Lines = newLines
		files[model.Name] = file
	}

	return nil
}

func generateTable(t model.Table, files map[string]model.CSVFile, tt ui.TimerFunc) error {
	defer tt(time.Now(), fmt.Sprintf("generated table: %s", t.Name))

	// Create any foreign_key columns next
	var fk generator.ForeignKeyGenerator
	if err := fk.Generate(t, files); err != nil {
		return fmt.Errorf("generating fk columns: %w", err)
	}

	// Create the Cartesian product of any each types first.
	var eg generator.EachGenerator
	if err := eg.Generate(t, files); err != nil {
		return fmt.Errorf("generating each columns: %w", err)
	}

	// Create any const columns next.
	var cg generator.ConstGenerator
	if err := cg.Generate(t, files); err != nil {
		return fmt.Errorf("generating const columns: %w", err)
	}

	for _, col := range t.Columns {
		switch col.Type {
		case "ref":
			var g generator.RefGenerator
			if err := col.Generator.UnmarshalFunc(&g); err != nil {
				return fmt.Errorf("parsing ref process for %s.%s: %w", t.Name, col.Name, err)
			}
			if err := g.Generate(t, col, files); err != nil {
				return fmt.Errorf("running ref process for %s.%s: %w", t.Name, col.Name, err)
			}

		case "gen":
			var g generator.GenGenerator
			if err := col.Generator.UnmarshalFunc(&g); err != nil {
				return fmt.Errorf("parsing each process for %s: %w", col.Name, err)
			}
			if err := g.Generate(t, col, files); err != nil {
				return fmt.Errorf("running gen process for %s.%s: %w", t.Name, col.Name, err)
			}

		case "set":
			var g generator.SetGenerator
			if err := col.Generator.UnmarshalFunc(&g); err != nil {
				return fmt.Errorf("parsing set process for %s.%s: %w", t.Name, col.Name, err)
			}
			if err := g.Generate(t, col, files); err != nil {
				return fmt.Errorf("running set process for %s.%s: %w", t.Name, col.Name, err)
			}

		case "inc":
			var g generator.IncGenerator
			if err := col.Generator.UnmarshalFunc(&g); err != nil {
				return fmt.Errorf("parsing each process for %s: %w", col.Name, err)
			}
			if err := g.Generate(t, col, files); err != nil {
				return fmt.Errorf("running inc process for %s.%s: %w", t.Name, col.Name, err)
			}

		case "range":
			var g generator.RangeGenerator
			if err := col.Generator.UnmarshalFunc(&g); err != nil {
				return fmt.Errorf("parsing range process for %s: %w", col.Name, err)
			}
			if err := g.Generate(t, col, files); err != nil {
				return fmt.Errorf("running range process for %s.%s: %w", t.Name, col.Name, err)
			}

		case "match":
			var g generator.MatchGenerator
			if err := col.Generator.UnmarshalFunc(&g); err != nil {
				return fmt.Errorf("parsing match process for %s: %w", col.Name, err)
			}
			if err := g.Generate(t, col, files); err != nil {
				return fmt.Errorf("running match process for %s.%s: %w", t.Name, col.Name, err)
			}

		case "cuid2":
			var g generator.Cuid2Generator
			if err := col.Generator.UnmarshalFunc(&g); err != nil {
				return fmt.Errorf("parsing cuid2 process for %s: %w", col.Name, err)
			}
			if err := g.Generate(t, col, files); err != nil {
				return fmt.Errorf("running cuid2 process for %s.%s: %w", t.Name, col.Name, err)
			}

		case "rel_date", "relative_date":
			var g generator.RelDateGenerator
			if err := col.Generator.UnmarshalFunc(&g); err != nil {
				return fmt.Errorf("parsing rel_date process for %s: %w", col.Name, err)
			}
			if err := g.Generate(t, col, files); err != nil {
				return fmt.Errorf("running rel_date process for %s.%s: %w", t.Name, col.Name, err)
			}

		case "rand":
			var g generator.RandGenerator
			if err := col.Generator.UnmarshalFunc(&g); err != nil {
				return fmt.Errorf("parsing rand process for %s: %w", col.Name, err)
			}
			if err := g.Generate(t, col, files); err != nil {
				return fmt.Errorf("running rand process for %s.%s: %w", t.Name, col.Name, err)
			}

		case "expr":
			var g generator.ExprGenerator
			if err := col.Generator.UnmarshalFunc(&g); err != nil {
				return fmt.Errorf("parsing expr process for %s: %w", col.Name, err)
			}
			if err := g.Generate(t, col, files); err != nil {
				return fmt.Errorf("running expr process for %s.%s: %w", t.Name, col.Name, err)
			}

		case "case":
			var g generator.CaseGenerator
			if err := col.Generator.UnmarshalFunc(&g); err != nil {
				return fmt.Errorf("parsing case process for %s: %w", col.Name, err)
			}
			if err := g.Generate(t, col, files); err != nil {
				return fmt.Errorf("running case process for %s.%s: %w", t.Name, col.Name, err)
			}

		case "map":
			var g generator.MapGenerator
			if err := col.Generator.UnmarshalFunc(&g); err != nil {
				return fmt.Errorf("parsing map process for %s: %w", col.Name, err)
			}
			if err := g.Generate(t, col, files); err != nil {
				return fmt.Errorf("running map process for %s.%s: %w", t.Name, col.Name, err)
			}

		case "pick":
			var g generator.PickGenerator
			if err := col.Generator.UnmarshalFunc(&g); err != nil {
				return fmt.Errorf("parsing once process for %s: %w", col.Name, err)
			}
			if err := g.Generate(t, col, files); err != nil {
				return fmt.Errorf("running once process for %s.%s: %w", t.Name, col.Name, err)
			}

		case "lookup":
			var g generator.LookupGenerator
			if err := col.Generator.UnmarshalFunc(&g); err != nil {
				return fmt.Errorf("parsing lookup process for %s: %w", col.Name, err)
			}
			if err := g.Generate(t, col, files); err != nil {
				return fmt.Errorf("running lookup process for %s.%s: %w", t.Name, col.Name, err)
			}

		case "dist":
			var g generator.DistGenerator
			if err := col.Generator.UnmarshalFunc(&g); err != nil {
				return fmt.Errorf("parsing dist process for %s: %w", col.Name, err)
			}
			if err := g.Generate(t, col, files); err != nil {
				return fmt.Errorf("running dist process for %s.%s: %w", t.Name, col.Name, err)
			}
		}
	}

	file, ok := files[t.Name]
	if !ok {
		return fmt.Errorf("missing table: %q", t.Name)
	}

	if len(file.UniqueColumns) > 0 {
		file.Lines = generator.Transpose(file.Lines)
		file.Lines = file.Unique()
		file.Lines = generator.Transpose(file.Lines)
	}
	files[t.Name] = file

	return nil
}

func removeSuppressedColumns(c model.Config, tt ui.TimerFunc, files map[string]model.CSVFile) error {
	defer tt(time.Now(), "removed suppressed columns")

	for _, table := range c.Tables {
		for _, column := range table.Columns {
			if !column.Suppress {
				continue
			}

			file, ok := files[table.Name]
			if !ok {
				return fmt.Errorf("missing table: %q", table.Name)
			}

			// Remove suppressed column from header.
			var headerIndex int
			file.Header = lo.Reject(file.Header, func(v string, i int) bool {
				if v == column.Name {
					headerIndex = i
					return true
				}
				return false
			})

			// Remove suppressed column from lines.
			file.Lines = append(file.Lines[:headerIndex], file.Lines[headerIndex+1:]...)

			files[table.Name] = file
		}
	}

	return nil
}

func writeFiles(outputDir string, cfs map[string]model.CSVFile, tt ui.TimerFunc) error {
	defer tt(time.Now(), "wrote all csvs")

	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	for name, file := range cfs {
		if !file.Output {
			continue
		}

		if err := writeFile(outputDir, name, file, tt); err != nil {
			return fmt.Errorf("writing file %q: %w", file.Name, err)
		}
	}

	return nil
}

func writeFile(outputDir, name string, cf model.CSVFile, tt ui.TimerFunc) error {
	defer tt(time.Now(), fmt.Sprintf("wrote csv: %s", name))

	fullPath := path.Join(outputDir, fmt.Sprintf("%s.csv", name))
	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("creating csv file %q: %w", name, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if err = writer.Write(cf.Header); err != nil {
		return fmt.Errorf("writing csv header for %q: %w", name, err)
	}

	cf.Lines = generator.Transpose(cf.Lines)

	if err = writer.WriteAll(cf.Lines); err != nil {
		return fmt.Errorf("writing csv lines for %q: %w", name, err)
	}

	writer.Flush()
	return nil
}

func writeImports(outputDir, name string, c model.Config, files map[string]model.CSVFile, tt ui.TimerFunc) error {
	defer tt(time.Now(), fmt.Sprintf("wrote imports: %s", name))

	importTmpl := template.Must(template.New("import").
		Funcs(template.FuncMap{"join": strings.Join}).
		Parse(`IMPORT INTO {{.Name}} (
	{{ join .Header ", " }}
)
CSV DATA (
    '.../{{.Name}}.csv'
)
WITH skip='1', nullif = '', allow_quoted_null;

`),
	)

	fullPath := path.Join(outputDir, name)
	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("creating csv file %q: %w", name, err)
	}
	defer file.Close()

	// Iterate through the tables in the config file, so the imports are in the right order.
	for _, table := range c.Tables {
		csv := files[table.Name]
		if !csv.Output {
			continue
		}

		if err := importTmpl.Execute(file, csv); err != nil {
			return fmt.Errorf("writing import statement for %q: %w", name, err)
		}
	}

	return nil
}

func launchProfiler(cpuprofile string) func() {
	f, err := os.Create(cpuprofile)
	if err != nil {
		log.Fatalf("creating file for profiler: %v", err)
	}
	pprof.StartCPUProfile(f)

	return func() {
		pprof.StopCPUProfile()
	}
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

func (realClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

// Generator interface for all generator types
type Generator interface {
	Generate(t model.Table, col model.Column, files map[string]model.CSVFile) error
}
