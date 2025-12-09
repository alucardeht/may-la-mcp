package intel

import (
	"regexp"
	"strings"
)

type PatternType string

const (
	PatternSingleton PatternType = "SINGLETON"
	PatternFactory   PatternType = "FACTORY"
	PatternObserver  PatternType = "OBSERVER"
	PatternStrategy  PatternType = "STRATEGY"
	PatternDecorator PatternType = "DECORATOR"
	PatternAdapter   PatternType = "ADAPTER"
	PatternBuilder   PatternType = "BUILDER"
	PatternProxy     PatternType = "PROXY"
)

type CodePattern struct {
	Type        PatternType
	Name        string
	Location    string
	Confidence  float64
	Description string
	Indicators  []string
}

type ComplexityMetrics struct {
	CyclomaticComplexity int
	NestingDepth         int
	FunctionCount        int
	LineCount            int
	Ratio                float64
	Level                string
}

func DetectPatterns(content string) []CodePattern {
	var patterns []CodePattern

	patterns = append(patterns, detectSingletonPattern(content)...)
	patterns = append(patterns, detectFactoryPattern(content)...)
	patterns = append(patterns, detectObserverPattern(content)...)
	patterns = append(patterns, detectStrategyPattern(content)...)
	patterns = append(patterns, detectBuilderPattern(content)...)

	return patterns
}

func detectSingletonPattern(content string) []CodePattern {
	var patterns []CodePattern

	singletonRegex := regexp.MustCompile(`(?m)(private\s+static\s+[a-zA-Z_][a-zA-Z0-9_]*\s+instance|getInstance\s*\(|class\s+\w+.*\{\s*private\s+static\s+final\s+\w+)`)

	matches := singletonRegex.FindAllStringIndex(content, -1)
	if len(matches) > 0 {
		patterns = append(patterns, CodePattern{
			Type:       PatternSingleton,
			Name:       "Singleton",
			Confidence: 0.8,
			Indicators: []string{"static instance", "getInstance method", "private constructor"},
		})
	}

	return patterns
}

func detectFactoryPattern(content string) []CodePattern {
	var patterns []CodePattern

	factoryRegex := regexp.MustCompile(`(?m)(create[A-Z]\w+|make[A-Z]\w+|new[A-Z]\w+|.*Factory.*\{)`)

	lines := strings.Split(content, "\n")
	factoryCount := 0

	for _, line := range lines {
		if factoryRegex.MatchString(line) {
			factoryCount++
		}
	}

	if factoryCount > 2 {
		patterns = append(patterns, CodePattern{
			Type:       PatternFactory,
			Name:       "Factory",
			Confidence: float64(factoryCount) / 10.0,
			Indicators: []string{"create* methods", "object instantiation", "polymorphism"},
		})
	}

	return patterns
}

func detectObserverPattern(content string) []CodePattern {
	var patterns []CodePattern

	observerRegex := regexp.MustCompile(`(?m)(subscribe|unsubscribe|notify|addEventListener|removeEventListener|Observer|Listener)`)

	matches := observerRegex.FindAllStringIndex(content, -1)
	if len(matches) > 3 {
		patterns = append(patterns, CodePattern{
			Type:       PatternObserver,
			Name:       "Observer",
			Confidence: 0.7,
			Indicators: []string{"subscribe/unsubscribe", "notify", "event listeners"},
		})
	}

	return patterns
}

func detectStrategyPattern(content string) []CodePattern {
	var patterns []CodePattern

	strategyRegex := regexp.MustCompile(`(?m)(interface\s+\w*Strategy|class\s+\w*Strategy|execute|perform|strategy)`)

	matches := strategyRegex.FindAllString(content, -1)
	if len(matches) > 2 {
		patterns = append(patterns, CodePattern{
			Type:       PatternStrategy,
			Name:       "Strategy",
			Confidence: 0.6,
			Indicators: []string{"strategy interface", "multiple implementations", "polymorphic behavior"},
		})
	}

	return patterns
}

func detectBuilderPattern(content string) []CodePattern {
	var patterns []CodePattern

	builderRegex := regexp.MustCompile(`(?m)(Builder|\.with[A-Z]\w+|\.set[A-Z]\w+|\.build\(\))`)

	matches := builderRegex.FindAllString(content, -1)
	if len(matches) > 4 {
		patterns = append(patterns, CodePattern{
			Type:       PatternBuilder,
			Name:       "Builder",
			Confidence: 0.75,
			Indicators: []string{"with* methods", "set* methods", "build method", "fluent interface"},
		})
	}

	return patterns
}

func AnalyzeComplexity(content string) ComplexityMetrics {
	lines := strings.Split(content, "\n")

	metrics := ComplexityMetrics{
		LineCount:            len(lines),
		CyclomaticComplexity: calculateCyclomaticComplexity(content),
		NestingDepth:         calculateNestingDepth(content),
		FunctionCount:        countFunctions(content),
	}

	if metrics.LineCount > 0 {
		metrics.Ratio = float64(metrics.FunctionCount) / float64(metrics.LineCount)
	}

	metrics.Level = determineComplexityLevel(metrics)

	return metrics
}

func calculateCyclomaticComplexity(content string) int {
	complexity := 1

	conditions := []string{
		"if ", "else if ", "else ",
		"for ", "while ", "switch ",
		"case ", "catch ",
		"?",
		"||", "&&",
	}

	for _, cond := range conditions {
		complexity += strings.Count(content, cond)
	}

	return complexity
}

func calculateNestingDepth(content string) int {
	maxDepth := 0
	currentDepth := 0

	for _, char := range content {
		switch char {
		case '{', '[', '(':
			currentDepth++
			if currentDepth > maxDepth {
				maxDepth = currentDepth
			}
		case '}', ']', ')':
			if currentDepth > 0 {
				currentDepth--
			}
		}
	}

	return maxDepth
}

func countFunctions(content string) int {
	functionPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\bfunc\s+\w+`),
		regexp.MustCompile(`\bfunction\s+\w+`),
		regexp.MustCompile(`\bdef\s+\w+`),
		regexp.MustCompile(`\w+\s*\([^)]*\)\s*{`),
	}

	count := 0
	for _, pattern := range functionPatterns {
		count += len(pattern.FindAllString(content, -1))
	}

	return count
}

func determineComplexityLevel(metrics ComplexityMetrics) string {
	score := 0.0

	if metrics.CyclomaticComplexity > 20 {
		score += 3.0
	} else if metrics.CyclomaticComplexity > 10 {
		score += 2.0
	} else if metrics.CyclomaticComplexity > 5 {
		score += 1.0
	}

	if metrics.NestingDepth > 5 {
		score += 2.0
	} else if metrics.NestingDepth > 3 {
		score += 1.0
	}

	if metrics.Ratio > 0.2 {
		score += 1.0
	}

	switch {
	case score > 5:
		return "VERY_HIGH"
	case score > 3:
		return "HIGH"
	case score > 1.5:
		return "MEDIUM"
	case score > 0.5:
		return "LOW"
	default:
		return "VERY_LOW"
	}
}

func SuggestImprovements(content string) []string {
	var suggestions []string

	complexity := AnalyzeComplexity(content)

	if complexity.CyclomaticComplexity > 20 {
		suggestions = append(suggestions, "Extract some conditions into separate functions to reduce complexity")
	}

	if complexity.NestingDepth > 5 {
		suggestions = append(suggestions, "Reduce nesting depth by extracting nested logic into helper functions")
	}

	if complexity.FunctionCount == 0 && len(content) > 500 {
		suggestions = append(suggestions, "Consider breaking this into multiple functions")
	}

	longLines := countLongLines(content, 100)
	if longLines > len(strings.Split(content, "\n"))/4 {
		suggestions = append(suggestions, "Some lines are quite long, consider breaking them up for readability")
	}

	patterns := DetectPatterns(content)
	if len(patterns) == 0 && complexity.FunctionCount > 5 {
		suggestions = append(suggestions, "Consider using design patterns to improve code organization")
	}

	if strings.Count(content, "TODO") > 0 || strings.Count(content, "FIXME") > 0 {
		suggestions = append(suggestions, "Address TODO/FIXME comments in the code")
	}

	return suggestions
}

func countLongLines(content string, threshold int) int {
	lines := strings.Split(content, "\n")
	count := 0

	for _, line := range lines {
		if len(line) > threshold {
			count++
		}
	}

	return count
}

func (cp CodePattern) String() string {
	return string(cp.Type) + ": " + cp.Name +
		" (confidence: " + string(rune(int(cp.Confidence*100))) + "%)"
}

func (cm ComplexityMetrics) String() string {
	return "Complexity Level: " + cm.Level +
		" | Cyclomatic: " + string(rune(cm.CyclomaticComplexity)) +
		" | Nesting: " + string(rune(cm.NestingDepth))
}
