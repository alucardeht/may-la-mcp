package intel

import (
	"bufio"
	"bytes"
	"regexp"
	"strings"
)

type Context struct {
	FilePath       string
	Line           int
	StartLine      int
	EndLine        int
	Content        string
	ParentFunction string
	ParentClass    string
	Imports        []string
	Scope          string
	IndentLevel    int
	SemanticInfo   map[string]string
}

func ExtractContext(fileContent string, lineNum int, radius int) Context {
	lines := strings.Split(fileContent, "\n")

	if lineNum < 1 || lineNum > len(lines) {
		return Context{Line: lineNum}
	}

	startLine := lineNum - radius
	if startLine < 1 {
		startLine = 1
	}

	endLine := lineNum + radius
	if endLine > len(lines) {
		endLine = len(lines)
	}

	contextLines := lines[startLine-1 : endLine]
	contextContent := strings.Join(contextLines, "\n")

	imports := extractImports(lines)
	parentFunc := findParentFunction(lines, lineNum)
	parentClass := findParentClass(lines, lineNum)
	indentLevel := getIndentLevel(lines[lineNum-1])
	scope := determineScope(lines, lineNum, parentFunc, parentClass)

	return Context{
		Line:           lineNum,
		StartLine:      startLine,
		EndLine:        endLine,
		Content:        contextContent,
		ParentFunction: parentFunc,
		ParentClass:    parentClass,
		Imports:        imports,
		Scope:          scope,
		IndentLevel:    indentLevel,
		SemanticInfo: map[string]string{
			"parent_function": parentFunc,
			"parent_class":    parentClass,
			"scope":           scope,
		},
	}
}

func extractImports(lines []string) []string {
	var imports []string

	importPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^\s*import\s+["']([^"']+)["']`),
		regexp.MustCompile(`^\s*from\s+["']([^"']+)["']\s+import`),
		regexp.MustCompile(`^\s*package\s+([a-zA-Z0-9_]+)`),
		regexp.MustCompile(`^\s*use\s+([a-zA-Z0-9_\\:]+)`),
	}

	for _, line := range lines {
		for _, pattern := range importPatterns {
			matches := pattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				imports = append(imports, matches[1])
			}
		}

		if len(imports) >= 10 {
			break
		}
	}

	return imports
}

func findParentFunction(lines []string, lineNum int) string {
	funcPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^\s*(func|function|def)\s+([a-zA-Z_][a-zA-Z0-9_]*)`),
		regexp.MustCompile(`^\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\([^)]*\)\s*{`),
		regexp.MustCompile(`^\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*:\s*function`),
	}

	braceDepth := 0
	functionLine := -1
	var functionName string

	for i := lineNum - 1; i >= 0 && i >= lineNum-100; i-- {
		line := lines[i]

		braceDepth += strings.Count(line, "}") - strings.Count(line, "{")

		for _, pattern := range funcPatterns {
			matches := pattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				if matches[1] == "func" || matches[1] == "function" || matches[1] == "def" {
					functionName = matches[2]
				} else {
					functionName = matches[1]
				}

				functionLine = i
				break
			}
		}

		if functionLine != -1 && braceDepth <= 0 {
			return functionName
		}
	}

	return ""
}

func findParentClass(lines []string, lineNum int) string {
	classPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^\s*(class|interface|struct|type)\s+([a-zA-Z_][a-zA-Z0-9_]*)`),
	}

	braceDepth := 0
	classLine := -1
	var className string

	for i := lineNum - 1; i >= 0 && i >= lineNum-100; i-- {
		line := lines[i]

		braceDepth += strings.Count(line, "}") - strings.Count(line, "{")

		for _, pattern := range classPatterns {
			matches := pattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				className = matches[2]
				classLine = i
				break
			}
		}

		if classLine != -1 && braceDepth <= 0 {
			return className
		}
	}

	return ""
}

func getIndentLevel(line string) int {
	if line == "" {
		return 0
	}

	count := 0
	for _, char := range line {
		if char == ' ' {
			count++
		} else if char == '\t' {
			count += 4
		} else {
			break
		}
	}

	return count / 4
}

func determineScope(lines []string, lineNum int, parentFunc string, parentClass string) string {
	if parentFunc != "" && parentClass != "" {
		return "method"
	}
	if parentFunc != "" {
		return "function"
	}
	if parentClass != "" {
		return "class"
	}

	line := strings.TrimSpace(lines[lineNum-1])
	if strings.HasPrefix(line, "export") || strings.HasPrefix(line, "public") {
		return "public"
	}
	if strings.HasPrefix(line, "private") {
		return "private"
	}

	return "module"
}

func ExtractContextAroundSymbol(fileContent string, symbolName string, lineHint int) Context {
	lines := strings.Split(fileContent, "\n")

	startLine := 1
	if lineHint > 0 && lineHint <= len(lines) {
		startLine = lineHint
	}

	symbolPattern := regexp.MustCompile(`(func|function|class|interface|type|const|var|let)\s+` + regexp.QuoteMeta(symbolName))

	targetLine := -1
	for i := startLine - 1; i < len(lines); i++ {
		if symbolPattern.MatchString(lines[i]) {
			targetLine = i + 1
			break
		}
	}

	if targetLine == -1 {
		for i := 0; i < len(lines); i++ {
			if symbolPattern.MatchString(lines[i]) {
				targetLine = i + 1
				break
			}
		}
	}

	if targetLine == -1 {
		targetLine = startLine
	}

	return ExtractContext(fileContent, targetLine, 10)
}

func MergeContexts(contexts ...Context) Context {
	if len(contexts) == 0 {
		return Context{}
	}

	merged := contexts[0]

	for i := 1; i < len(contexts); i++ {
		if contexts[i].StartLine < merged.StartLine {
			merged.StartLine = contexts[i].StartLine
		}
		if contexts[i].EndLine > merged.EndLine {
			merged.EndLine = contexts[i].EndLine
		}

		if contexts[i].ParentFunction != "" && merged.ParentFunction == "" {
			merged.ParentFunction = contexts[i].ParentFunction
		}
		if contexts[i].ParentClass != "" && merged.ParentClass == "" {
			merged.ParentClass = contexts[i].ParentClass
		}

		for _, imp := range contexts[i].Imports {
			found := false
			for _, existing := range merged.Imports {
				if existing == imp {
					found = true
					break
				}
			}
			if !found {
				merged.Imports = append(merged.Imports, imp)
			}
		}
	}

	return merged
}

func ContextToString(ctx Context) string {
	var buf bytes.Buffer
	buf.WriteString("Context Information\n")
	buf.WriteString("===================\n\n")

	buf.WriteString("Location:\n")
	buf.WriteString("  Line: " + string(rune(ctx.Line)) + "\n")
	buf.WriteString("  Range: " + string(rune(ctx.StartLine)) + "-" + string(rune(ctx.EndLine)) + "\n")
	buf.WriteString("  IndentLevel: " + string(rune(ctx.IndentLevel)) + "\n\n")

	if ctx.ParentClass != "" {
		buf.WriteString("  Class: " + ctx.ParentClass + "\n")
	}
	if ctx.ParentFunction != "" {
		buf.WriteString("  Function: " + ctx.ParentFunction + "\n")
	}
	if ctx.Scope != "" {
		buf.WriteString("  Scope: " + ctx.Scope + "\n")
	}

	if len(ctx.Imports) > 0 {
		buf.WriteString("\nImports:\n")
		for _, imp := range ctx.Imports {
			buf.WriteString("  - " + imp + "\n")
		}
	}

	if ctx.Content != "" {
		buf.WriteString("\nContent:\n")
		scanner := bufio.NewScanner(strings.NewReader(ctx.Content))
		lineNum := ctx.StartLine
		for scanner.Scan() {
			buf.WriteString("  " + string(rune(lineNum)) + " | " + scanner.Text() + "\n")
			lineNum++
		}
	}

	return buf.String()
}
