package expression

import (
	"fmt"
	"math"
	"strings"
	"time"
)

type builtinFunc func(args []any) (any, error)

func builtins() map[string]builtinFunc {
	return map[string]builtinFunc{
		"TRIM": func(args []any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("TRIM expects 1 argument")
			}
			if args[0] == nil {
				return nil, nil
			}
			return strings.TrimSpace(toString(args[0])), nil
		},
		"UPPER": func(args []any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("UPPER expects 1 argument")
			}
			if args[0] == nil {
				return nil, nil
			}
			return strings.ToUpper(toString(args[0])), nil
		},
		"LOWER": func(args []any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("LOWER expects 1 argument")
			}
			if args[0] == nil {
				return nil, nil
			}
			return strings.ToLower(toString(args[0])), nil
		},
		"LENGTH": func(args []any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("LENGTH expects 1 argument")
			}
			if args[0] == nil {
				return nil, nil
			}
			return float64(len(toString(args[0]))), nil
		},
		"SUBSTRING": func(args []any) (any, error) {
			if len(args) != 3 {
				return nil, fmt.Errorf("SUBSTRING expects 3 arguments")
			}
			if args[0] == nil || args[1] == nil || args[2] == nil {
				return nil, nil
			}
			source := []rune(toString(args[0]))
			start, ok := toFloat(args[1])
			if !ok {
				return nil, fmt.Errorf("SUBSTRING start must be numeric")
			}
			length, ok := toFloat(args[2])
			if !ok {
				return nil, fmt.Errorf("SUBSTRING length must be numeric")
			}
			i := int(start)
			n := int(length)
			if i < 0 {
				i = 0
			}
			if i > len(source) {
				return "", nil
			}
			end := i + n
			if end > len(source) {
				end = len(source)
			}
			return string(source[i:end]), nil
		},
		"CONCAT": func(args []any) (any, error) {
			var b strings.Builder
			for _, arg := range args {
				if arg == nil {
					continue
				}
				b.WriteString(toString(arg))
				if b.Len() > 10000 {
					return nil, fmt.Errorf("concatenated string exceeds maximum length")
				}
			}
			return b.String(), nil
		},
		"COALESCE": func(args []any) (any, error) {
			for _, arg := range args {
				if arg != nil {
					return arg, nil
				}
			}
			return nil, nil
		},
		"ABS": func(args []any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("ABS expects 1 argument")
			}
			value, ok := toFloat(args[0])
			if !ok {
				return nil, fmt.Errorf("ABS expects numeric input")
			}
			return math.Abs(value), nil
		},
		"ROUND": func(args []any) (any, error) {
			if len(args) < 1 || len(args) > 2 {
				return nil, fmt.Errorf("ROUND expects 1 or 2 arguments")
			}
			value, ok := toFloat(args[0])
			if !ok {
				return nil, fmt.Errorf("ROUND expects numeric input")
			}
			precision := 0.0
			if len(args) == 2 {
				var precisionOK bool
				precision, precisionOK = toFloat(args[1])
				if !precisionOK {
					return nil, fmt.Errorf("ROUND precision must be numeric")
				}
			}
			factor := math.Pow(10, precision)
			return math.Round(value*factor) / factor, nil
		},
		"FLOOR": func(args []any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("FLOOR expects 1 argument")
			}
			value, ok := toFloat(args[0])
			if !ok {
				return nil, fmt.Errorf("FLOOR expects numeric input")
			}
			return math.Floor(value), nil
		},
		"CEIL": func(args []any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("CEIL expects 1 argument")
			}
			value, ok := toFloat(args[0])
			if !ok {
				return nil, fmt.Errorf("CEIL expects numeric input")
			}
			return math.Ceil(value), nil
		},
		"YEAR": func(args []any) (any, error) {
			tm, err := expectTimeFunc("YEAR", args)
			if err != nil {
				return nil, err
			}
			if tm == nil {
				return nil, nil
			}
			return float64(tm.Year()), nil
		},
		"MONTH": func(args []any) (any, error) {
			tm, err := expectTimeFunc("MONTH", args)
			if err != nil {
				return nil, err
			}
			if tm == nil {
				return nil, nil
			}
			return float64(tm.Month()), nil
		},
		"DAY": func(args []any) (any, error) {
			tm, err := expectTimeFunc("DAY", args)
			if err != nil {
				return nil, err
			}
			if tm == nil {
				return nil, nil
			}
			return float64(tm.Day()), nil
		},
		"NOW": func(args []any) (any, error) {
			if len(args) != 0 {
				return nil, fmt.Errorf("NOW expects 0 arguments")
			}
			return time.Now().UTC(), nil
		},
		"IF": func(args []any) (any, error) {
			if len(args) != 3 {
				return nil, fmt.Errorf("IF expects 3 arguments")
			}
			cond, ok := toBool(args[0])
			if !ok {
				return nil, fmt.Errorf("IF condition must be boolean")
			}
			if cond {
				return args[1], nil
			}
			return args[2], nil
		},
	}
}

func expectTimeFunc(name string, args []any) (*time.Time, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s expects 1 argument", name)
	}
	if args[0] == nil {
		return nil, nil
	}
	tm, ok := toTime(args[0])
	if !ok {
		return nil, fmt.Errorf("%s expects a datetime input", name)
	}
	return &tm, nil
}

