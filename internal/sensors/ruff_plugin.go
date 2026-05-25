package sensors

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type RuffPlugin struct{}

func (p RuffPlugin) Name() string {
	return "ruff"
}

type RuffMessage struct {
	Filename string `json:"filename"`
	Message  string `json:"message"`
	Code     string `json:"code"`
	Location struct {
		Row int `json:"row"`
	} `json:"location"`
	EndLocation struct {
		Row int `json:"row"`
	} `json:"end_location"`
}

func extractRuffVal(msg RuffMessage, reVal *regexp.Regexp) int {
	var val int
	if strings.Contains(msg.Message, "(") && strings.Contains(msg.Message, ">") {
		if m := reVal.FindStringSubmatch(msg.Message); len(m) > 1 {
			fmt.Sscanf(m[1], "%d", &val)
		}
	}
	return val
}

func extractRuffRule(msg RuffMessage, val *int) string {
	if msg.Code == "C901" || strings.HasPrefix(msg.Code, "C90") {
		if *val == 0 {
			*val = 1
		}
		return RuleComplexity
	}
	if *val > 0 {
		if msg.Code == "PLR0915" {
			return RuleFunctionLength
		} else if msg.Code == "PLR0913" {
			return RuleArgumentCount
		}
	}
	return ""
}

func extractRuffRuleAndVal(msg RuffMessage, reVal *regexp.Regexp) (string, int) {
	val := extractRuffVal(msg, reVal)
	rule := extractRuffRule(msg, &val)
	return rule, val
}

func parseSingleRuffMessage(msg RuffMessage, reVal *regexp.Regexp, fileViolations *[]Violation) {
	rule, val := extractRuffRuleAndVal(msg, reVal)

	if rule != "" {
		endLine := msg.EndLocation.Row
		if endLine == 0 {
			endLine = msg.Location.Row + FallbackEndLineOffset
		}
		*fileViolations = append(*fileViolations, Violation{RuleName: rule, Value: val, StartLine: msg.Location.Row, EndLine: endLine, Message: msg.Message})
	}
}

//nolint:gocognit,cyclop // Highly cohesive mapping logic, splitting hurts readability
func parseRuffMessages(list []RuffMessage) map[string][]Violation {
	metricsMap := make(map[string][]Violation)
	reVal := regexp.MustCompile(`\((\d+)\s*>`)
	for _, msg := range list {
		var violations []Violation
		parseSingleRuffMessage(msg, reVal, &violations)
		if len(violations) > 0 {
			metricsMap[msg.Filename] = append(metricsMap[msg.Filename], violations...)
		}
	}
	return metricsMap
}

func (p RuffPlugin) Analyze(files []FileContext) (map[string][]Violation, error) {
	var filePaths []string
	for _, f := range files {
		filePaths = append(filePaths, f.Path)
	}

	args := []string{"check", "--output-format=json"}
	args = append(args, "--")
	args = append(args, filePaths...)

	var list []RuffMessage
	exitCode, output, err := runLintCommandJSON("ruff", &list, args...)
	if execErr := checkLintExecutionError("ruff", exitCode, output, err); execErr != nil {
		return nil, execErr
	}

	if exitCode >= 0 {
		metricsMap := make(map[string][]Violation)
		if len(list) > 0 {
			metricsMap = parseRuffMessages(list)
		}
		if exitCode > 0 && len(metricsMap) == 0 {
			var dummy []interface{}
			if parseErr := json.Unmarshal(output, &dummy); parseErr != nil {
				return nil, fmt.Errorf("ruff crashed or encountered a configuration error (exit code %d): %s", exitCode, strings.TrimSpace(string(output)))
			}
		}
		return metricsMap, nil
	}

	return nil, fmt.Errorf("ruff exited with unexpected code %d: %s", exitCode, strings.TrimSpace(string(output)))
}
