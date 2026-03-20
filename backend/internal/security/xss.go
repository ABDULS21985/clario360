package security

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

// compileXSSPatterns compiles XSS detection patterns.
func compileXSSPatterns() []*compiledPattern {
	patterns := []struct {
		pattern  string
		category string
	}{
		// Script tags
		{`<\s*script[\s>]`, "script_tag"},
		// Script close
		{`<\s*/\s*script\s*>`, "script_close"},
		// Event handlers
		{`\bon\w+\s*=`, "event_handler"},
		// JavaScript URI
		{`javascript\s*:`, "javascript_uri"},
		// Data URI with script
		{`data\s*:\s*text/html`, "data_uri"},
		// VBScript URI
		{`vbscript\s*:`, "vbscript_uri"},
		// CSS expression
		{`expression\s*\(`, "css_expression"},
		// SVG onload
		{`<\s*svg[^>]*\bonload\s*=`, "svg_onload"},
		// Iframe
		{`<\s*iframe[\s>]`, "iframe"},
		// Object/Embed/Applet
		{`<\s*(object|embed|applet)[\s>]`, "object_embed"},
		// Base tag hijacking
		{`<\s*base[\s>]`, "base_tag"},
		// Meta redirect
		{`<\s*meta[^>]+http-equiv\s*=\s*["']?refresh`, "meta_redirect"},
		// CSS import
		{`@import\s`, "css_import"},
		// HTML entity evasion
		{`&#x?[0-9a-f]+;`, "html_entity_evasion"},
		// Template injection
		{`\{\{.*\}\}`, "template_injection"},
	}

	compiled := make([]*compiledPattern, 0, len(patterns))
	for _, p := range patterns {
		r, err := regexp.Compile("(?i)" + p.pattern)
		if err != nil {
			continue
		}
		compiled = append(compiled, &compiledPattern{
			regex:    r,
			category: p.category,
		})
	}
	return compiled
}

// ValidateNoXSS checks the input string against known XSS patterns.
func (s *Sanitizer) ValidateNoXSS(input string) error {
	if input == "" {
		return nil
	}

	// Check raw input
	for _, pattern := range s.xssPatterns {
		if pattern.regex.MatchString(input) {
			return &InjectionError{
				Category: pattern.category,
				Type:     "xss",
			}
		}
	}

	// Check URL-decoded version for encoded payloads
	decoded, err := url.QueryUnescape(input)
	if err == nil && decoded != input {
		for _, pattern := range s.xssPatterns {
			if pattern.regex.MatchString(decoded) {
				return &InjectionError{
					Category: pattern.category + "_encoded",
					Type:     "xss",
				}
			}
		}
	}

	return nil
}

// ValidateNoXSSBatch validates multiple fields for XSS and returns the first error.
func (s *Sanitizer) ValidateNoXSSBatch(fields map[string]string) error {
	for fieldName, value := range fields {
		if err := s.ValidateNoXSS(value); err != nil {
			if injErr, ok := err.(*InjectionError); ok {
				injErr.Field = fieldName
				return injErr
			}
			return fmt.Errorf("field %q: %w", fieldName, err)
		}
	}
	return nil
}

// HTMLEncode encodes input for safe embedding in HTML contexts.
func HTMLEncode(input string) string {
	return html.EscapeString(input)
}

// JSEncode encodes input for safe embedding in JavaScript string literals.
func JSEncode(input string) string {
	var b strings.Builder
	b.Grow(len(input) * 2)

	for i := 0; i < len(input); {
		r, size := utf8.DecodeRuneInString(input[i:])
		i += size

		switch r {
		case '\'':
			b.WriteString(`\'`)
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '/':
			b.WriteString(`\/`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '<':
			b.WriteString(`\u003c`)
		case '>':
			b.WriteString(`\u003e`)
		case '&':
			b.WriteString(`\u0026`)
		default:
			if r >= 0x20 && r <= 0x7E {
				b.WriteRune(r)
			} else {
				b.WriteString(fmt.Sprintf(`\u%04x`, r))
			}
		}
	}

	return b.String()
}

// URLEncode encodes input for safe embedding in URL parameters.
func URLEncode(input string) string {
	return url.QueryEscape(input)
}

// CSSEncode encodes input for safe embedding in CSS values.
func CSSEncode(input string) string {
	var b strings.Builder
	b.Grow(len(input) * 7)

	for _, r := range input {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteString(fmt.Sprintf(`\%06x`, r))
		}
	}

	return b.String()
}
