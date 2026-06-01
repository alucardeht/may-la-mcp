package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/alucardeht/may-la-mcp/internal/encoding"
	"github.com/alucardeht/may-la-mcp/internal/gitignore"
	"github.com/alucardeht/may-la-mcp/internal/lang"
)

type SymbolsTool struct{}

func (t *SymbolsTool) Name() string { return "symbols" }

func (t *SymbolsTool) Description() string {
	return "Extract code structure (functions, types, classes, methods) from files or directories."
}

func (t *SymbolsTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "File or directory path"
			},
			"kind": {
				"type": "string",
				"description": "Filter by symbol kind: 'function', 'class', 'type', 'interface', 'method', 'const'",
				"enum": ["function", "class", "type", "interface", "method", "const"]
			},
			"query": {
				"type": "string",
				"description": "Filter by name (case-insensitive substring)"
			},
			"max_results": {
				"type": "integer",
				"description": "Maximum number of results (default 200)",
				"default": 200
			}
		},
		"required": ["path"]
	}`)
}

func (t *SymbolsTool) Annotations() map[string]bool {
	return map[string]bool{
		"readOnlyHint":    true,
		"destructiveHint": false,
		"idempotentHint":  true,
		"openWorldHint":   false,
	}
}

func (t *SymbolsTool) Execute(ctx context.Context, input json.RawMessage) (interface{}, error) {
	var params struct {
		Path       string `json:"path"`
		Kind       string `json:"kind"`
		Query      string `json:"query"`
		MaxResults int    `json:"max_results"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}
	if params.Path == "" {
		return nil, fmt.Errorf("path is required")
	}
	if params.MaxResults <= 0 {
		params.MaxResults = 200
	}

	info, err := os.Stat(params.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path not found: %s", params.Path)
		}
		return nil, fmt.Errorf("cannot access path: %w", err)
	}

	cwd, _ := os.Getwd()

	if !info.IsDir() {
		return extractSymbolsFromFile(ctx, params.Path, params.Kind, params.Query, params.MaxResults, cwd)
	}

	return extractSymbolsFromDir(ctx, params.Path, params.Kind, params.Query, params.MaxResults, cwd)
}

func extractSymbolsFromFile(ctx context.Context, path, kindFilter, query string, maxResults int, cwd string) (string, error) {
	content, err := encoding.ReadFileAsUTF8(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", path)
		}
		return "", fmt.Errorf("cannot read file: %w", err)
	}

	if isBinaryContent(content) {
		return fmt.Sprintf("# Symbols: %s\n\n(binary file, skipped)\n", toRelativePath(path, cwd)), nil
	}

	symbols := lang.ExtractSymbols(content, path)
	symbols = filterSymbols(symbols, kindFilter, query)
	if len(symbols) > maxResults {
		symbols = symbols[:maxResults]
	}

	displayPath := toRelativePath(path, cwd)

	if len(symbols) == 0 {
		return fmt.Sprintf("# Symbols: %s\n\nNo symbols found.\n", displayPath), nil
	}

	return formatSingleFileSymbols(displayPath, symbols), nil
}

func formatSingleFileSymbols(displayPath string, symbols []lang.Symbol) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Symbols: %s\n\n", displayPath)

	grouped := groupSymbolsByKind(symbols)

	kindOrder := []string{"type", "interface", "class", "const", "function", "method"}
	kindLabels := map[string]string{
		"type":      "Types",
		"interface": "Interfaces",
		"class":     "Classes",
		"const":     "Constants",
		"function":  "Functions",
		"method":    "Methods",
	}

	methodParents := make(map[string][]lang.Symbol)
	var standaloneMethods []lang.Symbol

	if methods, ok := grouped["method"]; ok {
		for _, m := range methods {
			if m.Parent != "" {
				methodParents[m.Parent] = append(methodParents[m.Parent], m)
			} else {
				standaloneMethods = append(standaloneMethods, m)
			}
		}
	}

	for _, kind := range kindOrder {
		if kind == "method" {
			continue
		}
		syms, ok := grouped[kind]
		if !ok {
			continue
		}

		label := kindLabels[kind]
		if label == "" {
			if len(kind) > 0 {
				label = strings.ToUpper(kind[:1]) + kind[1:] + "s"
			} else {
				label = "Symbols"
			}
		}
		fmt.Fprintf(&sb, "## %s\n", label)
		for _, sym := range syms {
			sig := sym.Signature
			if sig == "" {
				sig = sym.Name
			}
			fmt.Fprintf(&sb, "- %-44s [line %d]\n", sig, sym.Line)

			if methods, hasChildren := methodParents[sym.Name]; hasChildren {
				parentLabel := "Methods (" + sym.Name + ")"
				fmt.Fprintf(&sb, "\n## %s\n", parentLabel)
				for _, m := range methods {
					msig := m.Signature
					if msig == "" {
						msig = m.Name
					}
					fmt.Fprintf(&sb, "- %-44s [line %d]\n", msig, m.Line)
				}
				sb.WriteString("\n")
				delete(methodParents, sym.Name)
			}
		}
		sb.WriteString("\n")
	}

	if len(standaloneMethods) > 0 {
		sb.WriteString("## Methods\n")
		for _, m := range standaloneMethods {
			sig := m.Signature
			if sig == "" {
				sig = m.Name
			}
			fmt.Fprintf(&sb, "- %-44s [line %d]\n", sig, m.Line)
		}
		sb.WriteString("\n")
	}

	for parent, methods := range methodParents {
		fmt.Fprintf(&sb, "## Methods (%s)\n", parent)
		for _, m := range methods {
			sig := m.Signature
			if sig == "" {
				sig = m.Name
			}
			fmt.Fprintf(&sb, "- %-44s [line %d]\n", sig, m.Line)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func groupSymbolsByKind(symbols []lang.Symbol) map[string][]lang.Symbol {
	grouped := make(map[string][]lang.Symbol)
	for _, sym := range symbols {
		grouped[sym.Kind] = append(grouped[sym.Kind], sym)
	}
	return grouped
}

type fileSymbols struct {
	path    string
	relPath string
	symbols []lang.Symbol
}

func extractSymbolsFromDir(ctx context.Context, dir, kindFilter, query string, maxResults int, cwd string) (string, error) {
	matcher := gitignore.New(dir)
	var fileResults []fileSymbols
	totalSymbols := 0

	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if ctx.Err() != nil {
			return filepath.SkipAll
		}
		if err != nil || info == nil {
			return nil
		}
		if path == dir {
			return nil
		}

		relPath, _ := filepath.Rel(dir, path)
		relPath = filepath.ToSlash(relPath)
		if matcher.IsIgnored(relPath, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() || !gitignore.IsSourceFile(path) || lang.LanguageID(path) == "" {
			return nil
		}

		if totalSymbols >= maxResults {
			return filepath.SkipAll
		}

		content, err := encoding.ReadFileAsUTF8(path)
		if err != nil || isBinaryContent(content) {
			return nil
		}

		symbols := lang.ExtractSymbols(content, path)
		symbols = filterSymbols(symbols, kindFilter, query)
		if len(symbols) == 0 {
			return nil
		}

		remaining := maxResults - totalSymbols
		if len(symbols) > remaining {
			symbols = symbols[:remaining]
		}
		totalSymbols += len(symbols)

		displayRel, _ := filepath.Rel(dir, path)
		fileResults = append(fileResults, fileSymbols{
			path:    path,
			relPath: displayRel,
			symbols: symbols,
		})

		return nil
	})

	sort.Slice(fileResults, func(i, j int) bool {
		return fileResults[i].relPath < fileResults[j].relPath
	})

	displayDir := toRelativePath(dir, cwd)

	if len(fileResults) == 0 {
		return fmt.Sprintf("# Symbols: %s/\n\nNo symbols found.\n", displayDir), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Symbols: %s/ (%d files)\n\n", displayDir, len(fileResults))

	for _, fr := range fileResults {
		fmt.Fprintf(&sb, "## %s\n", fr.relPath)
		for _, sym := range fr.symbols {
			sig := sym.Signature
			if sig == "" {
				sig = sym.Name
			}
			kind := sym.Kind
			if kind == "" {
				kind = "sym "
			} else {
				kind = padRight(kind, 4)
			}
			fmt.Fprintf(&sb, "  %s  %-40s [line %d]\n", kind, sig, sym.Line)
		}
		sb.WriteString("\n")
	}

	if totalSymbols >= maxResults {
		fmt.Fprintf(&sb, "(results truncated at %d symbols)\n", maxResults)
	}

	return sb.String(), nil
}

func filterSymbols(symbols []lang.Symbol, kindFilter, query string) []lang.Symbol {
	if kindFilter == "" && query == "" {
		return symbols
	}

	var result []lang.Symbol
	for _, sym := range symbols {
		if kindFilter != "" && !strings.EqualFold(sym.Kind, kindFilter) {
			continue
		}
		if !lang.MatchesQuery(sym.Name, query) {
			continue
		}
		result = append(result, sym)
	}
	return result
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}
