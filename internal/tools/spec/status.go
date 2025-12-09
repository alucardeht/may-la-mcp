package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ArtifactStatus struct {
	Name         string
	Exists       bool
	Size         int64
	Completeness float32
}

type StatusResult struct {
	ProjectInitialized bool
	Artifacts          []*ArtifactStatus
	CompletionScore    float32
	Summary            string
	Issues             []string
}

func Status(path string) (*StatusResult, error) {
	result := &StatusResult{
		Artifacts: make([]*ArtifactStatus, 0),
		Issues:    make([]string, 0),
	}

	maylaDir := filepath.Join(path, ".mayla")

	if !dirExists(maylaDir) {
		result.ProjectInitialized = false
		result.Summary = "Spec-driven project not initialized. Run spec_init first."
		return result, nil
	}

	result.ProjectInitialized = true

	artifactNames := []string{"constitution", "spec", "plan", "tasks"}
	totalCompleteness := float32(0)

	for _, name := range artifactNames {
		filePath := filepath.Join(maylaDir, name+".md")
		status := &ArtifactStatus{Name: name}

		if fileExists(filePath) {
			status.Exists = true
			info, _ := os.Stat(filePath)
			status.Size = info.Size()
			status.Completeness = calculateCompleteness(filePath, name)
		} else {
			status.Exists = false
			status.Completeness = 0
		}

		result.Artifacts = append(result.Artifacts, status)
		totalCompleteness += status.Completeness
	}

	result.CompletionScore = totalCompleteness / float32(len(artifactNames))

	constitution, err := ParseConstitution(filepath.Join(maylaDir, "constitution.md"))
	if err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("Constitution parsing error: %v", err))
	} else if len(constitution.Principles) == 0 {
		result.Issues = append(result.Issues, "Constitution has no principles defined")
	}

	specPath := filepath.Join(maylaDir, "spec.md")
	if fileExists(specPath) {
		if !validateSpecStructure(specPath) {
			result.Issues = append(result.Issues, "Spec missing required sections")
		}
	} else {
		result.Issues = append(result.Issues, "Spec artifact not found")
	}

	planPath := filepath.Join(maylaDir, "plan.md")
	if fileExists(planPath) {
		if !validatePlanStructure(planPath) {
			result.Issues = append(result.Issues, "Plan missing required sections")
		}
	} else {
		result.Issues = append(result.Issues, "Plan artifact not found")
	}

	tasksPath := filepath.Join(maylaDir, "tasks.md")
	if !fileExists(tasksPath) {
		result.Issues = append(result.Issues, "Tasks artifact not found")
	}

	result.Summary = formatStatusSummary(result)

	return result, nil
}

func calculateCompleteness(filePath string, artifactType string) float32 {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return 0
	}

	text := string(content)
	requiredSections := getRequiredSections(artifactType)
	foundSections := 0

	for _, section := range requiredSections {
		if strings.Contains(text, section) {
			foundSections++
		}
	}

	if len(requiredSections) == 0 {
		return 0.5
	}

	return float32(foundSections) / float32(len(requiredSections))
}

func getRequiredSections(artifactType string) []string {
	switch artifactType {
	case "constitution":
		return []string{"## Principles", "## Constraints", "## Governance"}
	case "spec":
		return []string{"## User Stories", "## Requirements", "## Acceptance Criteria"}
	case "plan":
		return []string{"## Technical Context", "## Architecture", "## Phases"}
	case "tasks":
		return []string{"## Phase", "TASK-", "Priority:", "Estimate:"}
	default:
		return []string{}
	}
}

func validateSpecStructure(filePath string) bool {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	text := string(content)
	requiredSections := []string{"## User Stories", "## Requirements"}

	for _, section := range requiredSections {
		if !strings.Contains(text, section) {
			return false
		}
	}

	return true
}

func validatePlanStructure(filePath string) bool {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	text := string(content)
	requiredSections := []string{"## Technical Context", "## Architecture", "## Phases"}

	for _, section := range requiredSections {
		if !strings.Contains(text, section) {
			return false
		}
	}

	return true
}

func formatStatusSummary(result *StatusResult) string {
	if !result.ProjectInitialized {
		return "Spec-driven project not initialized"
	}

	existingCount := 0
	for _, artifact := range result.Artifacts {
		if artifact.Exists {
			existingCount++
		}
	}

	score := int(result.CompletionScore * 100)
	issueInfo := ""
	if len(result.Issues) > 0 {
		issueInfo = fmt.Sprintf(" Found %d issues.", len(result.Issues))
	}

	return fmt.Sprintf("Project initialized. %d/%d artifacts present. Completeness: %d%%%s",
		existingCount, len(result.Artifacts), score, issueInfo)
}
