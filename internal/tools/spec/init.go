package spec

import (
	"fmt"
	"os"
	"path/filepath"
)

type InitResult struct {
	Success bool
	Message string
	Created []string
	Errors  []string
}

func Init(path string, force bool) (*InitResult, error) {
	result := &InitResult{
		Created: make([]string, 0),
		Errors:  make([]string, 0),
	}

	maylaDir := filepath.Join(path, ".mayla")

	if err := os.MkdirAll(maylaDir, 0755); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to create .mayla directory: %v", err))
		return result, err
	}

	memoriesDir := filepath.Join(maylaDir, "memories")
	if err := os.MkdirAll(memoriesDir, 0755); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to create memories directory: %v", err))
		return result, err
	}

	cacheDir := filepath.Join(maylaDir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to create cache directory: %v", err))
		return result, err
	}

	constitutionPath := filepath.Join(maylaDir, "constitution.md")
	constitutionExists := fileExists(constitutionPath)

	if constitutionExists && !force {
		result.Errors = append(result.Errors, "constitution.md already exists (use force=true to overwrite)")
	} else {
		if err := os.WriteFile(constitutionPath, []byte(constitutionTemplate), 0644); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to create constitution.md: %v", err))
		} else {
			result.Created = append(result.Created, constitutionPath)
		}
	}

	if len(result.Errors) == 0 {
		result.Success = true
		result.Message = "Spec-driven project initialized successfully"
		result.Created = append(result.Created, maylaDir, memoriesDir, cacheDir)
	} else {
		result.Message = "Initialization completed with errors"
	}

	return result, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
