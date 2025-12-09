package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type GenerateResult struct {
	Success   bool
	Content   string
	FilePath  string
	Timestamp time.Time
	Message   string
}

func Generate(path string, artifact string, content map[string]interface{}) (*GenerateResult, error) {
	result := &GenerateResult{
		Timestamp: time.Now(),
	}

	var generatedContent string
	projectName := ""
	input := ""

	if contentStr, ok := content["content"].(string); ok {
		input = contentStr
	}
	if projName, ok := content["projectName"].(string); ok {
		projectName = projName
	}

	switch strings.ToLower(artifact) {
	case "constitution":
		generatedContent = generateConstitution(input, projectName)
	case "spec":
		generatedContent = generateSpec(input, projectName)
	case "plan":
		generatedContent = generatePlan(input, projectName)
	case "tasks":
		generatedContent = generateTasks(input, projectName)
	default:
		return nil, fmt.Errorf("unknown artifact type: %s", artifact)
	}

	maylaDir := filepath.Join(path, ".mayla")
	if err := os.MkdirAll(maylaDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .mayla directory: %w", err)
	}

	fileName := fmt.Sprintf("%s.md", strings.ToLower(artifact))
	filePath := filepath.Join(maylaDir, fileName)

	if err := os.WriteFile(filePath, []byte(generatedContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write artifact file: %w", err)
	}

	result.Success = true
	result.Content = generatedContent
	result.FilePath = filePath
	result.Message = fmt.Sprintf("Generated %s artifact at %s", artifact, filePath)

	return result, nil
}

func generateConstitution(input string, projectName string) string {
	return strings.ReplaceAll(strings.ReplaceAll(constitutionTemplate,
		"[Project Name]", projectName),
		"[Description]", input)
}

func generateSpec(input string, projectName string) string {
	return strings.ReplaceAll(strings.ReplaceAll(specTemplate,
		"[Feature Name]", projectName),
		"[Description]", input)
}

func generatePlan(input string, projectName string) string {
	return strings.ReplaceAll(strings.ReplaceAll(planTemplate,
		"[Feature Name]", projectName),
		"[Technical Context]", input)
}

func generateTasks(input string, projectName string) string {
	return strings.ReplaceAll(strings.ReplaceAll(tasksTemplate,
		"[Feature Name]", projectName),
		"[Initial Tasks]", input)
}
