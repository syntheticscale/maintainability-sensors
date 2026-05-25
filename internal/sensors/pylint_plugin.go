package sensors

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

//nolint:gocognit,cyclop
func parsePyLintMessage(msg PyLintMessage) (string, int) {
	var val int
	if msg.Symbol == "too-many-statements" && strings.Contains(msg.Message, "Too many statements") {
		fmt.Sscanf(msg.Message, "Too many statements (%d/%*d)", &val)
		return RuleFunctionLength, val
	}
	if msg.Symbol == "too-many-arguments" && strings.Contains(msg.Message, "Too many arguments") {
		fmt.Sscanf(msg.Message, "Too many arguments (%d/%*d)", &val)
		return RuleArgumentCount, val
	}
	if (msg.Symbol == "too-many-branches" || msg.Symbol == "too-complex") && strings.Contains(msg.Message, "McCabe rating is") {
		fmt.Sscanf(msg.Message, "McCabe rating is %d", &val)
		return RuleComplexity, val
	}
	return "", 0
}

func parsePyLintMessages(list []PyLintMessage) map[string][]Violation {
	metricsMap := make(map[string][]Violation)

	for _, msg := range list {
		rule, val := parsePyLintMessage(msg)
		if rule == "" {
			continue
		}
		endLine := msg.EndLine
		if endLine == 0 {
			endLine = msg.Line + FallbackEndLineOffset
		}
		metricsMap[msg.Path] = append(metricsMap[msg.Path], Violation{RuleName: rule, Value: val, StartLine: msg.Line, EndLine: endLine, Message: msg.Message})
	}
	return metricsMap
}

func handlePyLintError(exitCode int, output []byte) error {
	var dummy []interface{}
	if parseErr := json.Unmarshal(output, &dummy); parseErr != nil {
		return fmt.Errorf("pylint crashed or encountered a configuration error (exit code %d): %s", exitCode, strings.TrimSpace(string(output)))
	}
	return nil
}

func checkPyLintError(exitCode int, output []byte, err error) error {
	if err != nil {
		if exitCode > 0 {
			return fmt.Errorf("pylint crashed or encountered a configuration error (exit code %d): %v\n%s", exitCode, err, string(output))
		}
		return fmt.Errorf("pylint error: %w", err)
	}
	if exitCode < 0 {
		return fmt.Errorf("pylint exited with unexpected code %d: %s", exitCode, strings.TrimSpace(string(output)))
	}
	return nil
}

func (p PyLintPlugin) Analyze(files []FileContext) (map[string][]Violation, error) {
	var filePaths []string
	for _, f := range files {
		filePaths = append(filePaths, f.Path)
	}

	args := append([]string{"--output-format=json", "--"}, filePaths...)

	var list []PyLintMessage
	exitCode, output, err := runLintCommandJSON("pylint", &list, args...)
	if checkErr := checkPyLintError(exitCode, output, err); checkErr != nil {
		return nil, checkErr
	}

	metricsMap := make(map[string][]Violation)
	if len(list) > 0 {
		metricsMap = parsePyLintMessages(list)
	}
	if exitCode > 0 && len(metricsMap) == 0 {
		if err := handlePyLintError(exitCode, output); err != nil {
			return nil, err
		}
	}
	return metricsMap, nil
}
