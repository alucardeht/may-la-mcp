package lang

import (
	"bufio"
	"regexp"
	"strings"
)

var (
	goFuncRe   = regexp.MustCompile(`^\s*func\s+([a-zA-Z_]\w*)\s*\(`)
	goMethodRe = regexp.MustCompile(`^\s*func\s+\((\w+)\s+\*?(\w+)\)\s+([a-zA-Z_]\w*)\s*\(`)
	goTypeRe   = regexp.MustCompile(`^\s*type\s+([a-zA-Z_]\w*)\s+(struct|interface)`)
	goConstRe  = regexp.MustCompile(`^\s*(?:const|var)\s+([a-zA-Z_]\w*)\s*[=\s]`)
)

func extractGoSymbols(content, filePath string) []Symbol {
	var symbols []Symbol
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if m := goMethodRe.FindStringSubmatch(line); len(m) > 3 {
			symbols = append(symbols, Symbol{
				Name:      m[3],
				Kind:      "method",
				Line:      lineNum,
				Signature: strings.TrimSpace(line),
				Parent:    m[2],
			})
			continue
		}

		if m := goFuncRe.FindStringSubmatch(line); len(m) > 1 {
			symbols = append(symbols, Symbol{
				Name:      m[1],
				Kind:      "function",
				Line:      lineNum,
				Signature: strings.TrimSpace(line),
			})
			continue
		}

		if m := goTypeRe.FindStringSubmatch(line); len(m) > 2 {
			kind := "type"
			if m[2] == "interface" {
				kind = "interface"
			}
			symbols = append(symbols, Symbol{
				Name:      m[1],
				Kind:      kind,
				Line:      lineNum,
				Signature: strings.TrimSpace(line),
			})
		}

		if m := goConstRe.FindStringSubmatch(line); len(m) > 1 {
			symbols = append(symbols, Symbol{
				Name:      m[1],
				Kind:      "const",
				Line:      lineNum,
				Signature: strings.TrimSpace(line),
			})
		}
	}

	return symbols
}
