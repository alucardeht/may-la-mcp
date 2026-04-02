package lang

import (
	"bufio"
	"regexp"
	"strings"
)

var (
	pyFuncRe  = regexp.MustCompile(`^\s*(?:async\s+)?def\s+([a-zA-Z_]\w*)\s*\(`)
	pyClassRe = regexp.MustCompile(`^\s*class\s+([a-zA-Z_]\w*)`)
)

func extractPythonSymbols(content, filePath string) []Symbol {
	var symbols []Symbol
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0
	currentClass := ""
	classIndent := -1

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		indent := getIndent(line)

		if classIndent >= 0 && indent <= classIndent && trimmed != "" {
			currentClass = ""
			classIndent = -1
		}

		if m := pyClassRe.FindStringSubmatch(line); len(m) > 1 {
			currentClass = m[1]
			classIndent = indent
			symbols = append(symbols, Symbol{
				Name:      m[1],
				Kind:      "class",
				Line:      lineNum,
				Signature: trimmed,
			})
			continue
		}

		if m := pyFuncRe.FindStringSubmatch(line); len(m) > 1 {
			kind := "function"
			parent := ""
			if currentClass != "" && indent > classIndent {
				kind = "method"
				parent = currentClass
			}
			symbols = append(symbols, Symbol{
				Name:      m[1],
				Kind:      kind,
				Line:      lineNum,
				Signature: trimmed,
				Parent:    parent,
			})
		}
	}

	return symbols
}
