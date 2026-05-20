package sensors

import (
	"encoding/json"
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

// OrchestratedScan scans a specific file. It auto-detects configuration and runs local tooling
// (ESLint/PyLint/Go AST) or falls back to Level 0 ("Working Blind").
func OrchestratedScan(filePath string) (OrchestratorResult, error) {
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

	// Run npx eslint -f json <file>
	cmd := exec.Command("npx", "eslint", "-f", "json", filePath)
	output, _ := cmd.CombinedOutput() // ignore exit 1 because lint failures return error code

	var list []struct {
		Messages []struct {
			RuleID   string `json:"ruleId"`
			Message  string `json:"message"`
			Line     int    `json:"line"`
			Severity int    `json:"severity"`
		} `json:"messages"`
	}

	if err := json.Unmarshal(output, &list); err != nil || len(list) == 0 {
		return metrics, fmt.Errorf("failed to parse ESLint JSON output: %w", err)
	}

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

func runPyLint(filePath string) (MaintainabilityMetrics, error) {
	var metrics MaintainabilityMetrics

	// Run pylint --output-format=json <file>
	cmd := exec.Command("pylint", "--output-format=json", filePath)
	output, _ := cmd.CombinedOutput()

	var list []struct {
		Message string `json:"message"`
		Symbol  string `json:"symbol"`
		Line    int    `json:"line"`
	}

	if err := json.Unmarshal(output, &list); err != nil {
		return metrics, fmt.Errorf("failed to parse PyLint JSON output: %w", err)
	}

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

func runRuboCop(filePath string) (MaintainabilityMetrics, error) {
	var metrics MaintainabilityMetrics

	// Run rubocop --format json <file>
	cmd := exec.Command("rubocop", "--format", "json", filePath)
	output, _ := cmd.CombinedOutput()

	var result struct {
		Files []struct {
			Offenses []struct {
				CopName string `json:"cop_name"`
				Message string `json:"message"`
			} `json:"offenses"`
		} `json:"files"`
	}

	if err := json.Unmarshal(output, &result); err != nil || len(result.Files) == 0 {
		return metrics, fmt.Errorf("failed to parse RuboCop JSON output: %w", err)
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

	return metrics, nil
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
