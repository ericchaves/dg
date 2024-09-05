package generator

import (
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"github.com/alpeb/go-finance/fin"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/codingconcepts/dg/internal/pkg/model"
	"github.com/expr-lang/expr"
	"github.com/martinusso/go-docs/cnpj"
	"github.com/martinusso/go-docs/cpf"
	"github.com/samber/lo"
)

type ExprContext struct {
	Files  map[string]model.CSVFile
	Format string
}

func (ec *ExprContext) makeEnvFromLine(filename string, line int) map[string]any {
	refFile, ok := ec.Files[filename]
	if !ok {
		return ec.makeEnv(map[string]any{model.ROW_NUMBER: strconv.Itoa(line)})
	}
	record := refFile.GetRecord(line)
	return ec.makeEnv(record)
}

func (ec *ExprContext) makeEnv(record map[string]any) map[string]any {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	faker := initGofakeit()
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
			switch args[3].(type) {
			case time.Time:
				data = args[3].(time.Time)
				return data.AddDate(years, months, days), nil
			case float64:
				float, _ := args[3].(float64)
				sec := int64(float)
				nano := int64((float - float64(sec)) * 1e9)
				data = time.Unix(sec, nano)
				return data.AddDate(years, months, days), nil
			case int:
				sec := int64(args[3].(int))
				data = time.Unix(sec, 0)
				return data.AddDate(years, months, days), nil
			case string:
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
		"get_record": func(args ...any) (map[string]any, error) {
			if len(args) != 2 {
				return map[string]any{}, fmt.Errorf("get_record function expects 2 arguments: get_record(table string, line int)")
			}
			table, line := args[0].(string), args[1].(int)
			return model.GetRecord(table, line, ec.Files), nil
		},
		"get_column": func(args ...any) ([]string, error) {
			if len(args) != 2 {
				return []string{}, fmt.Errorf("get_record function expects 2 arguments: get_column(table string, column string)")
			}
			table, column := args[0].(string), args[1].(string)
			return model.GetColumnValues(table, column, ec.Files), nil
		},
		"payments": func(args ...any) ([]float64, error) {
			if len(args) != 3 {
				return []float64{}, fmt.Errorf("payments function expects 3 arguments: payments(total float64, installments int, percentage float64)")
			}
			total, ok := args[0].(float64)
			if !ok {
				return []float64{}, fmt.Errorf("total must be a float64, got %T", args[0])
			}
			installments, ok := args[1].(int)
			if !ok {
				return []float64{}, fmt.Errorf("installments must be an int, got %T", args[1])
			}
			percentage, ok := args[2].(float64)
			if !ok {
				return []float64{}, fmt.Errorf("percentage must be a float64, got %T", args[2])
			}

			if installments <= 0 {
				return []float64{}, fmt.Errorf("installments must be a positive int, got %d", installments)
			}
			if percentage < 0.0 || percentage > 1.0 {
				return []float64{}, fmt.Errorf("percentage must be float64 between 0.0 and 1.0: %s", args[2])
			}
			if installments == 1 {
				return []float64{total, 0.0}, nil
			}
			downPaymentAmount := total * percentage
			remainingAmount := total - downPaymentAmount
			installmentAmount := remainingAmount / float64(installments-1)
			totalInstallments := installmentAmount * float64(installments-1)
			difference := remainingAmount - totalInstallments

			if math.Abs(difference) > 0 {
				downPaymentAmount = downPaymentAmount + difference
			}

			result := []float64{downPaymentAmount, installmentAmount}
			return result, nil
		},
		"pmt": func(args ...any) (float64, error) {
			if len(args) != 5 {
				return 0, fmt.Errorf("pmt expected 5 arguments: pmt(rate float64, nper int, pv float64, fv float64, type int)")
			}
			rate, ok := args[0].(float64)
			if !ok {
				return 0, fmt.Errorf("rate must be a float64 got %T", args[0])
			}
			nper, ok := args[1].(int)
			if !ok {
				return 0, fmt.Errorf("nper must be an int, got %T", args[1])
			}
			pv, ok := args[2].(float64)
			if !ok {
				return 0, fmt.Errorf("pv must be a float64, got %T", args[2])
			}
			fv, ok := args[3].(float64)
			if !ok {
				return 0, fmt.Errorf("fv must be a float64, got %T", args[3])
			}
			typ, ok := args[4].(int)
			if !ok {
				return 0, fmt.Errorf("type must be a bool, got %T", args[4])
			}

			result, err := fin.Payment(rate, nper, pv, fv, typ)
			return result, err
		},
		"fakeit": func(function string, params map[string]any) (any, error) {
			info := gofakeit.GetFuncLookup(function)
			if info == nil {
				return nil, fmt.Errorf("function %s not found in gofakeit", function)
			}

			mapString := gofakeit.NewMapParams()
			for key, value := range params {
				v := reflect.ValueOf(value)
				switch v.Kind() {
				case reflect.Bool:
					mapString.Add(key, fmt.Sprintf("%v", value))
				case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
					mapString.Add(key, fmt.Sprintf("%v", value))
				case reflect.Uint, reflect.Uint8, reflect.Uint32, reflect.Uint64:
					mapString.Add(key, fmt.Sprintf("%v", value))
				case reflect.Float32, reflect.Float64:
					mapString.Add(key, fmt.Sprintf("%v", value))
				case reflect.String:
					mapString.Add(key, fmt.Sprintf("%v", value))
				case reflect.Map:
					mapString.Add(key, fmt.Sprintf("%v", value))
				case reflect.Slice:
					var vals []string
					for i := 0; i < v.Len(); i++ {
						vals = append(vals, fmt.Sprintf("%v", v.Index(i).Interface()))
					}

					for _, val := range vals {
						mapString.Add(key, val)
					}
				}
			}

			data, error := info.Generate(faker, mapString, info)
			if error != nil {
				return nil, error
			}
			return data, nil
		},
	}

	for k, v := range record {
		if _, has := env[k]; has {
			env["fn_"+k] = env[k]
		}
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
	return ec.searchValue(sourceFile, sourceColumn, sourceValue, matchColumn)
}

func (ec *ExprContext) searchValue(sourceFile model.CSVFile, sourceColumn, sourceValue, matchColumn string) (string, error) {
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

func (ec *ExprContext) searchRecord(sourceFile model.CSVFile, sourceColumn, sourceValue, matchColumn string) (map[string]any, error) {
	sourceColumnIndex := lo.IndexOf(sourceFile.Header, sourceColumn)
	matchColumnIndex := lo.IndexOf(sourceFile.Header, matchColumn)
	if sourceColumnIndex == -1 || matchColumnIndex == -1 {
		return map[string]any{}, fmt.Errorf("column not found: %s ou %s in %s", sourceColumn, matchColumn, sourceFile.Name)
	}
	_, index, found := lo.FindIndexOf(sourceFile.Lines[sourceColumnIndex], func(item string) bool {
		return item == sourceValue
	})
	if found {
		return sourceFile.GetRecord(index), nil
	}

	return map[string]any{}, fmt.Errorf("value not found for %s in column %s", sourceValue, sourceColumn)
}

func (ec *ExprContext) AnyToString(value any) string {
	switch v := value.(type) {
	case time.Time:
		if ec.Format == "" {
			ec.Format = "2006-01-02"
		}
		return v.Format(ec.Format)
	case float64:
		if ec.Format == "" {
			ec.Format = "%g"
		}
		return fmt.Sprintf(ec.Format, v)
	case int:
		if ec.Format == "" {
			ec.Format = "%d"
		}
		return fmt.Sprintf(ec.Format, v)
	case bool:
		if ec.Format == "" {
			ec.Format = "%t"
		}
		return fmt.Sprintf(ec.Format, v)
	case string:
		return v
	default:
		return fmt.Sprintf("%q", value)
	}
}

func (ec *ExprContext) AnyToBool(value any) bool {
	if value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(v).Int() != 0
	case uint, uint8, uint16, uint32, uint64:
		return reflect.ValueOf(v).Uint() != 0
	case float32, float64:
		return reflect.ValueOf(v).Float() != 0
	case string:
		return v != ""
	case []interface{}:
		return len(v) > 0
	case map[interface{}]interface{}:
		return len(v) > 0
	default:
		return true
	}
}

func initGofakeit() *gofakeit.Faker {
	cpfInfo := gofakeit.Info{
		Display:     "cpf",
		Category:    "cpf",
		Description: "generate brazilian cpf",
		Generate: func(f *gofakeit.Faker, m *gofakeit.MapParams, info *gofakeit.Info) (any, error) {
			return cpf.Generate(), nil
		},
	}
	gofakeit.AddFuncLookup("Cpf", cpfInfo)
	gofakeit.AddFuncLookup("cpf", cpfInfo)
	cpnjInfo := gofakeit.Info{
		Display:     "cnpj",
		Category:    "cnpj",
		Description: "generate brazilian cnpj",
		Generate: func(f *gofakeit.Faker, m *gofakeit.MapParams, info *gofakeit.Info) (any, error) {
			return cnpj.Generate(), nil
		},
	}
	gofakeit.AddFuncLookup("Cnpj", cpnjInfo)
	gofakeit.AddFuncLookup("cnpj", cpnjInfo)
	faker := gofakeit.New(0)
	return faker
}
