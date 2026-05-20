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

	// 1b. If C# (csharp), run native high-assurance parsing instantly
	if result.Language == "csharp" {
		metrics, err := ParseCSharp(filePath)
		if err != nil {
			return result, fmt.Errorf("native C# parse error: %w", err)
		}
		result.ToolingDetected = true
		result.Metrics = MaintainabilityMetrics{
			Complexity:     metrics.Complexity,
			FunctionLength: metrics.FunctionLength,
			ArgumentCount:  metrics.ArgumentCount,
		}
		configAnchor := detectConfig(filePath, "csharp")
		if configAnchor != "" {
			result.Exceptions = detectRelaxedLimits(configAnchor, "csharp")
		}
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
	case ".cs":
		return "csharp"
	}
	return ""
}

func detectConfig(filePath string, lang string) string {
	dir := filepath.Dir(filePath)
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}

	var anchors []string
	switch lang {
	case "typescript", "javascript":
		anchors = []string{"package.json", ".eslintrc.json", ".eslintrc.js", ".eslintrc.yaml", ".eslintrc.yml", "eslint.config.js", "eslint.config.mjs"}
	case "python":
		anchors = []string{"pyproject.toml", ".pylintrc", "setup.cfg", "tox.ini"}
	case "go":
		anchors = []string{".golangci.yml", "golangci.yml"}
	case "ruby":
		anchors = []string{".rubocop.yml", "Gemfile"}
	case "csharp":
		anchors = []string{".editorconfig"}
	}

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
	data, err := os.ReadFile(configPath)
	if err != nil {
		return exceptions
	}
	content := string(data)

	// 1. Cyclomatic Complexity (baseline: 8)
	var complexityVal int
	var foundComplexity bool
	if lang == "typescript" || lang == "javascript" {
		vals := findAllConfigVals(content, "complexity")
		if len(vals) > 0 {
			complexityVal = maxOf(vals)
			foundComplexity = true
		}
	} else if lang == "python" {
		vals := findAllConfigVals(content, "max-complexity")
		if len(vals) > 0 {
			complexityVal = maxOf(vals)
			foundComplexity = true
		}
	} else if lang == "go" {
		vals := findAllConfigVals(content, "min-complexity")
		if len(vals) > 0 {
			complexityVal = maxOf(vals)
			foundComplexity = true
		}
	} else if lang == "ruby" {
		vals := findAllConfigVals(content, "CyclomaticComplexity")
		if len(vals) > 0 {
			complexityVal = maxOf(vals)
			foundComplexity = true
		}
	} else if lang == "csharp" {
		vals := findAllConfigVals(content, "maximum_cyclomatic_complexity")
		if len(vals) > 0 {
			complexityVal = maxOf(vals)
			foundComplexity = true
		}
	}
	if foundComplexity && complexityVal > 8 {
		exceptions = append(exceptions, RelaxedLimit{
			RuleName:      "Cyclomatic Complexity",
			ConfiguredVal: complexityVal,
			BaselineVal:   8,
		})
	}

	// 2. Function Length (baseline: 50)
	var funcLenVal int
	var foundFuncLen bool
	if lang == "typescript" || lang == "javascript" {
		vals := findAllConfigVals(content, "max-lines-per-function")
		if len(vals) > 0 {
			funcLenVal = maxOf(vals)
			foundFuncLen = true
		}
	} else if lang == "python" {
		vals := findAllConfigVals(content, "max-statements")
		if len(vals) > 0 {
			funcLenVal = maxOf(vals)
			foundFuncLen = true
		}
	} else if lang == "go" {
		vals := findAllConfigVals(content, "lines")
		if len(vals) > 0 {
			funcLenVal = maxOf(vals)
			foundFuncLen = true
		}
	} else if lang == "ruby" {
		vals := findAllConfigVals(content, "MethodLength")
		if len(vals) > 0 {
			funcLenVal = maxOf(vals)
			foundFuncLen = true
		}
	}
	if foundFuncLen && funcLenVal > 50 {
		exceptions = append(exceptions, RelaxedLimit{
			RuleName:      "Function Length",
			ConfiguredVal: funcLenVal,
			BaselineVal:   50,
		})
	}

	// 3. Argument Count (baseline: 4)
	var argCountVal int
	var foundArgCount bool
	if lang == "typescript" || lang == "javascript" {
		vals := findAllConfigVals(content, "max-params")
		if len(vals) > 0 {
			argCountVal = maxOf(vals)
			foundArgCount = true
		}
	} else if lang == "python" {
		vals := findAllConfigVals(content, "max-args")
		if len(vals) > 0 {
			argCountVal = maxOf(vals)
			foundArgCount = true
		}
	} else if lang == "ruby" {
		vals := findAllConfigVals(content, "ParameterLists")
		if len(vals) > 0 {
			argCountVal = maxOf(vals)
			foundArgCount = true
		}
	}
	if foundArgCount && argCountVal > 4 {
		exceptions = append(exceptions, RelaxedLimit{
			RuleName:      "Argument Count",
			ConfiguredVal: argCountVal,
			BaselineVal:   4,
		})
	}

	// 4. File Length (baseline: 300)
	var fileLenVal int
	var foundFileLen bool
	if lang == "typescript" || lang == "javascript" {
		vals := findAllConfigVals(content, "max-lines")
		if len(vals) > 0 {
			fileLenVal = maxOf(vals)
			foundFileLen = true
		}
	} else if lang == "python" {
		vals := findAllConfigVals(content, "max-module-lines")
		if len(vals) > 0 {
			fileLenVal = maxOf(vals)
			foundFileLen = true
		}
	} else if lang == "ruby" {
		vals := findAllConfigVals(content, "ModuleLength")
		if len(vals) > 0 {
			fileLenVal = maxOf(vals)
			foundFileLen = true
		}
	}
	if foundFileLen && fileLenVal > 300 {
		exceptions = append(exceptions, RelaxedLimit{
			RuleName:      "File Length",
			ConfiguredVal: fileLenVal,
			BaselineVal:   300,
		})
	}

	return exceptions
}

func findAllConfigVals(content string, key string) []int {
	var pattern string
	if key == "max-lines" {
		pattern = `\bmax-lines\b[^-][^\d]{0,100}?(\d+)`
	} else {
		pattern = `\b` + regexp.QuoteMeta(key) + `\b[^\d]{0,100}?(\d+)`
	}
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(content, -1)
	var vals []int
	for _, m := range matches {
		if len(m) > 1 {
			var val int
			if _, err := fmt.Sscanf(m[1], "%d", &val); err == nil {
				vals = append(vals, val)
			}
		}
	}
	return vals
}

func maxOf(vals []int) int {
	if len(vals) == 0 {
		return 0
	}
	m := vals[0]
	for _, v := range vals {
		if v > m {
			m = v
		}
	}
	return m
}
