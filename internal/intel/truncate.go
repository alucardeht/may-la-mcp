package intel

import (
	"fmt"
	"strings"
)

type TruncateMode string

const (
	TruncateModeHead   TruncateMode = "HEAD"
	TruncateModeTail   TruncateMode = "TAIL"
	TruncateModeMid    TruncateMode = "MIDDLE"
	TruncateModeSmart  TruncateMode = "SMART"
)

func Truncate(content string, maxLen int, mode TruncateMode) string {
	if len(content) <= maxLen {
		return content
	}

	switch mode {
	case TruncateModeHead:
		return truncateHead(content, maxLen)
	case TruncateModeTail:
		return truncateTail(content, maxLen)
	case TruncateModeMid:
		return truncateMid(content, maxLen)
	case TruncateModeSmart:
		return truncateSmart(content, maxLen)
	default:
		return truncateSmart(content, maxLen)
	}
}

func truncateHead(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}

	remaining := len(content) - maxLen
	headLen := (maxLen * 3) / 4
	tailLen := maxLen / 4

	if headLen > len(content) {
		headLen = maxLen
		tailLen = 0
	}

	result := content[:headLen]

	if tailLen > 0 && tailLen < len(content) {
		result = result + "\n\n[... " + fmt.Sprintf("%d", remaining) + " chars omitted ...]\n\n" + content[len(content)-tailLen:]
	}

	if len(result) > maxLen {
		result = result[:maxLen] + "..."
	}

	return result
}

func truncateTail(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}

	remaining := len(content) - maxLen
	headLen := maxLen / 2
	tailLen := (maxLen * 3) / 4

	if headLen > len(content) {
		headLen = maxLen
	}

	result := content[:headLen] + "\n\n[... " + fmt.Sprintf("%d", remaining) + " chars omitted ...]\n\n"

	if tailLen < len(content) {
		result = result + content[len(content)-tailLen:]
	}

	if len(result) > maxLen {
		result = result[:maxLen] + "..."
	}

	return result
}

func truncateMid(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}

	remaining := len(content) - maxLen
	headLen := maxLen / 2
	tailLen := maxLen / 2

	if headLen+tailLen > len(content) {
		headLen = maxLen
		tailLen = 0
	}

	result := content[:headLen] + "\n[... " + fmt.Sprintf("%d", remaining) + " chars ...]\n" + content[len(content)-tailLen:]

	if len(result) > maxLen {
		result = result[:maxLen] + "..."
	}

	return result
}

func truncateSmart(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}

	lines := strings.Split(content, "\n")
	result := strings.Builder{}
	omittedLines := 0

	for _, line := range lines {
		if result.Len()+len(line)+1 > maxLen {
			omittedLines++
			continue
		}

		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString(line)
	}

	finalResult := result.String()

	if omittedLines > 0 {
		indicator := fmt.Sprintf("\n\n... (%d more lines)", omittedLines)
		if len(finalResult)+len(indicator) > maxLen {
			finalResult = finalResult[:maxLen-len(indicator)] + indicator
		} else {
			finalResult = finalResult + indicator
		}
	}

	return finalResult
}

func smartBoundaryTruncate(content string, maxLen int) string {
	lines := strings.Split(content, "\n")

	var selectedLines []string
	currentLen := 0

	for _, line := range lines {
		lineLen := len(line) + 1

		if currentLen+lineLen > maxLen {
			break
		}

		selectedLines = append(selectedLines, line)
		currentLen += lineLen
	}

	if len(selectedLines) < len(lines) {
		omitted := len(lines) - len(selectedLines)
		return strings.Join(selectedLines, "\n") + fmt.Sprintf("\n\n... (%d more lines)", omitted)
	}

	return strings.Join(selectedLines, "\n")
}

func findBoundary(content string, position int) int {
	if position >= len(content) {
		return len(content)
	}

	for i := position; i >= 0; i-- {
		if content[i] == '\n' || content[i] == '.' || content[i] == ';' || content[i] == '}' {
			return i + 1
		}
	}

	return position
}
