package lang

import (
	"bufio"
	"regexp"
	"strings"
)

var (
	jsFuncRe   = regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?function\s+([a-zA-Z_$]\w*)\s*\(`)
	jsArrowRe  = regexp.MustCompile(`^\s*(?:export\s+)?(?:const|let|var)\s+([a-zA-Z_$]\w*)\s*=\s*(?:async\s+)?(?:\([^)]*\)|[a-zA-Z_$]\w*)\s*=>`)
	jsClassRe  = regexp.MustCompile(`^\s*(?:export\s+)?(?:default\s+)?(?:abstract\s+)?class\s+([a-zA-Z_$]\w*)`)
	jsMethodRe = regexp.MustCompile(`^\s*(?:async\s+)?(?:static\s+)?(?:get\s+|set\s+)?([a-zA-Z_$]\w*)\s*\([^)]*\)\s*[:{]`)
	jsInterfRe = regexp.MustCompile(`^\s*(?:export\s+)?interface\s+([a-zA-Z_$]\w*)`)
	jsTypeRe   = regexp.MustCompile(`^\s*(?:export\s+)?type\s+([a-zA-Z_$]\w*)\s*[=<]`)
	_          = regexp.MustCompile(`^\s*export\s+(?:default\s+)?(?:const|let|var|function|class|type|interface|enum)\s+([a-zA-Z_$]\w*)`)
)

var jsControlKeywords = map[string]bool{
	"if": true, "for": true, "while": true, "switch": true, "catch": true,
}

func extractJSSymbols(content, filePath string) []Symbol {
	var symbols []Symbol
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0
	currentClass := ""

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if m := jsClassRe.FindStringSubmatch(line); len(m) > 1 {
			currentClass = m[1]
			symbols = append(symbols, Symbol{
				Name:      m[1],
				Kind:      "class",
				Line:      lineNum,
				Signature: trimmed,
			})
			continue
		}

		if m := jsInterfRe.FindStringSubmatch(line); len(m) > 1 {
			symbols = append(symbols, Symbol{
				Name:      m[1],
				Kind:      "interface",
				Line:      lineNum,
				Signature: trimmed,
			})
			continue
		}

		if m := jsTypeRe.FindStringSubmatch(line); len(m) > 1 {
			symbols = append(symbols, Symbol{
				Name:      m[1],
				Kind:      "type",
				Line:      lineNum,
				Signature: trimmed,
			})
			continue
		}

		if m := jsFuncRe.FindStringSubmatch(line); len(m) > 1 {
			symbols = append(symbols, Symbol{
				Name:      m[1],
				Kind:      "function",
				Line:      lineNum,
				Signature: trimmed,
			})
			continue
		}

		if m := jsArrowRe.FindStringSubmatch(line); len(m) > 1 {
			symbols = append(symbols, Symbol{
				Name:      m[1],
				Kind:      "function",
				Line:      lineNum,
				Signature: trimmed,
			})
			continue
		}

		if currentClass != "" && getIndent(line) > 0 {
			if m := jsMethodRe.FindStringSubmatch(line); len(m) > 1 {
				name := m[1]
				if !jsControlKeywords[name] {
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

		if trimmed == "}" && getIndent(line) == 0 {
			currentClass = ""
		}
	}

	return symbols
}
