package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/alucardeht/may-la-mcp/internal/gitignore"
	"github.com/bmatcuk/doublestar/v4"
)

type FindTool struct{}

func (t *FindTool) Name() string { return "find" }

func (t *FindTool) Description() string {
	return "Discover files by glob pattern with metadata, grouped by directory."
}

func (t *FindTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"pattern": {
				"type": "string",
				"description": "Glob pattern (e.g. '**/*.go', 'src/**/*.ts', 'Dockerfile*')"
			},
			"path": {
				"type": "string",
				"description": "Root path to search from (defaults to cwd)"
			},
			"type": {
				"type": "string",
				"description": "Entry type: 'file', 'dir', or 'all' (default 'file')",
				"enum": ["file", "dir", "all"],
				"default": "file"
			},
			"max_results": {
				"type": "integer",
				"description": "Maximum number of results (default 200)",
				"default": 200
			}
		},
		"required": ["pattern"]
	}`)
}

func (t *FindTool) Annotations() map[string]bool {
	return map[string]bool{
		"readOnlyHint":    true,
		"destructiveHint": false,
		"idempotentHint":  true,
		"openWorldHint":   false,
	}
}

type foundEntry struct {
	absPath string
	relPath string
	size    int64
	isDir   bool
}

func (t *FindTool) Execute(ctx context.Context, input json.RawMessage) (interface{}, error) {
	var params struct {
		Pattern    string `json:"pattern"`
		Path       string `json:"path"`
		Type       string `json:"type"`
		MaxResults int    `json:"max_results"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}
	if params.Pattern == "" {
		return nil, fmt.Errorf("pattern is required")
	}
	if params.Path == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("cannot determine cwd: %w", err)
		}
		params.Path = cwd
	}
	if params.Type == "" {
		params.Type = "file"
	}
	if params.MaxResults <= 0 {
		params.MaxResults = 200
	}

	root := params.Path
	rootInfo, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("cannot access path: %w", err)
	}
	if !rootInfo.IsDir() {
		return nil, fmt.Errorf("path is not a directory")
	}

	matcher := gitignore.New(root)
	var entries []foundEntry

	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if ctx.Err() != nil {
			return filepath.SkipAll
		}
		if err != nil || info == nil {
			return nil
		}
		if path == root {
			return nil
		}

		relPath, _ := filepath.Rel(root, path)
		relPath = filepath.ToSlash(relPath)

		if matcher.IsIgnored(relPath, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		matchesType := false
		switch params.Type {
		case "file":
			matchesType = !info.IsDir()
		case "dir":
			matchesType = info.IsDir()
		case "all":
			matchesType = true
		}

		if !matchesType {
			return nil
		}

		matched := matchesGlob(params.Pattern, relPath, info.Name())
		if !matched {
			return nil
		}

		if len(entries) >= params.MaxResults {
			return filepath.SkipAll
		}

		entries = append(entries, foundEntry{
			absPath: path,
			relPath: relPath,
			size:    info.Size(),
			isDir:   info.IsDir(),
		})

		return nil
	})

	return formatFindOutput(params.Pattern, root, entries, params.MaxResults), nil
}

func matchesGlob(pattern, relPath, name string) bool {
	matched, _ := doublestar.PathMatch(pattern, relPath)
	if matched {
		return true
	}
	matched, _ = doublestar.PathMatch(pattern, name)
	if matched {
		return true
	}
	if strings.Contains(pattern, "/") {
		return false
	}
	matched, _ = filepath.Match(pattern, name)
	return matched
}

type dirGroup struct {
	dir     string
	entries []foundEntry
}

func formatFindOutput(pattern, root string, entries []foundEntry, maxResults int) string {
	if len(entries) == 0 {
		return fmt.Sprintf("# Find: %s in %s\n\nNo matches found.\n", pattern, root)
	}

	groups := make(map[string][]foundEntry)
	for _, e := range entries {
		dir := filepath.Dir(e.relPath)
		if dir == "." {
			dir = ""
		}
		groups[dir] = append(groups[dir], e)
	}

	dirNames := make([]string, 0, len(groups))
	for d := range groups {
		dirNames = append(dirNames, d)
	}
	sort.Strings(dirNames)

	for d := range groups {
		sort.Slice(groups[d], func(i, j int) bool {
			return filepath.Base(groups[d][i].relPath) < filepath.Base(groups[d][j].relPath)
		})
	}

	var totalSize int64
	for _, e := range entries {
		totalSize += e.size
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Find: %s in %s (%d matches)\n\n", pattern, root, len(entries))

	for _, dir := range dirNames {
		grp := groups[dir]
		var groupSize int64
		for _, e := range grp {
			groupSize += e.size
		}

		prefix := dir
		if prefix == "" {
			prefix = "."
		}
		fmt.Fprintf(&sb, "## %s/ (%d files, %s)\n", prefix, len(grp), formatSize(groupSize))

		for _, e := range grp {
			name := filepath.Base(e.relPath)
			if e.isDir {
				name += "/"
			}
			fmt.Fprintf(&sb, "  %-30s %s\n", name, formatSize(e.size))
		}
		sb.WriteString("\n")
	}

	fmt.Fprintf(&sb, "Total: %d files, %s\n", len(entries), formatSize(totalSize))
	if len(entries) >= maxResults {
		fmt.Fprintf(&sb, "(results truncated at %d)\n", maxResults)
	}

	return sb.String()
}

func formatSize(bytes int64) string {
	const kb = 1024
	const mb = 1024 * kb

	switch {
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
