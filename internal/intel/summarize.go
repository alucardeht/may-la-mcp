package intel

import (
	"regexp"
	"strings"
)

func Summarize(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}

	contentType := detectContentType(content)

	switch contentType {
	case ContentTypeCode:
		return summarizeCode(content, maxLen)
	case ContentTypeText:
		return summarizeText(content, maxLen)
	case ContentTypeList:
		return summarizeList(content, maxLen)
	default:
		return summarizeGeneric(content, maxLen)
	}
}

func detectContentType(content string) string {
	lines := strings.Split(content, "\n")
	codeLines := 0
	listLines := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isCodeLine(trimmed) {
			codeLines++
		}
		if isListLine(trimmed) {
			listLines++
		}
	}

	if len(lines) > 0 {
		if float64(codeLines)/float64(len(lines)) > 0.6 {
			return ContentTypeCode
		}
		if float64(listLines)/float64(len(lines)) > 0.5 {
			return ContentTypeList
		}
	}

	return ContentTypeText
}

func isCodeLine(line string) bool {
	if strings.Contains(line, "{") || strings.Contains(line, "}") {
		return true
	}
	if strings.Contains(line, "func ") || strings.Contains(line, "class ") {
		return true
	}
	if strings.Contains(line, "const ") || strings.Contains(line, "var ") {
		return true
	}
	if strings.Contains(line, "import ") || strings.Contains(line, "package ") {
		return true
	}
	return false
}

func isListLine(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "-") ||
		strings.HasPrefix(strings.TrimSpace(line), "*") ||
		strings.HasPrefix(strings.TrimSpace(line), "+") ||
		regexp.MustCompile(`^\d+\.`).MatchString(strings.TrimSpace(line))
}

func summarizeCode(content string, maxLen int) string {
	structures := extractCodeStructures(content)

	if len(structures) == 0 {
		return summarizeGeneric(content, maxLen)
	}

	summary := strings.Builder{}
	summary.WriteString("Code Structure:\n")

	for _, s := range structures {
		summary.WriteString("  - " + s + "\n")
		if summary.Len() > maxLen {
			break
		}
	}

	result := summary.String()
	if len(result) > maxLen {
		result = result[:maxLen] + "..."
	}

	return result
}

func extractCodeStructures(content string) []string {
	var structures []string
	lines := strings.Split(content, "\n")

	funcPattern := regexp.MustCompile(`^\s*(func|function|def|class|interface|type)\s+([a-zA-Z_][a-zA-Z0-9_]*)`)

	for _, line := range lines {
		matches := funcPattern.FindStringSubmatch(line)
		if len(matches) > 0 {
			structures = append(structures, matches[1]+" "+matches[2])
		}
	}

	return structures
}

func summarizeText(content string, maxLen int) string {
	sentences := extractSentences(content)

	if len(sentences) == 0 {
		return summarizeGeneric(content, maxLen)
	}

	summary := strings.Builder{}
	for _, sentence := range sentences {
		if summary.Len()+len(sentence) > maxLen {
			break
		}
		summary.WriteString(sentence + " ")
	}

	result := strings.TrimSpace(summary.String())
	if result == "" {
		return summarizeGeneric(content, maxLen)
	}

	return result
}

func extractSentences(content string) []string {
	sentences := strings.FieldsFunc(content, func(r rune) bool {
		return r == '.' || r == '!' || r == '?'
	})

	var result []string
	for _, s := range sentences {
		trimmed := strings.TrimSpace(s)
		if len(trimmed) > 10 {
			result = append(result, trimmed)
		}
	}

	return result
}

func summarizeList(content string, maxLen int) string {
	items := extractListItems(content)

	if len(items) == 0 {
		return summarizeGeneric(content, maxLen)
	}

	summary := strings.Builder{}
	summary.WriteString("Items: " + string(rune(len(items))) + "\n")

	groupedItems := groupListItems(items)
	for group, count := range groupedItems {
		if summary.Len() > maxLen {
			break
		}
		summary.WriteString("  " + group + ": " + string(rune(count)) + "\n")
	}

	result := summary.String()
	if len(result) > maxLen {
		result = result[:maxLen] + "..."
	}

	return result
}

func extractListItems(content string) []string {
	var items []string
	lines := strings.Split(content, "\n")

	listPattern := regexp.MustCompile(`^\s*[-*+]\s+(.+)$|^\s*\d+\.\s+(.+)$`)

	for _, line := range lines {
		matches := listPattern.FindStringSubmatch(line)
		if len(matches) > 0 {
			item := matches[1]
			if item == "" {
				item = matches[2]
			}
			if item != "" {
				items = append(items, strings.TrimSpace(item))
			}
		}
	}

	return items
}

func groupListItems(items []string) map[string]int {
	groups := make(map[string]int)

	for _, item := range items {
		prefix := extractItemPrefix(item)
		if prefix != "" {
			groups[prefix]++
		}
	}

	return groups
}

func extractItemPrefix(item string) string {
	parts := strings.Fields(item)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func summarizeGeneric(content string, maxLen int) string {
	words := strings.Fields(content)
	summary := strings.Builder{}

	for _, word := range words {
		if summary.Len()+len(word) > maxLen {
			summary.WriteString("...")
			break
		}
		if summary.Len() > 0 {
			summary.WriteString(" ")
		}
		summary.WriteString(word)
	}

	return summary.String()
}
