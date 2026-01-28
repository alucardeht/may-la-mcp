package search

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type ripgrepResult struct {
	Type   string `json:"type"`
	Data   ripgrepData `json:"data"`
}

type ripgrepData struct {
	Path   ripgrepPath `json:"path"`
	Lines  ripgrepLines `json:"lines"`
	LineNum uint64 `json:"line_number"`
	Column  uint64 `json:"column"`
}

type ripgrepPath struct {
	Text string `json:"text"`
}

type ripgrepLines struct {
	Text string `json:"text"`
}

var (
	rgOnce      sync.Once
	rgAvailable bool
)

func isRipgrepAvailable() bool {
	rgOnce.Do(func() {
		_, err := exec.LookPath("rg")
		rgAvailable = (err == nil)
	})
	return rgAvailable
}

func executeRipgrep(req SearchRequest) (*SearchResponse, error) {
	if !isRipgrepAvailable() {
		return nil, fmt.Errorf("ripgrep not available")
	}

	args := []string{
		"--json",
		"--color=never",
	}

	if !req.CaseSensitive {
		args = append(args, "-i")
	}

	if req.Regex {
		args = append(args, "-e", req.Pattern)
	} else {
		args = append(args, "-F", req.Pattern)
	}

	if req.MaxResults > 0 {
		args = append(args, fmt.Sprintf("--max-count=%d", req.MaxResults))
	}

	args = append(args, req.Path)

	cmd := exec.Command("rg", args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil && !strings.Contains(err.Error(), "exit status 1") {
		return nil, fmt.Errorf("ripgrep error: %w", err)
	}

	matches := []Match{}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		var result ripgrepResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			continue
		}

		if result.Type == "match" {
			match := Match{
				File:    result.Data.Path.Text,
				Line:    int(result.Data.LineNum),
				Column:  int(result.Data.Column),
				Content: result.Data.Lines.Text,
			}

			if req.ContextLines > 0 {
				match.Context = getContextFromRipgrep(req.Path, match.File, match.Line, req.ContextLines)
			}

			matches = append(matches, match)
		}
	}

	return &SearchResponse{
		Matches: matches,
		Count:   len(matches),
		Path:    req.Path,
	}, nil
}

func getContextFromRipgrep(searchPath string, filePath string, lineNum int, contextLines int) []string {
	fileInfo, err := os.Stat(filePath)
	if err == nil && fileInfo.Size() > MaxGrepFileSize {
		return nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	var fileLines []string
	for scanner.Scan() {
		fileLines = append(fileLines, scanner.Text())
	}

	context := []string{}
	start := lineNum - contextLines - 1
	if start < 0 {
		start = 0
	}

	end := lineNum + contextLines
	if end > len(fileLines) {
		end = len(fileLines)
	}

	for i := start; i < end; i++ {
		if i >= 0 && i < len(fileLines) {
			context = append(context, fileLines[i])
		}
	}

	return context
}
