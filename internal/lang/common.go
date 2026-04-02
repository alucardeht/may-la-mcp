package lang

import (
	"path/filepath"
	"strings"
)

type Symbol struct {
	Name      string
	Kind      string
	Line      int
	Signature string
	Parent    string
}

func LanguageID(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".go":
		return "go"
	case ".js", ".jsx", ".mjs":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".py":
		return "python"
	case ".java":
		return "java"
	case ".rs":
		return "rust"
	case ".c", ".h":
		return "c"
	case ".cpp", ".hpp", ".cc", ".cxx":
		return "cpp"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".swift":
		return "swift"
	case ".kt":
		return "kotlin"
	default:
		return ""
	}
}

func ExtractSymbols(content string, filePath string) []Symbol {
	lang := LanguageID(filePath)
	switch lang {
	case "go":
		return extractGoSymbols(content, filePath)
	case "javascript", "typescript":
		return extractJSSymbols(content, filePath)
	case "python":
		return extractPythonSymbols(content, filePath)
	case "java", "kotlin":
		return extractJavaSymbols(content, filePath)
	case "rust":
		return extractRustSymbols(content, filePath)
	case "c", "cpp":
		return extractCSymbols(content, filePath)
	default:
		return nil
	}
}

func MatchesQuery(name, query string) bool {
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(name), strings.ToLower(query))
}

func FindEnclosingFunction(lines []string, lineNum int) string {
	if lineNum < 1 || lineNum > len(lines) {
		return ""
	}

	braceDepth := 0
	for i := lineNum - 1; i >= 0 && i >= lineNum-200; i-- {
		line := lines[i]
		braceDepth += strings.Count(line, "}") - strings.Count(line, "{")

		for _, pattern := range funcPatterns {
			matches := pattern.FindStringSubmatch(line)
			if len(matches) > 0 {
				name := extractFuncName(matches)
				if name != "" && braceDepth <= 0 {
					return name
				}
			}
		}
	}
	return ""
}

func FindFunctionBounds(lines []string, lineNum int) (int, int) {
	if lineNum < 1 || lineNum > len(lines) {
		return 0, 0
	}

	startLine := 0
	braceDepth := 0
	for i := lineNum - 1; i >= 0 && i >= lineNum-200; i-- {
		line := lines[i]
		braceDepth += strings.Count(line, "}") - strings.Count(line, "{")

		for _, pattern := range funcPatterns {
			if pattern.MatchString(line) && braceDepth <= 0 {
				startLine = i + 1
				break
			}
		}
		if startLine > 0 {
			break
		}
	}

	if startLine == 0 {
		return 0, 0
	}

	depth := 0
	started := false
	for i := startLine - 1; i < len(lines); i++ {
		line := lines[i]
		for _, ch := range line {
			if ch == '{' {
				depth++
				started = true
			} else if ch == '}' {
				depth--
				if started && depth == 0 {
					return startLine, i + 1
				}
			}
		}
	}

	if !started {
		baseIndent := getIndent(lines[startLine-1])
		for i := startLine; i < len(lines); i++ {
			line := lines[i]
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			if getIndent(line) <= baseIndent && trimmed != "" {
				return startLine, i
			}
		}
		return startLine, len(lines)
	}

	return startLine, len(lines)
}

func getIndent(line string) int {
	count := 0
	for _, ch := range line {
		if ch == ' ' {
			count++
		} else if ch == '\t' {
			count += 4
		} else {
			break
		}
	}
	return count
}

func extractFuncName(matches []string) string {
	for i := len(matches) - 1; i >= 1; i-- {
		if matches[i] != "" && matches[i] != "func" && matches[i] != "function" && matches[i] != "def" && matches[i] != "fn" {
			return matches[i]
		}
	}
	return ""
}
