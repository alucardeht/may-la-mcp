package router

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"

	"github.com/alucardeht/may-la-mcp/internal/index"
)

type FileHasher struct{}

func (h *FileHasher) ComputeHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func IsFileFresh(store *index.IndexStore, path string) (bool, error) {
	indexed, err := store.GetFile(path)
	if err != nil {
		return false, err
	}
	if indexed == nil {
		return false, nil
	}

	hasher := &FileHasher{}
	currentHash, err := hasher.ComputeHash(path)
	if err != nil {
		return false, err
	}

	return indexed.ContentHash == currentHash, nil
}
