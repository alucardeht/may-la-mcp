package lang

import (
	"bufio"
	"regexp"
	"strings"
)

var (
	rustFnRe     = regexp.MustCompile(`^\s*(?:pub(?:\(crate\))?\s+)?(?:async\s+)?fn\s+([a-zA-Z_]\w*)\s*[<(]`)
	rustStructRe = regexp.MustCompile(`^\s*(?:pub(?:\(crate\))?\s+)?(?:struct|enum|union)\s+([a-zA-Z_]\w*)`)
	rustTraitRe  = regexp.MustCompile(`^\s*(?:pub(?:\(crate\))?\s+)?trait\s+([a-zA-Z_]\w*)`)
	rustImplRe   = regexp.MustCompile(`^\s*impl(?:<[^>]*>)?\s+(?:(\w+)\s+for\s+)?(\w+)`)
	rustConstRe  = regexp.MustCompile(`^\s*(?:pub(?:\(crate\))?\s+)?(?:const|static)\s+([A-Z_]\w*)\s*:`)
)

func extractRustSymbols(content, filePath string) []Symbol {
	var symbols []Symbol
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0
	currentImpl := ""

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if m := rustImplRe.FindStringSubmatch(line); len(m) > 2 {
			currentImpl = m[2]
			continue
		}

		if m := rustTraitRe.FindStringSubmatch(line); len(m) > 1 {
			symbols = append(symbols, Symbol{
				Name:      m[1],
				Kind:      "interface",
				Line:      lineNum,
				Signature: trimmed,
			})
			continue
		}

		if m := rustStructRe.FindStringSubmatch(line); len(m) > 1 {
			symbols = append(symbols, Symbol{
				Name:      m[1],
				Kind:      "type",
				Line:      lineNum,
				Signature: trimmed,
			})
			continue
		}

		if m := rustFnRe.FindStringSubmatch(line); len(m) > 1 {
			kind := "function"
			parent := ""
			if currentImpl != "" && getIndent(line) > 0 {
				kind = "method"
				parent = currentImpl
			}
			symbols = append(symbols, Symbol{
				Name:      m[1],
				Kind:      kind,
				Line:      lineNum,
				Signature: trimmed,
				Parent:    parent,
			})
			continue
		}

		if m := rustConstRe.FindStringSubmatch(line); len(m) > 1 {
			symbols = append(symbols, Symbol{
				Name:      m[1],
				Kind:      "const",
				Line:      lineNum,
				Signature: trimmed,
			})
		}

		if trimmed == "}" && getIndent(line) == 0 {
			currentImpl = ""
		}
	}

	return symbols
}
