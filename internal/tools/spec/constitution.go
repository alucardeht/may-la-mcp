package spec

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

type Principle struct {
	ID          string
	Name        string
	Description string
}

type Constraint struct {
	ID          string
	Name        string
	Description string
}

type Governance struct {
	Rules []string
}

type Constitution struct {
	Principles  []*Principle
	Constraints []*Constraint
	Governance  *Governance
	RawContent  string
}

type Violation struct {
	Type        string
	Description string
	Severity    string
}

func ParseConstitution(path string) (*Constitution, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read constitution file: %w", err)
	}

	text := string(content)
	constitution := &Constitution{
		Principles:  make([]*Principle, 0),
		Constraints: make([]*Constraint, 0),
		Governance:  &Governance{Rules: make([]string, 0)},
		RawContent:  text,
	}

	extractSection := func(sectionName string) string {
		startPattern := fmt.Sprintf(`## %s\n`, sectionName)
		endPattern := `## `

		startIdx := strings.Index(text, startPattern)
		if startIdx == -1 {
			return ""
		}

		startIdx += len(startPattern)
		endIdx := strings.Index(text[startIdx:], endPattern)

		if endIdx == -1 {
			return text[startIdx:]
		}

		return text[startIdx : startIdx+endIdx]
	}

	principlesText := extractSection("Principles")
	constitution.Principles = parsePrinciples(principlesText)

	constraintsText := extractSection("Constraints")
	constitution.Constraints = parseConstraints(constraintsText)

	governanceText := extractSection("Governance")
	constitution.Governance.Rules = parseGovernanceRules(governanceText)

	return constitution, nil
}

func parsePrinciples(text string) []*Principle {
	principles := make([]*Principle, 0)

	if text == "" {
		return principles
	}

	lines := strings.Split(text, "\n")
	var currentPrinciple *Principle

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "### P") {
			if currentPrinciple != nil {
				principles = append(principles, currentPrinciple)
			}

			idPattern := regexp.MustCompile(`### (P\d+):?\s*(.*)`)
			matches := idPattern.FindStringSubmatch(line)

			if len(matches) >= 2 {
				currentPrinciple = &Principle{
					ID:   matches[1],
					Name: matches[2],
				}
			}
		} else if currentPrinciple != nil && line != "" && !strings.HasPrefix(line, "###") {
			currentPrinciple.Description += line + " "
		}
	}

	if currentPrinciple != nil {
		principles = append(principles, currentPrinciple)
	}

	for _, p := range principles {
		p.Description = strings.TrimSpace(p.Description)
	}

	return principles
}

func parseConstraints(text string) []*Constraint {
	constraints := make([]*Constraint, 0)

	if text == "" {
		return constraints
	}

	lines := strings.Split(text, "\n")
	var currentConstraint *Constraint

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "### TC") || strings.HasPrefix(line, "### C") {
			if currentConstraint != nil {
				constraints = append(constraints, currentConstraint)
			}

			idPattern := regexp.MustCompile(`### ((?:TC|C)\d+):?\s*(.*)`)
			matches := idPattern.FindStringSubmatch(line)

			if len(matches) >= 2 {
				currentConstraint = &Constraint{
					ID:   matches[1],
					Name: matches[2],
				}
			}
		} else if currentConstraint != nil && line != "" && !strings.HasPrefix(line, "###") {
			currentConstraint.Description += line + " "
		}
	}

	if currentConstraint != nil {
		constraints = append(constraints, currentConstraint)
	}

	for _, c := range constraints {
		c.Description = strings.TrimSpace(c.Description)
	}

	return constraints
}

func parseGovernanceRules(text string) []string {
	rules := make([]string, 0)

	if text == "" {
		return rules
	}

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			if strings.HasPrefix(line, "- ") {
				rules = append(rules, strings.TrimPrefix(line, "- "))
			} else if line != "" {
				rules = append(rules, line)
			}
		}
	}

	return rules
}

func (c *Constitution) ValidateAgainst(code string) []Violation {
	violations := make([]Violation, 0)

	for _, principle := range c.Principles {
		if principle.Description != "" {
			if !strings.Contains(code, principle.Name) {
				violations = append(violations, Violation{
					Type:        "PRINCIPLE_MISSING",
					Description: fmt.Sprintf("Principle %s (%s) not reflected in code", principle.ID, principle.Name),
					Severity:    "WARNING",
				})
			}
		}
	}

	for _, constraint := range c.Constraints {
		if constraint.Description != "" {
			violations = append(violations, Violation{
				Type:        "CONSTRAINT_CHECK",
				Description: fmt.Sprintf("Constraint %s (%s) requires manual verification", constraint.ID, constraint.Name),
				Severity:    "INFO",
			})
		}
	}

	return violations
}
