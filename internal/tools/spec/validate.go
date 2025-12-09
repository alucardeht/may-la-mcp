package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ValidationResult struct {
	Valid      bool
	Violations []Violation
	Warnings   []string
	Summary    string
}

func Validate(path string) (*ValidationResult, error) {
	result := &ValidationResult{
		Violations: make([]Violation, 0),
		Warnings:   make([]string, 0),
	}

	maylaDir := filepath.Join(path, ".mayla")

	if !dirExists(maylaDir) {
		result.Violations = append(result.Violations, Violation{
			Type:        "MISSING_MAYLA_DIR",
			Description: ".mayla directory not found",
			Severity:    "ERROR",
		})
		result.Summary = "Spec-driven project not initialized"
		return result, nil
	}

	constitution, err := ParseConstitution(filepath.Join(maylaDir, "constitution.md"))
	if err != nil {
		result.Violations = append(result.Violations, Violation{
			Type:        "INVALID_CONSTITUTION",
			Description: fmt.Sprintf("Failed to parse constitution: %v", err),
			Severity:    "ERROR",
		})
	} else {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Constitution loaded with %d principles and %d constraints",
			len(constitution.Principles), len(constitution.Constraints)))
	}

	specPath := filepath.Join(maylaDir, "spec.md")
	if fileExists(specPath) {
		content, _ := os.ReadFile(specPath)
		if constitution != nil {
			violations := validateSpecAgainstConstitution(constitution, string(content))
			result.Violations = append(result.Violations, violations...)
		}
	} else {
		result.Warnings = append(result.Warnings, "spec.md not found")
	}

	planPath := filepath.Join(maylaDir, "plan.md")
	if fileExists(planPath) {
		result.Warnings = append(result.Warnings, "plan.md found")
		if fileExists(specPath) {
			violations := validatePlanAgainstSpec(filepath.Join(maylaDir, "spec.md"), planPath)
			result.Violations = append(result.Violations, violations...)
		}
	} else {
		result.Warnings = append(result.Warnings, "plan.md not found")
	}

	tasksPath := filepath.Join(maylaDir, "tasks.md")
	if fileExists(tasksPath) {
		result.Warnings = append(result.Warnings, "tasks.md found")
		if fileExists(planPath) {
			violations := validateTasksAgainstPlan(planPath, tasksPath)
			result.Violations = append(result.Violations, violations...)
		}
	} else {
		result.Warnings = append(result.Warnings, "tasks.md not found")
	}

	result.Valid = len(result.Violations) == 0
	result.Summary = formatValidationSummary(result)

	return result, nil
}

func validateSpecAgainstConstitution(constitution *Constitution, spec string) []Violation {
	violations := make([]Violation, 0)

	if len(constitution.Principles) == 0 {
		violations = append(violations, Violation{
			Type:        "NO_PRINCIPLES_DEFINED",
			Description: "Constitution has no principles defined",
			Severity:    "WARNING",
		})
	}

	for _, principle := range constitution.Principles {
		if !strings.Contains(spec, principle.ID) {
			violations = append(violations, Violation{
				Type:        "PRINCIPLE_NOT_REFERENCED",
				Description: fmt.Sprintf("Principle %s not referenced in spec", principle.ID),
				Severity:    "INFO",
			})
		}
	}

	return violations
}

func validatePlanAgainstSpec(specPath, planPath string) []Violation {
	violations := make([]Violation, 0)

	spec, errSpec := os.ReadFile(specPath)
	plan, errPlan := os.ReadFile(planPath)

	if errSpec != nil || errPlan != nil {
		violations = append(violations, Violation{
			Type:        "FILE_READ_ERROR",
			Description: "Failed to read spec or plan file",
			Severity:    "ERROR",
		})
		return violations
	}

	specContent := string(spec)
	planContent := string(plan)

	userStoryPattern := "## User Stories"
	if !strings.Contains(specContent, userStoryPattern) {
		violations = append(violations, Violation{
			Type:        "MISSING_USER_STORIES",
			Description: "Spec missing User Stories section",
			Severity:    "WARNING",
		})
	}

	if !strings.Contains(planContent, "## Technical Context") {
		violations = append(violations, Violation{
			Type:        "MISSING_TECH_CONTEXT",
			Description: "Plan missing Technical Context section",
			Severity:    "WARNING",
		})
	}

	return violations
}

func validateTasksAgainstPlan(planPath, tasksPath string) []Violation {
	violations := make([]Violation, 0)

	plan, errPlan := os.ReadFile(planPath)
	tasks, errTasks := os.ReadFile(tasksPath)

	if errPlan != nil || errTasks != nil {
		violations = append(violations, Violation{
			Type:        "FILE_READ_ERROR",
			Description: "Failed to read plan or tasks file",
			Severity:    "ERROR",
		})
		return violations
	}

	planContent := string(plan)
	tasksContent := string(tasks)

	if !strings.Contains(planContent, "## Phases") {
		violations = append(violations, Violation{
			Type:        "MISSING_PHASES",
			Description: "Plan missing Phases section",
			Severity:    "WARNING",
		})
	}

	if !strings.Contains(tasksContent, "## Phase") {
		violations = append(violations, Violation{
			Type:        "MISSING_TASK_PHASES",
			Description: "Tasks missing Phase sections",
			Severity:    "WARNING",
		})
	}

	return violations
}

func formatValidationSummary(result *ValidationResult) string {
	errorCount := 0
	warningCount := 0
	infoCount := 0

	for _, v := range result.Violations {
		switch v.Severity {
		case "ERROR":
			errorCount++
		case "WARNING":
			warningCount++
		case "INFO":
			infoCount++
		}
	}

	return fmt.Sprintf("Validation: %d errors, %d warnings, %d infos. Valid: %v",
		errorCount, warningCount, infoCount, result.Valid)
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
