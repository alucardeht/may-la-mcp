package lang

import (
	"bufio"
	"regexp"
	"strings"
)

var (
	javaClassRe  = regexp.MustCompile(`\b(?:public|private|protected)?\s*(?:abstract\s+)?(?:class|interface|enum)\s+([a-zA-Z_]\w*)`)
	javaMethodRe = regexp.MustCompile(`^\s*(?:public|private|protected)?\s*(?:static\s+)?(?:final\s+)?(?:synchronized\s+)?(?:abstract\s+)?\w[\w<>\[\],\s]*\s+([a-zA-Z_]\w*)\s*\(`)
)

var javaControlKeywords = map[string]bool{
	"if": true, "for": true, "while": true, "switch": true,
	"catch": true, "return": true, "new": true,
}

func extractJavaSymbols(content, filePath string) []Symbol {
	var symbols []Symbol
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0
	currentClass := ""

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if m := javaClassRe.FindStringSubmatch(line); len(m) > 1 {
			currentClass = m[1]
			kind := "class"
			if strings.Contains(line, "interface") {
				kind = "interface"
			}
			symbols = append(symbols, Symbol{
				Name:      m[1],
				Kind:      kind,
				Line:      lineNum,
				Signature: trimmed,
			})
			continue
		}

		if m := javaMethodRe.FindStringSubmatch(line); len(m) > 1 {
			name := m[1]
			if !javaControlKeywords[name] {
				symbols = append(symbols, Symbol{
					Name:      name,
					Kind:      "method",
					Line:      lineNum,
					Signature: trimmed,
					Parent:    currentClass,
				})
			}
		}
	}

	return symbols
}
