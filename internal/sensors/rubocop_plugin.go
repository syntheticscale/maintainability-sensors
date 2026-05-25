package sensors

import (
	"fmt"
	"regexp"
	"strings"
)

type RuboCopPlugin struct{}

func (p RuboCopPlugin) Name() string {
	return "rubocop"
}

type RuboCopLocation struct {
	Line     int `json:"line"`
	LastLine int `json:"last_line"`
}

type RuboCopOffense struct {
	CopName  string          `json:"cop_name"`
	Message  string          `json:"message"`
	Location RuboCopLocation `json:"location"`
}

type RuboCopFile struct {
	Path     string           `json:"path"`
	Offenses []RuboCopOffense `json:"offenses"`
}

type RuboCopResult struct {
	Files []RuboCopFile `json:"files"`
}

//nolint:gocognit,cyclop // Highly cohesive mapping logic, splitting hurts readability
func parseSingleRuboCopOffense(off RuboCopOffense, reVal *regexp.Regexp, fileViolations *[]Violation) {
	var val int
	if strings.Contains(off.Message, "[") {
		if m := reVal.FindStringSubmatch(off.Message); len(m) > 1 {
			fmt.Sscanf(m[1], "%d", &val)
		}
	}
	if val == 0 {
		return
	}

	endLine := off.Location.LastLine
	if endLine == 0 {
		endLine = off.Location.Line + 100
	}

	switch off.CopName {
	case "Metrics/CyclomaticComplexity":
		*fileViolations = append(*fileViolations, Violation{RuleName: RuleComplexity, Value: val, StartLine: off.Location.Line, EndLine: endLine, Message: off.Message})
	case "Metrics/MethodLength":
		*fileViolations = append(*fileViolations, Violation{RuleName: RuleFunctionLength, Value: val, StartLine: off.Location.Line, EndLine: endLine, Message: off.Message})
	case "Metrics/ParameterLists":
		*fileViolations = append(*fileViolations, Violation{RuleName: RuleArgumentCount, Value: val, StartLine: off.Location.Line, EndLine: endLine, Message: off.Message})
	}
}

//nolint:gocognit,cyclop // Highly cohesive mapping logic, splitting hurts readability
func parseRuboCopMessages(files []RuboCopFile) map[string][]Violation {
	metricsMap := make(map[string][]Violation)
	reVal := regexp.MustCompile(`\[(\d+)/`)
	for _, file := range files {
		var violations []Violation
		for _, off := range file.Offenses {
			parseSingleRuboCopOffense(off, reVal, &violations)
		}
		if len(violations) > 0 {
			metricsMap[file.Path] = violations
		}
	}
	return metricsMap
}

func processRuboCopAnalyzeResult(exitCode int, result RuboCopResult, output []byte) (map[string][]Violation, error) {
	if exitCode == 0 || exitCode == 1 {
		metricsMap := make(map[string][]Violation)
		if len(result.Files) > 0 {
			metricsMap = parseRuboCopMessages(result.Files)
		}
		if exitCode == 1 && len(result.Files) == 0 && len(output) > 0 {
			return nil, fmt.Errorf("rubocop crashed or encountered a configuration error (exit code 1): %s", strings.TrimSpace(string(output)))
		}
		return metricsMap, nil
	}

	return nil, fmt.Errorf("rubocop exited with unexpected code %d: %s", exitCode, strings.TrimSpace(string(output)))
}

func (p RuboCopPlugin) Analyze(files []FileContext) (map[string][]Violation, error) {
	var filePaths []string
	for _, f := range files {
		filePaths = append(filePaths, f.Path)
	}

	args := []string{"--format", "json"}
	args = append(args, "--")
	args = append(args, filePaths...)

	var result RuboCopResult
	exitCode, output, err := runLintCommandJSON("rubocop", &result, args...)
	if err != nil {
		if exitCode > 0 {
			return nil, fmt.Errorf("rubocop crashed or encountered a configuration error (exit code %d): %v\n%s", exitCode, err, string(output))
		}
		return nil, fmt.Errorf("rubocop error: %w", err)
	}

	return processRuboCopAnalyzeResult(exitCode, result, output)
}
