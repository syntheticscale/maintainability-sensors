package sensors

import (
	"fmt"
	"strings"
)

type StandardRBPlugin struct{}

func (p StandardRBPlugin) Name() string {
	return "standardrb"
}

type StandardRBLocation struct {
	Line     int `json:"line"`
	LastLine int `json:"last_line"`
}

type StandardRBOffense struct {
	CopName  string             `json:"cop_name"`
	Message  string             `json:"message"`
	Location StandardRBLocation `json:"location"`
}

type StandardRBFile struct {
	Path     string              `json:"path"`
	Offenses []StandardRBOffense `json:"offenses"`
}

type StandardRBResult struct {
	Files []StandardRBFile `json:"files"`
}

func parseSingleStandardRBOffense(off StandardRBOffense, fileViolations *[]Violation) {
	var val int
	if strings.Contains(off.Message, "[") {
		fmt.Sscanf(off.Message, "%*[^[][%d/%*d]", &val)
	}
	if val == 0 {
		return
	}

	endLine := off.Location.LastLine
	if endLine == 0 {
		endLine = off.Location.Line + FallbackEndLineOffset
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
func parseStandardRBMessages(files []StandardRBFile) map[string][]Violation {
	metricsMap := make(map[string][]Violation)
	for _, file := range files {
		var violations []Violation
		for _, off := range file.Offenses {
			parseSingleStandardRBOffense(off, &violations)
		}
		if len(violations) > 0 {
			metricsMap[file.Path] = violations
		}
	}
	return metricsMap
}

func processStandardRBAnalyzeResult(exitCode int, result StandardRBResult, output []byte) (map[string][]Violation, error) {
	if exitCode == 0 || exitCode == 1 {
		metricsMap := make(map[string][]Violation)
		if len(result.Files) > 0 {
			metricsMap = parseStandardRBMessages(result.Files)
		}
		if exitCode == 1 && len(result.Files) == 0 && len(output) > 0 {
			return nil, fmt.Errorf("standardrb crashed or encountered a configuration error (exit code 1): %s", strings.TrimSpace(string(output)))
		}
		return metricsMap, nil
	}

	return nil, fmt.Errorf("standardrb exited with unexpected code %d: %s", exitCode, strings.TrimSpace(string(output)))
}

func (p StandardRBPlugin) Analyze(files []FileContext) (map[string][]Violation, error) {
	var filePaths []string
	for _, f := range files {
		filePaths = append(filePaths, f.Path)
	}

	args := []string{"--format", "json"}
	args = append(args, "--")
	args = append(args, filePaths...)

	var result StandardRBResult
	exitCode, output, err := runLintCommandJSON("standardrb", &result, args...)
	if execErr := checkLintExecutionError("standardrb", exitCode, output, err); execErr != nil {
		return nil, execErr
	}

	return processStandardRBAnalyzeResult(exitCode, result, output)
}
