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
type ESLintRegexes struct {
	Complexity *regexp.Regexp
	Parameters *regexp.Regexp
	Lines      *regexp.Regexp
	Fallback   *regexp.Regexp
}

func parseESLintMessages(messages []ESLintMessage) []Violation {
	var violations []Violation
	regexes := ESLintRegexes{
		Complexity: regexp.MustCompile(`complexity of (\d+)`),
		Parameters: regexp.MustCompile(`has (\d+) parameters`),
		Lines:      regexp.MustCompile(`exceeds (\d+) lines`),
		Fallback:   regexp.MustCompile(`(\d+)`),
	}

	for _, msg := range messages {
		if v, ok := parseSingleESLintMessage(msg, regexes); ok {
			violations = append(violations, v)
		}
	}
	return violations
}

func parseSingleESLintMessage(msg ESLintMessage, regexes ESLintRegexes) (Violation, bool) {
	endLine := msg.EndLine
	if endLine == 0 {
		endLine = msg.Line + FallbackEndLineOffset
	}
	switch msg.RuleID {
	case "complexity":
		val := extractESLintValue(msg.Message, regexes.Complexity, regexes.Fallback)
		return Violation{RuleName: RuleComplexity, Value: val, StartLine: msg.Line, EndLine: endLine, Message: msg.Message}, true
	case "max-params":
		val := extractESLintValue(msg.Message, regexes.Parameters, regexes.Fallback)
		return Violation{RuleName: RuleArgumentCount, Value: val, StartLine: msg.Line, EndLine: endLine, Message: msg.Message}, true
	case "max-lines-per-function":
		val := extractESLintValue(msg.Message, regexes.Lines, regexes.Fallback)
		return Violation{RuleName: RuleFunctionLength, Value: val, StartLine: msg.Line, EndLine: endLine, Message: msg.Message}, true
	}
	return Violation{}, false
}

func processESLintAnalyzeResult(exitCode int, list []ESLintResult, output []byte) (map[string][]Violation, error) {
	if exitCode != 0 && exitCode != 1 && exitCode != 2 {
		return nil, fmt.Errorf("ESLint exited with unexpected code %d: %s", exitCode, strings.TrimSpace(string(output)))
	}

	metricsMap := make(map[string][]Violation)
	for _, result := range list {
		violations := parseESLintMessages(result.Messages)
		if len(violations) > 0 {
			metricsMap[result.FilePath] = violations
		}
	}
	return metricsMap, nil
}

func (p ESLintPlugin) Analyze(files []FileContext) (map[string][]Violation, error) {
	var filePaths []string
	for _, f := range files {
		filePaths = append(filePaths, f.Path)
	}

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
