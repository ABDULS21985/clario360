package jira

func BuildADF(entity map[string]any, viewURL string) map[string]any {
	title := firstNonEmpty(stringValue(entity["title"]), "Clario 360 item")
	summary := firstNonEmpty(extractSummary(entity), stringValue(entity["description"]))
	severity := firstNonEmpty(stringValue(entity["severity"]), stringValue(entity["priority"]))
	status := firstNonEmpty(stringValue(entity["status"]), "new")

	content := []map[string]any{
		paragraphNode([]map[string]any{{"type": "text", "text": title, "marks": []map[string]any{{"type": "strong"}}}}),
		paragraphNode([]map[string]any{{"type": "text", "text": "Severity: " + severity + " | Status: " + status}}),
	}
	if summary != "" {
		content = append(content, paragraphNode([]map[string]any{{"type": "text", "text": summary}}))
	}
	if viewURL != "" {
		content = append(content, paragraphNode([]map[string]any{
			{"type": "text", "text": "View in Clario 360: "},
			{"type": "text", "text": viewURL, "marks": []map[string]any{{"type": "link", "attrs": map[string]any{"href": viewURL}}}},
		}))
	}
	return map[string]any{
		"type":    "doc",
		"version": 1,
		"content": content,
	}
}

func paragraphNode(content []map[string]any) map[string]any {
	return map[string]any{
		"type":    "paragraph",
		"content": content,
	}
}
