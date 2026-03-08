package explainer

import (
	"fmt"
	"math"
	"strings"
	"text/template"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"percent": func(f float64) string {
			return fmt.Sprintf("%.0f%%", f*100)
		},
		"signedPercent": func(f float64) string {
			return fmt.Sprintf("%+.0f%%", f*100)
		},
		"signedFloat": func(f float64) string {
			return fmt.Sprintf("%+.2f", f)
		},
		"abs": math.Abs,
		"join": func(values []string, sep string) string {
			return strings.Join(values, sep)
		},
		"titleCase": func(value string) string {
			return cases.Title(language.English).String(value)
		},
		"truncate": func(value string, max int) string {
			value = strings.TrimSpace(value)
			if max <= 0 || len(value) <= max {
				return value
			}
			return value[:max] + "..."
		},
	}
}
