package encoding

import (
	"bytes"
	"os"
	"unicode/utf8"
)

var (
	utf8BOM    = []byte{0xEF, 0xBB, 0xBF}
	utf16LEBOM = []byte{0xFF, 0xFE}
	utf16BEBOM = []byte{0xFE, 0xFF}
)

func ReadFileAsUTF8(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	if bytes.HasPrefix(data, utf8BOM) {
		data = data[3:]
	}

	if bytes.HasPrefix(data, utf16LEBOM) {
		data = data[2:]
		return decodeUTF16LE(data), nil
	}

	if bytes.HasPrefix(data, utf16BEBOM) {
		data = data[2:]
		return decodeUTF16BE(data), nil
	}

	if utf8.Valid(data) {
		return string(data), nil
	}

	runes := make([]rune, len(data))
	for i, b := range data {
		runes[i] = rune(b)
	}
	return string(runes), nil
}

func decodeUTF16LE(data []byte) string {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	runes := make([]rune, 0, len(data)/2)
	for i := 0; i < len(data)-1; i += 2 {
		r := rune(data[i]) | rune(data[i+1])<<8
		runes = append(runes, r)
	}
	return string(runes)
}

func decodeUTF16BE(data []byte) string {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	runes := make([]rune, 0, len(data)/2)
	for i := 0; i < len(data)-1; i += 2 {
		r := rune(data[i])<<8 | rune(data[i+1])
		runes = append(runes, r)
	}
	return string(runes)
}
