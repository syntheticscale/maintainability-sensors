package sensors

import (
	"fmt"
	"regexp"
	"strings"
)

type ESLintPlugin struct{}

func (p ESLintPlugin) Name() string {
	return "eslint"
}

type ESLintMessage struct {
	RuleID   string `json:"ruleId"`
	Message  string `json:"message"`
	Line     int    `json:"line"`
	EndLine  int    `json:"endLine"`
	Severity int    `json:"severity"`
}

type ESLintResult struct {
	FilePath string          `json:"filePath"`
	Messages []ESLintMessage `json:"messages"`
}

func extractESLintValue(msg string, primaryRe *regexp.Regexp, fallbackRe *regexp.Regexp) int {
	var val int
	if m := primaryRe.FindStringSubmatch(msg); len(m) > 1 {
		fmt.Sscanf(m[1], "%d", &val)
	} else if m := fallbackRe.FindStringSubmatch(msg); len(m) > 1 {
		fmt.Sscanf(m[1], "%d", &val)
	}
	return val
}

//nolint:gocognit,cyclop // Highly cohesive mapping logic for linters. Splitting this hurts readability.
func parseESLintMessages(messages []ESLintMessage) []Violation {
	var violations []Violation
	reComplexity := regexp.MustCompile(`complexity of (\d+)`)
	reParameters := regexp.MustCompile(`has (\d+) parameters`)
	reLines := regexp.MustCompile(`exceeds (\d+) lines`)
	reFallback := regexp.MustCompile(`(\d+)`)

	for _, msg := range messages {
		endLine := msg.EndLine
		if endLine == 0 {
			endLine = msg.Line + 100
		}
		if msg.RuleID == "complexity" {
			val := extractESLintValue(msg.Message, reComplexity, reFallback)
			violations = append(violations, Violation{RuleName: "Complexity", Value: val, StartLine: msg.Line, EndLine: endLine, Message: msg.Message})
		} else if msg.RuleID == "max-params" {
			val := extractESLintValue(msg.Message, reParameters, reFallback)
			violations = append(violations, Violation{RuleName: "ArgumentCount", Value: val, StartLine: msg.Line, EndLine: endLine, Message: msg.Message})
		} else if msg.RuleID == "max-lines-per-function" {
			val := extractESLintValue(msg.Message, reLines, reFallback)
			violations = append(violations, Violation{RuleName: "FunctionLength", Value: val, StartLine: msg.Line, EndLine: endLine, Message: msg.Message})
		}
	}
	return violations
}

func processESLintAnalyzeResult(exitCode int, list []ESLintResult, output []byte) (map[string][]Violation, error) {
	if exitCode == 0 || exitCode == 1 || exitCode == 2 {
		metricsMap := make(map[string][]Violation)
		if len(list) > 0 {
			for _, result := range list {
				violations := parseESLintMessages(result.Messages)
				if len(violations) > 0 {
					metricsMap[result.FilePath] = violations
				}
			}
		}
		return metricsMap, nil
	}

	return nil, fmt.Errorf("ESLint exited with unexpected code %d: %s", exitCode, strings.TrimSpace(string(output)))
}

func (p ESLintPlugin) Analyze(filePaths []string) (map[string][]Violation, error) {
	args := []string{"--no-install", "eslint", "-f", "json"}
	args = append(args, "--")
	args = append(args, filePaths...)
	var list []ESLintResult
	exitCode, output, err := runLintCommandJSON("npx", &list, args...)
	if err != nil {
		if exitCode > 0 {
			return nil, fmt.Errorf("ESLint crashed or encountered a configuration error (exit code %d): %v\n%s", exitCode, err, string(output))
		}
		return nil, fmt.Errorf("ESLint error: %w", err)
	}

	return processESLintAnalyzeResult(exitCode, list, output)
}
