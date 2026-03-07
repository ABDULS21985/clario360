package templates

import "regexp"

var (
	ActionMarkerPattern = regexp.MustCompile(`(?i)(?:ACTION|TASK|TODO):\s*(.+)`)
	WillPattern         = regexp.MustCompile(`(?i)\b([A-Z][a-z]+(?:\s+[A-Z][a-z]+)?)\s+(?:will|shall|to|is to|agreed to)\s+(.+)`)
	AgreedPattern       = regexp.MustCompile(`(?i)(?:It was agreed|It was decided|The committee directed)\s+that\s+(.+)`)
	DatePattern         = regexp.MustCompile(`(?i)(?:by|before|due|deadline[:]?)\s+(\d{1,2}[\/\-]\d{1,2}[\/\-]\d{2,4}|\w+\s+\d{1,2},?\s+\d{4})`)
)
