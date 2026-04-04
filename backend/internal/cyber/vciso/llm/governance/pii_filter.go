package governance

import (
	"regexp"
	"strings"
)

var (
	ssnPattern    = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	cardPattern   = regexp.MustCompile(`\b(?:\d[ -]*?){13,16}\b`)
	emailPattern  = regexp.MustCompile(`\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b`)
)

type PIIFilter struct{}

func NewPIIFilter() *PIIFilter {
	return &PIIFilter{}
}

func (f *PIIFilter) Filter(value string) (string, int) {
	if strings.TrimSpace(value) == "" {
		return value, 0
	}
	count := 0
	replacements := []struct {
		re      *regexp.Regexp
		replace string
	}{
		{ssnPattern, "[redacted-ssn]"},
		{cardPattern, "[redacted-card]"},
		{emailPattern, "[redacted-email]"},
	}
	filtered := value
	for _, item := range replacements {
		matches := item.re.FindAllStringIndex(filtered, -1)
		if len(matches) == 0 {
			continue
		}
		count += len(matches)
		filtered = item.re.ReplaceAllString(filtered, item.replace)
	}
	return filtered, count
}
