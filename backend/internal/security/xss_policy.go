package security

import (
	"github.com/microcosm-cc/bluemonday"
)

// XSSPolicy defines per-field HTML sanitization rules.
type XSSPolicy struct {
	AllowedTags       []string
	AllowedAttributes map[string][]string
	AllowedProtocols  []string
	StripAll          bool
}

// StrictPolicy removes ALL HTML — used for plain text fields.
var StrictXSSPolicy = XSSPolicy{StripAll: true}

// RichTextPolicy allows limited safe HTML from rich text editors.
var RichTextXSSPolicy = XSSPolicy{
	AllowedTags:       []string{"p", "br", "strong", "em", "ul", "ol", "li", "a", "code", "pre", "blockquote"},
	AllowedAttributes: map[string][]string{"a": {"href", "title", "rel"}},
	AllowedProtocols:  []string{"http", "https", "mailto"},
}

// Sanitize applies the policy to input HTML and returns sanitized output.
func (p *XSSPolicy) Sanitize(input string) string {
	if p.StripAll {
		policy := bluemonday.StrictPolicy()
		return policy.Sanitize(input)
	}

	policy := bluemonday.NewPolicy()

	for _, tag := range p.AllowedTags {
		policy.AllowElements(tag)
	}

	for tag, attrs := range p.AllowedAttributes {
		policy.AllowAttrs(attrs...).OnElements(tag)
	}

	for _, proto := range p.AllowedProtocols {
		policy.AllowURLSchemes(proto)
	}

	// Force rel="nofollow noopener" on links
	policy.RequireNoFollowOnLinks(true)

	return policy.Sanitize(input)
}

// FieldXSSPolicies maps field paths to their XSS policies.
// Any field not listed here defaults to StrictPolicy.
var FieldXSSPolicies = map[string]*XSSPolicy{
	"description":          &RichTextXSSPolicy,
	"comment.content":      &RichTextXSSPolicy,
	"alert.comments.content": &RichTextXSSPolicy,
	"note.body":            &RichTextXSSPolicy,
	"remediation.notes":    &RichTextXSSPolicy,
}

// GetFieldPolicy returns the XSS policy for a given field path.
// Defaults to StrictPolicy for unlisted fields.
func GetFieldPolicy(fieldPath string) *XSSPolicy {
	if policy, ok := FieldXSSPolicies[fieldPath]; ok {
		return policy
	}
	return &StrictXSSPolicy
}

// SanitizeField sanitizes a field value using the appropriate XSS policy.
func SanitizeField(fieldPath, value string) string {
	policy := GetFieldPolicy(fieldPath)
	return policy.Sanitize(value)
}
