package generator

import (
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"github.com/expr-lang/expr"
	"github.com/samber/lo"

	"github.com/codingconcepts/dg/internal/pkg/model"
)

type ExprContext struct {
	Files  map[string]model.CSVFile
	Format string
}

func (ec *ExprContext) makeEnvFromLine(filename string, line int) map[string]any {
	refFile, ok := ec.Files[filename]
	if !ok {
		return ec.makeEnv(map[string]any{model.LN: strconv.Itoa(line)})
	}
	record := refFile.GetRecord(line)
	return ec.makeEnv(record)
}

func (ec *ExprContext) makeEnv(record map[string]any) map[string]any {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	env := map[string]any{
		"match": func(args ...any) (any, error) {
			if len(args) != 4 {
				return "", fmt.Errorf("match function expects 4 arguments: match(sourceTable string, sourceColumn string, sourceValue string, matchColumn string)")
			}
			sourceTable, sourceColumn, matchColumn := args[0].(string), args[1].(string), args[3].(string)
			sourceValue := fmt.Sprintf("%v", args[2])
			value, err := ec.searchTable(sourceTable, sourceColumn, sourceValue, matchColumn)
			if err != nil {
				return nil, err
			}
			return value, nil
		},
		"add_date": func(args ...any) (any, error) {
			if len(args) != 4 {
				return "", fmt.Errorf("add_date function expects 5 arguments: add_date(years int, months int, days int, data date)")
			}
			years, err := strconv.Atoi(fmt.Sprintf("%v", args[0]))
			if err != nil {
				return "", fmt.Errorf("error parsing years: %w", err)
			}
			months, err := strconv.Atoi(fmt.Sprintf("%v", args[1]))
			if err != nil {
				return "", fmt.Errorf("error parsing months: %w", err)
			}
			days, err := strconv.Atoi(fmt.Sprintf("%v", args[2]))
			if err != nil {
				return "", fmt.Errorf("error parsing days: %w", err)
			}
			var data time.Time
			tipo := ec.getType(args[3])
			switch tipo {
			case "time.Time":
				data = args[3].(time.Time)
				return data.AddDate(years, months, days), nil
			case "float64":
				float, _ := args[3].(float64)
				sec := int64(float)
				nano := int64((float - float64(sec)) * 1e9)
				data = time.Unix(sec, nano)
				return data.AddDate(years, months, days), nil
			case "int":
				sec := int64(args[3].(int))
				data = time.Unix(sec, 0)
				return data.AddDate(years, months, days), nil
			case "string":
				digits := regexp.MustCompile(`(\d)+`)
				match := digits.FindAllString(ec.Format, -1)
				if len(match) >= 3 {
					data, err = time.Parse(ec.Format, args[3].(string))
				} else {
					data, err = time.Parse("2006-01-02", args[3].(string))
				}
				if err != nil {
					return "", fmt.Errorf("error parsing date: %w", err)
				}
				return data.AddDate(years, months, days), nil
			}
			return "", fmt.Errorf("error parsing date")
		},
		"rand": func(args ...any) (any, error) {
			if len(args) == 0 {
				return rand.Int(), nil
			}
			n, ok := args[0].(int)
			if !ok {
				return nil, fmt.Errorf("value %s cannot be convrted to int", args[0])
			}
			if n <= 0 {
				return nil, fmt.Errorf("value %s must be a positive integer", args[0])
			}
			return r.Intn(n), nil
		},
		"rand_float64": func(args ...any) (any, error) {
			return r.Float64(), nil
		},
		"randn": func(args ...any) (any, error) {
			if len(args) < 1 {
				return "", fmt.Errorf("rand function expects 1 argument: randn(n int)")
			}
			n, ok := args[0].(int)
			if !ok {
				return "", fmt.Errorf("argument %v cannot be converted to int", args[0])
			}
			a := int(math.Abs(float64(n)))
			return rand.Intn(a+1) + n, nil
		},
		"rand_range": func(args ...any) (any, error) {
			if len(args) < 2 {
				return "", fmt.Errorf("rand_range function expects 2 argument: rand_range(min int, max int)")
			}
			min, ok := args[0].(int)
			if !ok {
				return "", fmt.Errorf("argument %v cannot be converted to int", args[0])
			}
			max, ok := args[1].(int)
			if !ok {
				return "", fmt.Errorf("argument %v cannot be converted to int", args[1])
			}
			if min > max {
				min, max = max, min
			}
			return rand.Intn(max-min+1) + min, nil
		},
		"rand_perm": func(args ...any) (any, error) {
			if len(args) != 1 {
				return "", fmt.Errorf("rand_perm function expects 1 argument: rand_perm(n int)")
			}
			n, ok := args[0].(int)
			if !ok {
				return "", fmt.Errorf("argument %v cannot be converted to int", args[0])
			}
			return r.Perm(n), nil
		},
	}
	for k, v := range record {
		env[k] = v
	}
	return env
}

func (ec *ExprContext) evaluate(expression string, env any) (any, error) {
	output, err := expr.Eval(expression, env)
	if err != nil {
		return nil, fmt.Errorf("error evaluating expression: %w", err)
	}
	return output, nil
}

func (ec *ExprContext) searchTable(sourceTable string, sourceColumn, sourceValue, matchColumn string) (string, error) {
	sourceFile, exists := ec.Files[sourceTable]
	if !exists {
		return "", fmt.Errorf("table not found: %s", sourceTable)
	}
	return ec.searchFile(sourceFile, sourceColumn, sourceValue, matchColumn)
}

func (ec *ExprContext) searchFile(sourceFile model.CSVFile, sourceColumn, sourceValue, matchColumn string) (string, error) {
	sourceColumnIndex := lo.IndexOf(sourceFile.Header, sourceColumn)
	matchColumnIndex := lo.IndexOf(sourceFile.Header, matchColumn)
	if sourceColumnIndex == -1 || matchColumnIndex == -1 {
		return "", fmt.Errorf("column not found: %s ou %s in %s", sourceColumn, matchColumn, sourceFile.Name)
	}
	_, index, found := lo.FindIndexOf(sourceFile.Lines[sourceColumnIndex], func(item string) bool {
		return item == sourceValue
	})
	if found {
		return sourceFile.Lines[matchColumnIndex][index], nil
	}

	return "", fmt.Errorf("value not found for %s in column %s", sourceValue, sourceColumn)
}

func (ec *ExprContext) getType(value any) string {
	return reflect.TypeOf(value).String()
}

func (ec *ExprContext) AnyToString(value any) string {
	switch ec.getType(value) {
	case "time.Time":
		if ec.Format == "" {
			ec.Format = "2006-01-02"
		}
		return value.(time.Time).Format(ec.Format)
	case "float64":
		if ec.Format == "" {
			ec.Format = "%g"
		}
		return fmt.Sprintf(ec.Format, value.(float64))
	case "int":
		if ec.Format == "" {
			ec.Format = "%d"
		}
		return fmt.Sprintf(ec.Format, value.(int))
	case "bool":
		if ec.Format == "" {
			ec.Format = "%t"
		}
		return fmt.Sprintf(ec.Format, value.(bool))
	case "string":
		return value.(string)
	default:
		return fmt.Sprintf("%v", value)
	}
}
