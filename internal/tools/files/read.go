package files

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

const maxMmapSize = 1024 * 1024

type ReadRequest struct {
	Path     string `json:"path"`
	Offset   int64  `json:"offset,omitempty"`
	Limit    int64  `json:"limit,omitempty"`
	Encoding string `json:"encoding,omitempty"`
}

type ReadResponse struct {
	Content  string `json:"content"`
	Size     int64  `json:"size"`
	Encoding string `json:"encoding"`
	Lines    int    `json:"lines"`
}

type ReadTool struct{}

func (t *ReadTool) Name() string {
	return "read"
}

func (t *ReadTool) Description() string {
	return "Efficiently read file contents with streaming and encoding detection"
}

func (t *ReadTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Path to the file to read (absolute path required)"
			},
			"offset": {
				"type": "integer",
				"description": "Starting byte offset (optional, default: 0)",
				"minimum": 0
			},
			"limit": {
				"type": "integer",
				"description": "Maximum bytes to read (optional, 0 = no limit)",
				"minimum": 0
			},
			"encoding": {
				"type": "string",
				"description": "Expected encoding (optional: utf-8, utf-16, iso-8859-1, auto)",
				"enum": ["utf-8", "utf-16", "iso-8859-1", "auto"]
			}
		},
		"required": ["path"]
	}`)
}

func (t *ReadTool) Execute(input json.RawMessage) (interface{}, error) {
	var req ReadRequest
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if req.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	file, err := os.Open(req.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	fileSize := stat.Size()

	if req.Offset > 0 {
		if _, err := file.Seek(req.Offset, io.SeekStart); err != nil {
			return nil, fmt.Errorf("failed to seek: %w", err)
		}
	}

	var content []byte
	var readSize int64

	if req.Limit > 0 {
		readSize = req.Limit
	} else if req.Offset > 0 {
		readSize = fileSize - req.Offset
	} else {
		readSize = fileSize
	}

	if readSize > 0 {
		if readSize > 50*1024*1024 {
			return nil, fmt.Errorf("file too large: %d bytes (max 50MB)", readSize)
		}

		content = make([]byte, readSize)
		if _, err := io.ReadFull(file, content); err != nil && err != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}
	}

	encoding := req.Encoding
	if encoding == "" || encoding == "auto" {
		encoding = detectEncoding(content)
	}

	contentStr := decodeContent(content, encoding)
	lineCount := strings.Count(contentStr, "\n") + 1
	if contentStr == "" {
		lineCount = 0
	}

	return ReadResponse{
		Content:  contentStr,
		Size:     fileSize,
		Encoding: encoding,
		Lines:    lineCount,
	}, nil
}

func detectEncoding(data []byte) string {
	if len(data) == 0 {
		return "utf-8"
	}

	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return "utf-8"
	}

	if len(data) >= 2 {
		if data[0] == 0xFF && data[1] == 0xFE {
			return "utf-16"
		}
		if data[0] == 0xFE && data[1] == 0xFF {
			return "utf-16"
		}
	}

	validUTF8 := 0
	for _, b := range data {
		if b < 0x80 || utf8.RuneStart(b) {
			validUTF8++
		}
	}

	if float64(validUTF8) > float64(len(data))*0.95 {
		return "utf-8"
	}

	isLatin1 := true
	for _, b := range data {
		if b > 0xFF {
			isLatin1 = false
			break
		}
	}
	if isLatin1 {
		return "iso-8859-1"
	}

	return "utf-8"
}

func decodeContent(data []byte, encoding string) string {
	switch encoding {
	case "utf-16":
		if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE {
			return decodeUTF16LE(data[2:])
		}
		if len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF {
			return decodeUTF16BE(data[2:])
		}
		return decodeUTF16LE(data)
	case "iso-8859-1":
		return string(data)
	default:
		return string(data)
	}
}

func decodeUTF16LE(data []byte) string {
	result := make([]rune, 0, len(data)/2)
	for i := 0; i < len(data)-1; i += 2 {
		r := rune(data[i]) | (rune(data[i+1]) << 8)
		result = append(result, r)
	}
	return string(result)
}

func decodeUTF16BE(data []byte) string {
	result := make([]rune, 0, len(data)/2)
	for i := 0; i < len(data)-1; i += 2 {
		r := (rune(data[i]) << 8) | rune(data[i+1])
		result = append(result, r)
	}
	return string(result)
}
