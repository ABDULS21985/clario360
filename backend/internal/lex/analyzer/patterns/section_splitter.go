package patterns

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Section struct {
	Reference  string
	Title      string
	Text       string
	PageNumber int
}

type SectionSplitter struct {
	numbered   *regexp.Regexp
	article    *regexp.Regexp
	capsTitle  *regexp.Regexp
	refPattern *regexp.Regexp
}

func NewSectionSplitter() *SectionSplitter {
	return &SectionSplitter{
		numbered:   regexp.MustCompile(`(?m)^(?:Section\s+)?(?:(?:\d+\.)+\d*|\d+)\s+.+$`),
		article:    regexp.MustCompile(`(?im)^Article\s+[A-ZIVXLC0-9]+.*$`),
		capsTitle:  regexp.MustCompile(`(?m)^[A-Z][A-Z\s&/\-]{3,}$`),
		refPattern: regexp.MustCompile(`(?i)^(Section\s+[A-Za-z0-9\.\-]+|Article\s+[A-ZIVXLC0-9]+|(?:\d+\.)+\d*|\d+)`),
	}
}

func (s *SectionSplitter) Split(text string) []Section {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	starts := s.boundaries(text)
	if len(starts) < 2 {
		return s.fallbackSplit(text)
	}

	sections := make([]Section, 0, len(starts)-1)
	for i := 0; i < len(starts)-1; i++ {
		chunk := strings.TrimSpace(text[starts[i]:starts[i+1]])
		if chunk == "" {
			continue
		}
		sections = append(sections, s.buildSection(text, starts[i], chunk, i+1))
	}
	return sections
}

func (s *SectionSplitter) boundaries(text string) []int {
	set := map[int]struct{}{0: {}}
	for _, re := range []*regexp.Regexp{s.numbered, s.article, s.capsTitle} {
		for _, loc := range re.FindAllStringIndex(text, -1) {
			set[loc[0]] = struct{}{}
		}
	}
	set[len(text)] = struct{}{}

	out := make([]int, 0, len(set))
	for start := range set {
		out = append(out, start)
	}
	sort.Ints(out)
	return out
}

func (s *SectionSplitter) fallbackSplit(text string) []Section {
	parts := strings.Split(text, "\n\n")
	sections := make([]Section, 0, len(parts))
	offset := 0
	for idx, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			offset += len(part) + 2
			continue
		}
		sections = append(sections, s.buildSection(text, offset, part, idx+1))
		offset += len(part) + 2
	}
	return sections
}

func (s *SectionSplitter) buildSection(fullText string, start int, chunk string, ordinal int) Section {
	lines := strings.Split(chunk, "\n")
	titleLine := strings.TrimSpace(lines[0])
	ref := "Paragraph " + strconv.Itoa(ordinal)
	if match := s.refPattern.FindStringSubmatch(titleLine); len(match) > 1 {
		ref = strings.TrimSpace(match[1])
	} else if strings.HasPrefix(strings.ToLower(titleLine), "section ") || strings.HasPrefix(strings.ToLower(titleLine), "article ") {
		ref = titleLine
	}

	title := titleLine
	if len(title) > 120 {
		title = title[:120]
	}

	page := 1 + strings.Count(fullText[:start], "\f")
	if page < 1 {
		page = 1
	}

	return Section{
		Reference:  ref,
		Title:      title,
		Text:       chunk,
		PageNumber: page,
	}
}
