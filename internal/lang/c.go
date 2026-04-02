package lang

import (
	"bufio"
	"regexp"
	"strings"
)

var (
	cFuncRe   = regexp.MustCompile(`^(?:static\s+)?(?:inline\s+)?(?:const\s+)?(?:unsigned\s+)?(?:signed\s+)?\w[\w*\s]*\s+\*?([a-zA-Z_]\w*)\s*\(`)
	cStructRe = regexp.MustCompile(`^\s*(?:typedef\s+)?(?:struct|union|enum)\s+([a-zA-Z_]\w*)`)
	cDefineRe = regexp.MustCompile(`^\s*#define\s+([A-Z_]\w*)`)
)

var cControlKeywords = map[string]bool{
	"if": true, "for": true, "while": true, "switch": true,
	"return": true, "sizeof": true,
}

func extractCSymbols(content, filePath string) []Symbol {
	var symbols []Symbol
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if m := cStructRe.FindStringSubmatch(line); len(m) > 1 {
			symbols = append(symbols, Symbol{
				Name:      m[1],
				Kind:      "type",
				Line:      lineNum,
				Signature: trimmed,
			})
			continue
		}

		if m := cDefineRe.FindStringSubmatch(line); len(m) > 1 {
			symbols = append(symbols, Symbol{
				Name:      m[1],
				Kind:      "const",
				Line:      lineNum,
				Signature: trimmed,
			})
			continue
		}

		if getIndent(line) == 0 {
			if m := cFuncRe.FindStringSubmatch(line); len(m) > 1 {
				name := m[1]
				if !cControlKeywords[name] {
					symbols = append(symbols, Symbol{
						Name:      name,
						Kind:      "function",
						Line:      lineNum,
						Signature: trimmed,
					})
				}
			}
		}
	}

	return symbols
}
