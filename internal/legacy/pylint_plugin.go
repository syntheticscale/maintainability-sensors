package legacy

import (
	"encoding/json"
	"fmt"
	"strings"
)

type PyLintPlugin struct{}

func (p PyLintPlugin) Name() string {
	return "pylint"
}

type PyLintMessage struct {
	Path    string `json:"path"`
	Message string `json:"message"`
	Symbol  string `json:"symbol"`
	Line    int    `json:"line"`
	EndLine int    `json:"endLine"`
}

//nolint:gocognit,cyclop // Highly cohesive mapping logic, splitting hurts readability
func parsePyLintMessages(list []PyLintMessage) map[string][]Violation {
	metricsMap := make(map[string][]Violation)

	for _, msg := range list {
		var val int
		var rule string
		if msg.Symbol == "too-many-statements" {
			if strings.Contains(msg.Message, "Too many statements") {
				fmt.Sscanf(msg.Message, "Too many statements (%d/%*d)", &val)
				rule = RuleFunctionLength
			}
		} else if msg.Symbol == "too-many-arguments" {
			if strings.Contains(msg.Message, "Too many arguments") {
				fmt.Sscanf(msg.Message, "Too many arguments (%d/%*d)", &val)
				rule = RuleArgumentCount
			}
		} else if msg.Symbol == "too-many-branches" || msg.Symbol == "too-complex" {
			if strings.Contains(msg.Message, "McCabe rating is") {
				fmt.Sscanf(msg.Message, "McCabe rating is %d", &val)
				rule = RuleComplexity
			}
		}
		if rule != "" {
			endLine := msg.EndLine
			if endLine == 0 {
				endLine = msg.Line + 100
			}
			metricsMap[msg.Path] = append(metricsMap[msg.Path], Violation{RuleName: rule, Value: val, StartLine: msg.Line, EndLine: endLine, Message: msg.Message})
		}
	}
	return metricsMap
}

func (p PyLintPlugin) Analyze(files []FileContext) (map[string][]Violation, error) {
	var filePaths []string
	for _, f := range files {
		filePaths = append(filePaths, f.Path)
	}

	args := []string{"--output-format=json"}
	args = append(args, "--")
	args = append(args, filePaths...)

	var list []PyLintMessage
	exitCode, output, err := runLintCommandJSON("pylint", &list, args...)
	if err != nil {
		if exitCode > 0 {
			return nil, fmt.Errorf("pylint crashed or encountered a configuration error (exit code %d): %v\n%s", exitCode, err, string(output))
		}
		return nil, fmt.Errorf("pylint error: %w", err)
	}

	if exitCode >= 0 {
		metricsMap := make(map[string][]Violation)
		if len(list) > 0 {
			metricsMap = parsePyLintMessages(list)
		}
		if exitCode > 0 && len(metricsMap) == 0 {
			// To catch crashes
			var dummy []interface{}
			if parseErr := json.Unmarshal(output, &dummy); parseErr != nil {
				return nil, fmt.Errorf("pylint crashed or encountered a configuration error (exit code %d): %s", exitCode, strings.TrimSpace(string(output)))
			}
		}
		return metricsMap, nil
	}

	return nil, fmt.Errorf("pylint exited with unexpected code %d: %s", exitCode, strings.TrimSpace(string(output)))
}
