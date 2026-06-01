package gitignore

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

var defaultIgnored = []string{
	".git",
	"node_modules",
	"vendor",
	"__pycache__",
	"dist",
	"build",
	".DS_Store",
	".next",
	".nuxt",
	"target",
	"coverage",
	".cache",
}

type Matcher struct {
	patterns []pattern
}

type pattern struct {
	raw     string
	negate  bool
	dirOnly bool
}

func New(root string) *Matcher {
	m := &Matcher{}

	for _, p := range defaultIgnored {
		m.patterns = append(m.patterns, pattern{raw: p})
	}

	gitignorePath := filepath.Join(root, ".gitignore")
	if f, err := os.Open(gitignorePath); err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			p := pattern{raw: line}
			if strings.HasPrefix(line, "!") {
				p.negate = true
				p.raw = line[1:]
			}
			if strings.HasSuffix(p.raw, "/") {
				p.dirOnly = true
				p.raw = strings.TrimSuffix(p.raw, "/")
			}
			m.patterns = append(m.patterns, p)
		}
	}

	return m
}

func (m *Matcher) IsIgnored(relPath string, isDir bool) bool {
	relPath = filepath.ToSlash(relPath)
	ignored := false
	name := filepath.Base(relPath)

	for _, p := range m.patterns {
		if p.dirOnly && !isDir {
			continue
		}

		matched := false
		if !strings.Contains(p.raw, "/") {
			if match, _ := filepath.Match(p.raw, name); match {
				matched = true
			}
		}
		if !matched {
			if match, _ := filepath.Match(p.raw, relPath); match {
				matched = true
			}
		}
		if !matched && strings.HasPrefix(p.raw, "**/") {
			trimmed := p.raw[3:]
			if match, _ := filepath.Match(trimmed, name); match {
				matched = true
			}
		}

		if matched {
			if p.negate {
				ignored = false
			} else {
				ignored = true
			}
		}
	}

	return ignored
}

func IsSourceFile(path string) bool {
	ext := filepath.Ext(path)
	sourceExts := map[string]bool{
		".go": true, ".js": true, ".ts": true, ".tsx": true, ".jsx": true, ".mjs": true,
		".py": true, ".rb": true, ".java": true, ".kt": true, ".scala": true,
		".rs": true, ".cpp": true, ".c": true, ".h": true, ".hpp": true,
		".cs": true, ".swift": true, ".php": true,
		".vue": true, ".svelte": true,
		".yaml": true, ".yml": true, ".toml": true, ".json": true,
		".md": true, ".txt": true,
		".sh": true, ".bash": true, ".zsh": true,
		".sql": true, ".graphql": true, ".gql": true,
		".proto": true, ".thrift": true,
		".css": true, ".scss": true, ".less": true,
		".html": true, ".xml": true,
		".env": true, ".conf": true, ".ini": true, ".cfg": true,
	}
	return sourceExts[ext]
}
