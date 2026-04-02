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
)

type InfoTool struct{}

func (t *InfoTool) Name() string { return "info" }

func (t *InfoTool) Description() string {
	return "Project overview in one call: tech stack detection, directory tree, file type counts, and entry points."
}

func (t *InfoTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Project root directory"
			},
			"depth": {
				"type": "integer",
				"description": "Max tree depth (default 4)",
				"default": 4
			}
		},
		"required": ["path"]
	}`)
}

func (t *InfoTool) Annotations() map[string]bool {
	return map[string]bool{
		"readOnlyHint":    true,
		"destructiveHint": false,
		"idempotentHint":  true,
		"openWorldHint":   false,
	}
}

func (t *InfoTool) Execute(ctx context.Context, input json.RawMessage) (interface{}, error) {
	var params struct {
		Path  string `json:"path"`
		Depth int    `json:"depth"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}
	if params.Path == "" {
		return nil, fmt.Errorf("path is required")
	}
	if params.Depth == 0 {
		params.Depth = 4
	}

	root := params.Path
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("cannot access path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory")
	}

	stack := detectTechStack(ctx, root)
	projectName := extractProjectName(stack, root)
	tree, totalFiles, extCounts, ignoredDirs := buildTree(ctx, root, params.Depth, gitignore.New(root))
	entryPoints := detectEntryPoints(ctx, root, gitignore.New(root))

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Project: %s\n", projectName)
	fmt.Fprintf(&sb, "Root: %s\n\n", root)

	sb.WriteString("## Tech Stack\n")
	if len(stack) == 0 {
		sb.WriteString("- (no known config files detected)\n")
	} else {
		for _, item := range stack {
			fmt.Fprintf(&sb, "- %s\n", item)
		}
	}

	fmt.Fprintf(&sb, "\n## Structure (%d files)\n", totalFiles)
	sb.WriteString(tree)

	sb.WriteString("\n## Files by Type\n")
	type extCount struct {
		ext   string
		count int
	}
	sortedExts := make([]extCount, 0, len(extCounts))
	for ext, count := range extCounts {
		sortedExts = append(sortedExts, extCount{ext, count})
	}
	sort.Slice(sortedExts, func(i, j int) bool {
		if sortedExts[i].count != sortedExts[j].count {
			return sortedExts[i].count > sortedExts[j].count
		}
		return sortedExts[i].ext < sortedExts[j].ext
	})
	for _, ec := range sortedExts {
		fmt.Fprintf(&sb, "%-10s %d files\n", ec.ext, ec.count)
	}

	if len(ignoredDirs) > 0 {
		sb.WriteString("\n## Ignored Directories\n")
		for _, d := range ignoredDirs {
			fmt.Fprintf(&sb, "  %s/   (ignored)\n", d)
		}
	}

	if len(entryPoints) > 0 {
		sb.WriteString("\n## Entry Points\n")
		for _, ep := range entryPoints {
			fmt.Fprintf(&sb, "- %s\n", ep)
		}
	}

	return sb.String(), nil
}

func detectTechStack(ctx context.Context, root string) []string {
	type configSpec struct {
		filename string
		extract  func(content string, root string) []string
	}

	specs := []configSpec{
		{
			filename: "go.mod",
			extract: func(content, root string) []string {
				var items []string
				for _, line := range strings.Split(content, "\n") {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "module ") {
						items = append(items, "Language: Go (go.mod)")
						items = append(items, "Module: "+strings.TrimPrefix(line, "module "))
					}
					if strings.HasPrefix(line, "go ") && !strings.Contains(line, "//") {
						ver := strings.TrimPrefix(line, "go ")
						items[0] = "Language: Go " + strings.TrimSpace(ver) + " (go.mod)"
					}
				}
				return items
			},
		},
		{
			filename: "package.json",
			extract: func(content, root string) []string {
				var items []string
				name := extractJSONStringField(content, "name")
				version := extractJSONStringField(content, "version")
				if name != "" {
					items = append(items, "Package: "+name+(func() string {
						if version != "" {
							return " v" + version
						}
						return ""
					})()+" (package.json)")
				}
				if strings.Contains(content, `"react"`) {
					items = append(items, "Framework: React")
				}
				if strings.Contains(content, `"next"`) {
					items = append(items, "Framework: Next.js")
				}
				if strings.Contains(content, `"vue"`) {
					items = append(items, "Framework: Vue")
				}
				if strings.Contains(content, `"typescript"`) || strings.Contains(content, `"ts-node"`) {
					items = append(items, "Language: TypeScript")
				} else {
					items = append(items, "Language: JavaScript")
				}
				return items
			},
		},
		{
			filename: "Cargo.toml",
			extract: func(content, root string) []string {
				var items []string
				items = append(items, "Language: Rust (Cargo.toml)")
				for _, line := range strings.Split(content, "\n") {
					if strings.HasPrefix(strings.TrimSpace(line), "name = ") {
						name := extractTOMLValue(line)
						if name != "" {
							items = append(items, "Package: "+name)
						}
						break
					}
				}
				return items
			},
		},
		{
			filename: "pyproject.toml",
			extract: func(content, root string) []string {
				items := []string{"Language: Python (pyproject.toml)"}
				for _, line := range strings.Split(content, "\n") {
					if strings.HasPrefix(strings.TrimSpace(line), "name = ") {
						if name := extractTOMLValue(line); name != "" {
							items = append(items, "Package: "+name)
							break
						}
					}
				}
				return items
			},
		},
		{
			filename: "requirements.txt",
			extract: func(content, root string) []string {
				return []string{"Language: Python (requirements.txt)"}
			},
		},
		{
			filename: "pom.xml",
			extract: func(content, root string) []string {
				return []string{"Language: Java (pom.xml)", "Build: Maven"}
			},
		},
		{
			filename: "build.gradle",
			extract: func(content, root string) []string {
				return []string{"Build: Gradle"}
			},
		},
		{
			filename: "Gemfile",
			extract: func(content, root string) []string {
				return []string{"Language: Ruby (Gemfile)"}
			},
		},
		{
			filename: "composer.json",
			extract: func(content, root string) []string {
				return []string{"Language: PHP (composer.json)"}
			},
		},
		{
			filename: "Makefile",
			extract: func(content, root string) []string {
				return []string{"Build: Makefile"}
			},
		},
		{
			filename: "CMakeLists.txt",
			extract: func(content, root string) []string {
				return []string{"Build: CMake"}
			},
		},
	}

	seen := make(map[string]bool)
	var result []string

	for _, spec := range specs {
		if ctx.Err() != nil {
			break
		}
		fullPath := filepath.Join(root, spec.filename)
		content, err := encoding.ReadFileAsUTF8(fullPath)
		if err != nil {
			continue
		}
		items := spec.extract(content, root)
		for _, item := range items {
			if !seen[item] {
				seen[item] = true
				result = append(result, item)
			}
		}
	}

	return result
}

func extractProjectName(stack []string, root string) string {
	for _, item := range stack {
		if strings.HasPrefix(item, "Package: ") {
			name := strings.TrimPrefix(item, "Package: ")
			if idx := strings.Index(name, " "); idx > 0 {
				name = name[:idx]
			}
			return name
		}
		if strings.HasPrefix(item, "Module: ") {
			mod := strings.TrimPrefix(item, "Module: ")
			parts := strings.Split(mod, "/")
			return parts[len(parts)-1]
		}
	}
	return filepath.Base(root)
}

func buildTree(ctx context.Context, root string, maxDepth int, matcher *gitignore.Matcher) (string, int, map[string]int, []string) {
	extCounts := make(map[string]int)
	totalFiles := 0
	var ignoredDirs []string
	var sb strings.Builder

	var walkDir func(dir string, depth int, prefix string)
	walkDir = func(dir string, depth int, prefix string) {
		if ctx.Err() != nil {
			return
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}

		sort.Slice(entries, func(i, j int) bool {
			di, dj := entries[i].IsDir(), entries[j].IsDir()
			if di != dj {
				return di
			}
			return entries[i].Name() < entries[j].Name()
		})

		for _, entry := range entries {
			if ctx.Err() != nil {
				return
			}

			relPath, _ := filepath.Rel(root, filepath.Join(dir, entry.Name()))

			if matcher.IsIgnored(relPath, entry.IsDir()) {
				if entry.IsDir() {
					ignoredDirs = append(ignoredDirs, relPath)
				}
				continue
			}

			if entry.IsDir() {
				if depth >= maxDepth {
					fileCount := countFilesInDir(filepath.Join(dir, entry.Name()), matcher, root)
					fmt.Fprintf(&sb, "%s%s/   (%d files)\n", prefix, entry.Name(), fileCount)
					totalFiles += fileCount
					countExtensions(filepath.Join(dir, entry.Name()), matcher, root, extCounts)
					continue
				}
				fmt.Fprintf(&sb, "%s%s/\n", prefix, entry.Name())
				walkDir(filepath.Join(dir, entry.Name()), depth+1, prefix+"  ")
			} else {
				totalFiles++
				ext := filepath.Ext(entry.Name())
				if ext == "" {
					ext = entry.Name()
				}
				extCounts[ext]++
			}
		}
	}

	walkDir(root, 1, "")
	return sb.String(), totalFiles, extCounts, ignoredDirs
}

func countFilesInDir(dir string, matcher *gitignore.Matcher, root string) int {
	count := 0
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		relPath, _ := filepath.Rel(root, path)
		if matcher.IsIgnored(relPath, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	return count
}

func countExtensions(dir string, matcher *gitignore.Matcher, root string, extCounts map[string]int) {
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		relPath, _ := filepath.Rel(root, path)
		if matcher.IsIgnored(relPath, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.IsDir() {
			ext := filepath.Ext(info.Name())
			if ext == "" {
				ext = info.Name()
			}
			extCounts[ext]++
		}
		return nil
	})
}

func detectEntryPoints(ctx context.Context, root string, matcher *gitignore.Matcher) []string {
	var entryPoints []string

	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if ctx.Err() != nil {
			return filepath.SkipAll
		}
		if err != nil || info == nil {
			return nil
		}
		relPath, _ := filepath.Rel(root, path)
		if matcher.IsIgnored(relPath, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() {
			return nil
		}

		content, err := encoding.ReadFileAsUTF8(path)
		if err != nil {
			return nil
		}

		ext := filepath.Ext(path)
		isEntry := false

		switch ext {
		case ".go":
			isEntry = strings.Contains(content, "func main()")
		case ".py":
			isEntry = strings.Contains(content, `if __name__ == "__main__"`) ||
				strings.Contains(content, "if __name__ == '__main__'")
		case ".js", ".ts", ".mjs":
			isEntry = strings.Contains(content, `"main"`) || strings.HasSuffix(path, "index.js") ||
				strings.HasSuffix(path, "index.ts") || strings.HasSuffix(path, "server.js") ||
				strings.HasSuffix(path, "server.ts")
		case ".java":
			isEntry = strings.Contains(content, "public static void main(String")
		case ".rs":
			isEntry = strings.Contains(content, "fn main()")
		}

		if info.Name() == "package.json" {
			if strings.Contains(content, `"main"`) {
				mainVal := extractJSONStringField(content, "main")
				if mainVal != "" {
					entryPoints = append(entryPoints, filepath.Join(relPath, "../"+mainVal))
					return nil
				}
			}
		}

		if isEntry {
			entryPoints = append(entryPoints, relPath)
		}
		return nil
	})

	sort.Strings(entryPoints)
	return entryPoints
}

func extractJSONStringField(content, field string) string {
	needle := `"` + field + `"`
	idx := strings.Index(content, needle)
	if idx < 0 {
		return ""
	}
	rest := content[idx+len(needle):]
	colonIdx := strings.Index(rest, ":")
	if colonIdx < 0 {
		return ""
	}
	rest = strings.TrimSpace(rest[colonIdx+1:])
	if len(rest) == 0 || rest[0] != '"' {
		return ""
	}
	rest = rest[1:]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

func extractTOMLValue(line string) string {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) < 2 {
		return ""
	}
	val := strings.TrimSpace(parts[1])
	val = strings.Trim(val, `"'`)
	return val
}
