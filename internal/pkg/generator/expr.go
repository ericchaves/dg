package generator

import (
	"crypto/sha256"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"strings"
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

func (ec *ExprContext) mergeEnv(env map[string]any, record map[string]any) error {
	for k, v := range record {
		if env[k] != nil {
			return fmt.Errorf("cannot merge field %s into env", k)
		}
		env[k] = v
	}
	return nil
}

func (ec *ExprContext) makeEnv() map[string]any {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	faker := initGofakeit()
	env := map[string]any{
		"match": func(sourceTable string, sourceColumn string, sourceValue string, matchColumn string) (any, error) {
			value, err := ec.searchTable(sourceTable, sourceColumn, sourceValue, matchColumn)
			if err != nil {
				return nil, err
			}
			return value, nil
		},
		"add_date": func(years int, months int, days int, data any) (any, error) {
			switch value := data.(type) {
			case time.Time:
				return value.AddDate(years, months, days), nil
			case float64:
				sec := int64(value)
				nano := int64((value - float64(sec)) * 1e9)
				output := time.Unix(sec, nano)
				return output.AddDate(years, months, days), nil
			case int:
				sec := int64(value)
				output := time.Unix(sec, 0)
				return output.AddDate(years, months, days), nil
			case string:
				if ec.Format == "" {
					ec.Format = "2006-01-02"
				}
				output, err := time.Parse(ec.Format, value)
				if err != nil {
					return "", fmt.Errorf("error formating date: %w", err)
				}
				return output.AddDate(years, months, days), nil
			}
			return "", fmt.Errorf("error parsing date: %q", data)
		},
		"rand": func(n int) int {
			if n < 0 {
				return r.Intn(n*-1) * -1
			}
			return r.Intn(n)
		},
		"randf64": func() float64 {
			return r.Float64()
		},
		"randr": func(min int, max int) int {
			if min > max {
				min, max = max, min
			}
			return r.Intn(max-min+1) + min
		},
		"randp": func(n int) []int {
			return r.Perm(n)
		},
		"get_record": func(table string, line int) (map[string]any, error) {
			return model.GetRecord(table, line, ec.Files), nil
		},
		"get_column": func(table string, column string) ([]string, error) {
			return model.GetColumnValues(table, column, ec.Files), nil
		},
		"payments": func(total float64, installments int, percentage float64) ([]float64, error) {
			if installments <= 0 {
				return []float64{}, fmt.Errorf("installments must be a positive int, got %d", installments)
			}
			if percentage < 0.0 || percentage > 1.0 {
				return []float64{}, fmt.Errorf("percentage must be float64 between 0.0 and 1.0: %v", percentage)
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
		"pmt": func(rate float64, nper int, pv float64, fv float64, pmtType int) (float64, error) {
			result, err := fin.Payment(rate, nper, pv, fv, pmtType)
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
		"sha256": func(data string) (string, error) {
			sum := sha256.Sum256([]byte(data))
			return fmt.Sprintf("%x", sum), nil
		},
		"pad": func(s string, char string, length int, left bool) string {
			if len(s) >= length {
				return s
			}
			padding := strings.Repeat(char, length-len(s))
			if left {
				return padding + s
			}
			return s + padding
		},
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
	if sourceColumnIndex == -1 {
		return "", fmt.Errorf("column not found: %s in %s", sourceColumn, sourceFile.Name)
	}
	if matchColumnIndex == -1 {
		return "", fmt.Errorf("column not found: %s in %s", matchColumn, sourceFile.Name)
	}
	_, index, found := lo.FindIndexOf(sourceFile.Lines[sourceColumnIndex], func(item string) bool {
		return item == sourceValue
	})
	if found {
		return sourceFile.Lines[matchColumnIndex][index], nil
	}

	return "", fmt.Errorf("value not found for %s in column %s", sourceValue, sourceColumn)
}

func (ec *ExprContext) searchRecord(sourceFile model.CSVFile, sourceColumn, sourceValue, matchColumn string, predicate string) (map[string]any, error) {
	sourceColumnIndex := lo.IndexOf(sourceFile.Header, sourceColumn)
	matchColumnIndex := lo.IndexOf(sourceFile.Header, matchColumn)
	if sourceColumnIndex == -1 {
		return map[string]any{}, fmt.Errorf("column not found: %s in %s", sourceColumn, sourceFile.Name)
	}
	if matchColumnIndex == -1 {
		return map[string]any{}, fmt.Errorf("column not found: %s in %s", matchColumn, sourceFile.Name)
	}
	columnValues := sourceFile.Lines[sourceColumnIndex]
	for i, item := range columnValues {
		if item == sourceValue {
			record := sourceFile.GetRecord(i)
			if predicate != "" {
				env := ec.makeEnv()
				if err := ec.mergeEnv(env, record); err != nil {
					return nil, err
				}
				match, err := ec.evaluate(predicate, env)
				if err != nil {
					return nil, err
				}
				if !ec.AnyToBool(match) {
					continue
				}
			}
			return record, nil
		}
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
			ec.Format = "%v"
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
		Generate: func(f *gofakeit.Faker, m *gofakeit.MapParams, info *gofakeit.Info) (any, error) {
			return cpf.Generate(), nil
		},
	}
	gofakeit.AddFuncLookup("Cpf", cpfInfo)
	gofakeit.AddFuncLookup("cpf", cpfInfo)

	cpnjInfo := gofakeit.Info{
		Generate: func(f *gofakeit.Faker, m *gofakeit.MapParams, info *gofakeit.Info) (any, error) {
			return cnpj.Generate(), nil
		},
	}
	gofakeit.AddFuncLookup("Cnpj", cpnjInfo)
	gofakeit.AddFuncLookup("cnpj", cpnjInfo)

	regexInfo := gofakeit.Info{
		Generate: func(f *gofakeit.Faker, m *gofakeit.MapParams, info *gofakeit.Info) (any, error) {
			pattern := m.Get("")
			return gofakeit.Regex(pattern[0]), nil
		},
	}
	gofakeit.AddFuncLookup("regex", regexInfo)

	faker := gofakeit.New(0)
	return faker
}
