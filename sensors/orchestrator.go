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

// OrchestratorResult represents the output of a file scan.
type OrchestratorResult struct {
	FilePath        string                 `json:"file_path"`
	Language        string                 `json:"language"`
	ToolingDetected bool                   `json:"tooling_detected"`
	Metrics         MaintainabilityMetrics `json:"metrics"`
	Message         string                 `json:"message,omitempty"`
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

	reComplexity := regexp.MustCompile(`Complexity is (\d+)`)
	reParams := regexp.MustCompile(`More than (\d+) parameters`)

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
