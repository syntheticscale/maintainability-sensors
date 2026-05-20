package sensors

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// MaintainabilityMetrics holds precise maintainability scores.
type MaintainabilityMetrics struct {
	Complexity     int `json:"complexity"`
	FunctionLength int `json:"function_length"`
	ArgumentCount  int `json:"argument_count"`
}

// RelaxedLimit represents a threshold that has been configured to be looser than our standard.
type RelaxedLimit struct {
	RuleName      string `json:"rule_name"`
	ConfiguredVal int    `json:"configured_val"`
	BaselineVal   int    `json:"baseline_val"`
}

// OrchestratorResult represents the output of a file scan.
type OrchestratorResult struct {
	FilePath        string                 `json:"file_path"`
	Language        string                 `json:"language"`
	ToolingDetected bool                   `json:"tooling_detected"`
	Metrics         MaintainabilityMetrics `json:"metrics"`
	Message         string                 `json:"message,omitempty"`
	Exceptions      []RelaxedLimit         `json:"exceptions,omitempty"`
}

func sanitizePath(path string) (string, error) {
	// Reject null bytes outright
	if strings.Contains(path, "\x00") {
		return "", fmt.Errorf("invalid path: contains null byte")
	}

	clean := filepath.Clean(path)

	// Reject paths that traverse above the root after cleaning.
	// filepath.Clean("../foo") stays "../foo". We reject if it starts with "..".
	if strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("invalid path: traversal outside current directory denied")
	}

	return clean, nil
}

// OrchestratedScan scans a specific file. It auto-detects configuration and runs local tooling
// (ESLint/PyLint/Go AST) or falls back to Level 0 ("Working Blind").
func OrchestratedScan(filePath string) (OrchestratorResult, error) {
	filePath, err := sanitizePath(filePath)
	if err != nil {
		return OrchestratorResult{}, err
	}

	lang := detectLanguage(filePath)
	if lang == "" {
		return OrchestratorResult{FilePath: filePath}, fmt.Errorf("unsupported or unrecognized language file: %s", filePath)
	}

	result := OrchestratorResult{FilePath: filePath, Language: lang}

	if lang == "go" {
		return scanGoFile(result, filePath)
	}
	if isExternalToolingRequired(lang) {
		return handleExternalToolingOnly(result, filePath, lang)
	}

	return scanWithLocalAnalyzers(result, filePath, lang)
}

func scanGoFile(result OrchestratorResult, filePath string) (OrchestratorResult, error) {
	metrics, err := ParseGoAST(filePath)
	if err != nil {
		return result, fmt.Errorf("native Go AST parse error: %w", err)
	}
	result.ToolingDetected = true
	result.Metrics = MaintainabilityMetrics{
		Complexity:     metrics.Complexity,
		FunctionLength: metrics.FunctionLength,
		ArgumentCount:  metrics.ArgumentCount,
	}
	if configAnchor := detectConfig(filePath, "go"); configAnchor != "" {
		result.Exceptions = detectRelaxedLimits(configAnchor, "go")
	}
	return result, nil
}

func isExternalToolingRequired(lang string) bool {
	return lang == "csharp" || lang == "java"
}

func handleExternalToolingOnly(result OrchestratorResult, filePath, lang string) (OrchestratorResult, error) {
	result.ToolingDetected = false
	if lang == "csharp" {
		result.Message = fmt.Sprintf("[WARNING] C# analysis requires external tooling (e.g., dotnet build with Roslyn analyzers or IDE analyzers). Native parsing is not supported for '%s'.", filepath.Base(filePath))
	} else if lang == "java" {
		result.Message = fmt.Sprintf("[WARNING] Java analysis requires external tooling (e.g., Checkstyle via Maven/Gradle). Native parsing is not supported for '%s'.", filepath.Base(filePath))
	}
	fmt.Fprintln(os.Stderr, result.Message)
	return result, nil
}

func scanWithLocalAnalyzers(result OrchestratorResult, filePath, lang string) (OrchestratorResult, error) {
	configAnchor := detectConfig(filePath, lang)
	if configAnchor == "" {
		result.ToolingDetected = false
		result.Message = fmt.Sprintf("[WARNING] RUNNING BLIND (Level 0) on '%s'. No local %s static analysis config detected. Run 'bootstrap' command to fix.", filepath.Base(filePath), strings.ToUpper(lang))
		fmt.Fprintln(os.Stderr, result.Message)
		return result, nil
	}

	result.ToolingDetected = true
	result.Exceptions = detectRelaxedLimits(configAnchor, lang)

	var metrics MaintainabilityMetrics
	var err error

	switch lang {
	case "typescript", "javascript":
		metrics, err = runESLint(filePath)
		if err != nil {
			result.Message = fmt.Sprintf("Orchestration error (ESLint): %v. (Verify dependencies are installed)", err)
		}
	case "python":
		metrics, err = runPyLint(filePath)
		if err != nil {
			result.Message = fmt.Sprintf("Orchestration error (PyLint): %v. (Verify pylint is installed)", err)
		}
	case "ruby":
		metrics, err = runRuboCop(filePath)
		if err != nil {
			result.Message = fmt.Sprintf("Orchestration error (RuboCop): %v. (Verify rubocop is installed)", err)
		}
	}

	if err == nil {
		result.Metrics = metrics
	}
	return result, nil
}

func detectLanguage(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".py":
		return "python"
	case ".go":
		return "go"
	case ".rb":
		return "ruby"
	case ".java":
		return "java"
	case ".cs":
		return "csharp"
	}
	return ""
}

// parserRegistry maps a language string to its ConfigParser implementation.
func getParserForLang(lang string) ConfigParser {
	switch lang {
	case "typescript", "javascript":
		return ESLintConfigParser{}
	case "python":
		return PyLintConfigParser{}
	case "go":
		return GoConfigParser{}
	case "ruby":
		return RuboCopConfigParser{}
	}
	return nil
}

func detectConfig(filePath string, lang string) string {
	parser := getParserForLang(lang)
	if parser == nil {
		return ""
	}

	dir := filepath.Dir(filePath)
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}

	anchors := parser.Anchors()

	for {
		for _, anchor := range anchors {
			p := filepath.Join(absDir, anchor)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		parent := filepath.Dir(absDir)
		if parent == absDir {
			break
		}
		absDir = parent
	}
	return ""
}

func runLintCommand(name string, args ...string) ([]byte, int, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()

	if errors.Is(err, exec.ErrNotFound) {
		return nil, 0, fmt.Errorf("%s not found in PATH", name)
	}

	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		return output, 0, fmt.Errorf("failed to run %s: %w", name, err)
	}

	return output, exitCode, nil
}

func updateMetric(metric *int, valStr string) {
	var val int
	fmt.Sscanf(valStr, "%d", &val)
	if val > *metric {
		*metric = val
	}
}

func runESLint(filePath string) (MaintainabilityMetrics, error) {
	var metrics MaintainabilityMetrics

	filePath, err := sanitizePath(filePath)
	if err != nil {
		return metrics, err
	}

	output, exitCode, err := runLintCommand("npx", "eslint", "-f", "json", filePath)
	if err != nil && exitCode == 0 {
		return metrics, fmt.Errorf("ESLint error: %w", err)
	}

	if exitCode == 0 {
		return metrics, nil
	}

	if exitCode == 1 {
		if parseErr := parseESLintOutput(output, &metrics); parseErr == nil {
			return metrics, nil
		}
		return metrics, fmt.Errorf("ESLint crashed or encountered a configuration error (exit code 1): %s", strings.TrimSpace(string(output)))
	}

	return metrics, fmt.Errorf("ESLint exited with unexpected code %d: %s", exitCode, strings.TrimSpace(string(output)))
}

func parseESLintOutput(output []byte, metrics *MaintainabilityMetrics) error {
	var list []struct {
		Messages []struct {
			RuleID   string `json:"ruleId"`
			Message  string `json:"message"`
			Line     int    `json:"line"`
			Severity int    `json:"severity"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(output, &list); err != nil || len(list) == 0 {
		return fmt.Errorf("invalid json or empty")
	}

	reComplexity := regexp.MustCompile(`complexity of (\d+)`)
	reParams := regexp.MustCompile(`has (\d+) parameters`)
	reLines := regexp.MustCompile(`exceeds (\d+) lines`)

	for _, result := range list {
		for _, msg := range result.Messages {
			if matches := reComplexity.FindStringSubmatch(msg.Message); matches != nil {
				updateMetric(&metrics.Complexity, matches[1])
			}
			if matches := reParams.FindStringSubmatch(msg.Message); matches != nil {
				updateMetric(&metrics.ArgumentCount, matches[1])
			}
			if matches := reLines.FindStringSubmatch(msg.Message); matches != nil {
				updateMetric(&metrics.FunctionLength, matches[1])
			}
		}
	}
	return nil
}

func runPyLint(filePath string) (MaintainabilityMetrics, error) {
	var metrics MaintainabilityMetrics

	filePath, err := sanitizePath(filePath)
	if err != nil {
		return metrics, err
	}

	output, exitCode, err := runLintCommand("pylint", "--output-format=json", filePath)
	if err != nil && exitCode == 0 {
		return metrics, fmt.Errorf("pylint error: %w", err)
	}

	if exitCode == 0 {
		return metrics, nil
	}

	if exitCode >= 1 {
		if parseErr := parsePyLintOutput(output, &metrics); parseErr == nil {
			return metrics, nil
		}
		return metrics, fmt.Errorf("pylint crashed or encountered a configuration error (exit code %d): %s", exitCode, strings.TrimSpace(string(output)))
	}

	return metrics, fmt.Errorf("pylint exited with unexpected code %d: %s", exitCode, strings.TrimSpace(string(output)))
}

func parsePyLintOutput(output []byte, metrics *MaintainabilityMetrics) error {
	var list []struct {
		Message string `json:"message"`
		Symbol  string `json:"symbol"`
		Line    int    `json:"line"`
	}
	if err := json.Unmarshal(output, &list); err != nil {
		return err
	}

	reComplexity := regexp.MustCompile(`McCabe rating is (\d+)`)
	reParams := regexp.MustCompile(`Too many arguments \((\d+)/`)

	for _, msg := range list {
		if msg.Symbol == "too-many-statements" {
			var val int
			fmt.Sscanf(msg.Message, "Too many statements (%d/50)", &val)
			if val > metrics.FunctionLength {
				metrics.FunctionLength = val
			}
		}
		if matches := reComplexity.FindStringSubmatch(msg.Message); matches != nil {
			updateMetric(&metrics.Complexity, matches[1])
		}
		if matches := reParams.FindStringSubmatch(msg.Message); matches != nil {
			updateMetric(&metrics.ArgumentCount, matches[1])
		}
	}
	return nil
}

func runRuboCop(filePath string) (MaintainabilityMetrics, error) {
	var metrics MaintainabilityMetrics

	filePath, err := sanitizePath(filePath)
	if err != nil {
		return metrics, err
	}

	output, exitCode, err := runLintCommand("rubocop", "--format", "json", filePath)
	if err != nil && exitCode == 0 {
		return metrics, fmt.Errorf("rubocop error: %w", err)
	}

	if exitCode == 0 {
		return metrics, nil
	}

	if exitCode == 1 {
		if parseErr := parseRuboCopOutput(output, &metrics); parseErr == nil {
			return metrics, nil
		}
		return metrics, fmt.Errorf("rubocop crashed or encountered a configuration error (exit code 1): %s", strings.TrimSpace(string(output)))
	}

	return metrics, fmt.Errorf("rubocop exited with unexpected code %d: %s", exitCode, strings.TrimSpace(string(output)))
}

func parseRuboCopOutput(output []byte, metrics *MaintainabilityMetrics) error {
	var result struct {
		Files []struct {
			Offenses []struct {
				CopName string `json:"cop_name"`
				Message string `json:"message"`
			} `json:"offenses"`
		} `json:"files"`
	}
	if err := json.Unmarshal(output, &result); err != nil || len(result.Files) == 0 {
		return fmt.Errorf("invalid json")
	}

	reVal := regexp.MustCompile(`\[(\d+)/`)

	for _, file := range result.Files {
		for _, off := range file.Offenses {
			var val int
			if matches := reVal.FindStringSubmatch(off.Message); matches != nil {
				fmt.Sscanf(matches[1], "%d", &val)
			}
			if val == 0 {
				continue
			}

			switch off.CopName {
			case "Metrics/CyclomaticComplexity":
				if val > metrics.Complexity {
					metrics.Complexity = val
				}
			case "Metrics/MethodLength":
				if val > metrics.FunctionLength {
					metrics.FunctionLength = val
				}
			case "Metrics/ParameterLists":
				if val > metrics.ArgumentCount {
					metrics.ArgumentCount = val
				}
			}
		}
	}
	return nil
}

func findMaxConfigVal(content string, ext string, keys []string) (int, bool) {
	for _, key := range keys {
		vals := findAllConfigVals(content, key, ext)
		if len(vals) > 0 {
			return maxOf(vals), true
		}
	}
	return 0, false
}

func detectRelaxedLimits(configPath string, lang string) []RelaxedLimit {
	var exceptions []RelaxedLimit
	if configPath == "" {
		return exceptions
	}
	parser := getParserForLang(lang)
	if parser == nil {
		return exceptions
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return exceptions
	}
	content := string(data)
	ext := filepath.Ext(configPath)

	for _, rule := range parser.Rules() {
		if foundVal, found := findMaxConfigVal(content, ext, rule.Keys); found && foundVal > rule.Baseline {
			exceptions = append(exceptions, RelaxedLimit{
				RuleName:      rule.RuleName,
				ConfiguredVal: foundVal,
				BaselineVal:   rule.Baseline,
			})
		}
	}

	return exceptions
}
