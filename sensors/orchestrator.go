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

	result := OrchestratorResult{
		FilePath: filePath,
		Language: detectLanguage(filePath),
	}

	if result.Language == "" {
		return result, fmt.Errorf("unsupported or unrecognized language file: %s", filePath)
	}

	// 1. If Go, run native high-assurance AST parsing instantly
	if result.Language == "go" {
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
		configAnchor := detectConfig(filePath, "go")
		if configAnchor != "" {
			result.Exceptions = detectRelaxedLimits(configAnchor, "go")
		}
		return result, nil
	}

	// 1b. C# requires external tooling (.NET analyzers). Native parsing is not supported.
	if result.Language == "csharp" {
		result.ToolingDetected = false
		result.Message = fmt.Sprintf("[WARNING] C# analysis requires external tooling (e.g., dotnet build with Roslyn analyzers or IDE analyzers). Native parsing is not supported for '%s'.", filepath.Base(filePath))
		fmt.Fprintln(os.Stderr, result.Message)
		return result, nil
	}

	// 1c. Java requires external tooling (Checkstyle/Maven/Gradle). Native parsing is not supported.
	if result.Language == "java" {
		result.ToolingDetected = false
		result.Message = fmt.Sprintf("[WARNING] Java analysis requires external tooling (e.g., Checkstyle via Maven/Gradle). Native parsing is not supported for '%s'.", filepath.Base(filePath))
		fmt.Fprintln(os.Stderr, result.Message)
		return result, nil
	}

	// 2. Walk up directory tree to search for config file anchors
	configAnchor := detectConfig(filePath, result.Language)
	if configAnchor == "" {
		// Level 0: Working Blind Mode
		result.ToolingDetected = false
		result.Message = fmt.Sprintf("[WARNING] RUNNING BLIND (Level 0) on '%s'. No local %s static analysis config detected. Run 'bootstrap' command to fix.", filepath.Base(filePath), strings.ToUpper(result.Language))
		fmt.Fprintln(os.Stderr, result.Message)
		return result, nil
	}

	result.ToolingDetected = true
	result.Exceptions = detectRelaxedLimits(configAnchor, result.Language)

	// 3. Subprocess execution of local analyzers
	switch result.Language {
	case "typescript", "javascript":
		metrics, err := runESLint(filePath)
		if err != nil {
			result.Message = fmt.Sprintf("Orchestration error (ESLint): %v. (Verify dependencies are installed)", err)
		} else {
			result.Metrics = metrics
		}
	case "python":
		metrics, err := runPyLint(filePath)
		if err != nil {
			result.Message = fmt.Sprintf("Orchestration error (PyLint): %v. (Verify pylint is installed)", err)
		} else {
			result.Metrics = metrics
		}
	case "ruby":
		metrics, err := runRuboCop(filePath)
		if err != nil {
			result.Message = fmt.Sprintf("Orchestration error (RuboCop): %v. (Verify rubocop is installed)", err)
		} else {
			result.Metrics = metrics
		}
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

func runESLint(filePath string) (MaintainabilityMetrics, error) {
	var metrics MaintainabilityMetrics

	filePath, err := sanitizePath(filePath)
	if err != nil {
		return metrics, err
	}

	// Run npx eslint -f json <file>
	cmd := exec.Command("npx", "eslint", "-f", "json", filePath)
	output, err := cmd.CombinedOutput()

	// Check if the command itself failed to start (e.g., npx not in PATH)
	if errors.Is(err, exec.ErrNotFound) {
		return metrics, fmt.Errorf("npx/eslint not found in PATH; please install Node.js and ESLint")
	}

	// Get exit code from the error
	exitErr, isExitError := err.(*exec.ExitError)
	exitCode := 0
	if isExitError {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		// Unknown error starting the command
		return metrics, fmt.Errorf("failed to run ESLint: %w", err)
	}

	// Exit code 0: no lint violations — return zero metrics (success)
	if exitCode == 0 {
		return metrics, nil
	}

	// Exit code 1: could be lint violations OR ESLint crashed
	if exitCode == 1 {
		var list []struct {
			Messages []struct {
				RuleID   string `json:"ruleId"`
				Message  string `json:"message"`
				Line     int    `json:"line"`
				Severity int    `json:"severity"`
			} `json:"messages"`
		}
		if jsonErr := json.Unmarshal(output, &list); jsonErr == nil && len(list) > 0 {
			// Valid JSON with lint violations — parse normally
			// Regex extractors for standard limits
			reComplexity := regexp.MustCompile(`complexity of (\d+)`)
			reParams := regexp.MustCompile(`has (\d+) parameters`)
			reLines := regexp.MustCompile(`exceeds (\d+) lines`)

			for _, result := range list {
				for _, msg := range result.Messages {
					if matches := reComplexity.FindStringSubmatch(msg.Message); matches != nil {
						var val int
						fmt.Sscanf(matches[1], "%d", &val)
						if val > metrics.Complexity {
							metrics.Complexity = val
						}
					}
					if matches := reParams.FindStringSubmatch(msg.Message); matches != nil {
						var val int
						fmt.Sscanf(matches[1], "%d", &val)
						if val > metrics.ArgumentCount {
							metrics.ArgumentCount = val
						}
					}
					if matches := reLines.FindStringSubmatch(msg.Message); matches != nil {
						var val int
						fmt.Sscanf(matches[1], "%d", &val)
						if val > metrics.FunctionLength {
							metrics.FunctionLength = val
						}
					}
				}
			}
			return metrics, nil
		}
		// Non-JSON output on exit code 1 means ESLint crashed
		outputStr := strings.TrimSpace(string(output))
		return metrics, fmt.Errorf("ESLint crashed or encountered a configuration error (exit code 1): %s", outputStr)
	}

	// Any other exit code: unknown failure
	return metrics, fmt.Errorf("ESLint exited with unexpected code %d: %s", exitCode, strings.TrimSpace(string(output)))
}

func runPyLint(filePath string) (MaintainabilityMetrics, error) {
	var metrics MaintainabilityMetrics

	filePath, err := sanitizePath(filePath)
	if err != nil {
		return metrics, err
	}

	// Run pylint --output-format=json <file>
	cmd := exec.Command("pylint", "--output-format=json", filePath)
	output, err := cmd.CombinedOutput()

	// Check if the command itself failed to start (e.g., pylint not in PATH)
	if errors.Is(err, exec.ErrNotFound) {
		return metrics, fmt.Errorf("pylint not found in PATH; please install pylint")
	}

	exitErr, isExitError := err.(*exec.ExitError)
	exitCode := 0
	if isExitError {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		return metrics, fmt.Errorf("failed to run pylint: %w", err)
	}

	// Exit code 0: no lint violations — return zero metrics (success)
	if exitCode == 0 {
		return metrics, nil
	}

	// Exit code >= 1: could be lint violations OR pylint crashed/config error
	if exitCode >= 1 {
		var list []struct {
			Message string `json:"message"`
			Symbol  string `json:"symbol"`
			Line    int    `json:"line"`
		}
		if jsonErr := json.Unmarshal(output, &list); jsonErr == nil {
			// Valid JSON with lint violations — parse normally
			reComplexity := regexp.MustCompile(`McCabe rating is (\d+)`)
			reParams := regexp.MustCompile(`Too many arguments \((\d+)/`)

			for _, msg := range list {
				if msg.Symbol == "too-many-statements" {
					// Extract lines of statements
					var val int
					fmt.Sscanf(msg.Message, "Too many statements (%d/50)", &val)
					if val > metrics.FunctionLength {
						metrics.FunctionLength = val
					}
				}
				if matches := reComplexity.FindStringSubmatch(msg.Message); matches != nil {
					var val int
					fmt.Sscanf(matches[1], "%d", &val)
					if val > metrics.Complexity {
						metrics.Complexity = val
					}
				}
				if matches := reParams.FindStringSubmatch(msg.Message); matches != nil {
					var val int
					fmt.Sscanf(matches[1], "%d", &val)
					if val > metrics.ArgumentCount {
						metrics.ArgumentCount = val
					}
				}
			}
			return metrics, nil
		}
		// Non-JSON output means pylint crashed
		outputStr := strings.TrimSpace(string(output))
		return metrics, fmt.Errorf("pylint crashed or encountered a configuration error (exit code %d): %s", exitCode, outputStr)
	}

	// Any other exit code: unknown failure
	return metrics, fmt.Errorf("pylint exited with unexpected code %d: %s", exitCode, strings.TrimSpace(string(output)))
}

func runRuboCop(filePath string) (MaintainabilityMetrics, error) {
	var metrics MaintainabilityMetrics

	filePath, err := sanitizePath(filePath)
	if err != nil {
		return metrics, err
	}

	// Run rubocop --format json <file>
	cmd := exec.Command("rubocop", "--format", "json", filePath)
	output, err := cmd.CombinedOutput()

	// Check if the command itself failed to start (e.g., rubocop not in PATH)
	if errors.Is(err, exec.ErrNotFound) {
		return metrics, fmt.Errorf("rubocop not found in PATH; please install rubocop")
	}

	exitErr, isExitError := err.(*exec.ExitError)
	exitCode := 0
	if isExitError {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		return metrics, fmt.Errorf("failed to run rubocop: %w", err)
	}

	// Exit code 0: no lint violations — return zero metrics (success)
	if exitCode == 0 {
		return metrics, nil
	}

	// Exit code 1: could be lint violations OR rubocop crashed/config error
	if exitCode == 1 {
		var result struct {
			Files []struct {
				Offenses []struct {
					CopName string `json:"cop_name"`
					Message string `json:"message"`
				} `json:"offenses"`
			} `json:"files"`
		}
		if jsonErr := json.Unmarshal(output, &result); jsonErr == nil && len(result.Files) > 0 {
			// Valid JSON with lint violations — parse normally
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
			return metrics, nil
		}
		// Non-JSON output on exit code 1 means rubocop crashed
		outputStr := strings.TrimSpace(string(output))
		return metrics, fmt.Errorf("rubocop crashed or encountered a configuration error (exit code 1): %s", outputStr)
	}

	// Any other exit code: unknown failure
	return metrics, fmt.Errorf("rubocop exited with unexpected code %d: %s", exitCode, strings.TrimSpace(string(output)))
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
		var foundVal int
		var found bool
		for _, key := range rule.Keys {
			vals := findAllConfigVals(content, key, ext)
			if len(vals) > 0 {
				foundVal = maxOf(vals)
				found = true
				break
			}
		}
		if found && foundVal > rule.Baseline {
			exceptions = append(exceptions, RelaxedLimit{
				RuleName:      rule.RuleName,
				ConfiguredVal: foundVal,
				BaselineVal:   rule.Baseline,
			})
		}
	}

	return exceptions
}
