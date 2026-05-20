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
	"sync"

	"golang.org/x/sync/errgroup"
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
	if strings.Contains(path, "\x00") {
		return "", fmt.Errorf("invalid path: contains null byte")
	}
	clean := filepath.Clean(path)
	if strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("invalid path: traversal outside current directory denied")
	}
	return clean, nil
}

// OrchestratedScan scans a specific file. It is a convenience wrapper over OrchestratedScanBatch.
func OrchestratedScan(filePath string) (OrchestratorResult, error) {
	lang := DetectLanguage(filePath)
	if lang == "" {
		return OrchestratorResult{FilePath: filePath}, fmt.Errorf("unsupported or unrecognized language file: %s", filePath)
	}
	results, err := OrchestratedScanBatch([]string{filePath}, lang)
	if err != nil {
		return OrchestratorResult{FilePath: filePath, Language: lang}, err
	}
	if len(results) > 0 {
		return results[0], nil
	}
	return OrchestratorResult{FilePath: filePath, Language: lang}, nil
}

// OrchestratedScanBatch scans a batch of files for a specific language concurrently.
func OrchestratedScanBatch(filePaths []string, lang string) ([]OrchestratorResult, error) {
	if len(filePaths) == 0 {
		return nil, nil
	}

	validPaths := make([]string, 0, len(filePaths))
	originalPaths := make(map[string]string)

	for _, p := range filePaths {
		clean, err := sanitizePath(p)
		if err != nil {
			return nil, err
		}
		abs, err := filepath.Abs(clean)
		if err == nil {
			originalPaths[abs] = p
		}
		originalPaths[clean] = p
		validPaths = append(validPaths, clean)
	}

	if lang == "go" {
		var mu sync.Mutex
		var g errgroup.Group
		results := make([]OrchestratorResult, 0, len(validPaths))

		for _, cleanPath := range validPaths {
			cleanPath := cleanPath
			g.Go(func() error {
				res := OrchestratorResult{FilePath: originalPaths[cleanPath], Language: lang}
				res, err := scanGoFile(res, cleanPath)
				if err != nil {
					return err
				}
				mu.Lock()
				results = append(results, res)
				mu.Unlock()
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return nil, err
		}
		return results, nil
	}

	if isExternalToolingRequired(lang) {
		var results []OrchestratorResult
		for _, cleanPath := range validPaths {
			res := OrchestratorResult{FilePath: originalPaths[cleanPath], Language: lang}
			res, _ = handleExternalToolingOnly(res, cleanPath, lang)
			results = append(results, res)
		}
		return results, nil
	}

	return scanWithLocalAnalyzersBatch(validPaths, originalPaths, lang)
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

func scanWithLocalAnalyzersBatch(validPaths []string, originalPaths map[string]string, lang string) ([]OrchestratorResult, error) {
	var results []OrchestratorResult

	var configuredPaths []string
	var blindPaths []string
	configAnchors := make(map[string]string)

	for _, cleanPath := range validPaths {
		anchor := detectConfig(cleanPath, lang)
		if anchor == "" {
			blindPaths = append(blindPaths, cleanPath)
		} else {
			configuredPaths = append(configuredPaths, cleanPath)
			configAnchors[cleanPath] = anchor
		}
	}

	for _, cleanPath := range blindPaths {
		orig := originalPaths[cleanPath]
		res := OrchestratorResult{
			FilePath:        orig,
			Language:        lang,
			ToolingDetected: false,
			Message:         fmt.Sprintf("[WARNING] RUNNING BLIND (Level 0) on '%s'. No local %s static analysis config detected. Run 'bootstrap' command to fix.", filepath.Base(cleanPath), strings.ToUpper(lang)),
		}
		fmt.Fprintln(os.Stderr, res.Message)
		results = append(results, res)
	}

	if len(configuredPaths) == 0 {
		return results, nil
	}

	var metricsMap map[string]MaintainabilityMetrics
	var err error
	var batchErrorMsg string

	switch lang {
	case "typescript", "javascript":
		metricsMap, err = runESLintBatch(configuredPaths)
		if err != nil {
			batchErrorMsg = fmt.Sprintf("Orchestration error (ESLint): %v. (Verify dependencies are installed)", err)
		}
	case "python":
		metricsMap, err = runPyLintBatch(configuredPaths)
		if err != nil {
			batchErrorMsg = fmt.Sprintf("Orchestration error (PyLint): %v. (Verify pylint is installed)", err)
		}
	case "ruby":
		metricsMap, err = runRuboCopBatch(configuredPaths)
		if err != nil {
			batchErrorMsg = fmt.Sprintf("Orchestration error (RuboCop): %v. (Verify rubocop is installed)", err)
		}
	}

	for _, cleanPath := range configuredPaths {
		orig := originalPaths[cleanPath]
		res := OrchestratorResult{
			FilePath:        orig,
			Language:        lang,
			ToolingDetected: true,
		}

		res.Exceptions = detectRelaxedLimits(configAnchors[cleanPath], lang)

		if batchErrorMsg != "" {
			res.Message = batchErrorMsg
		} else {
			var m MaintainabilityMetrics
			found := false

			absClean, _ := filepath.Abs(cleanPath)

			for outPath, outMetrics := range metricsMap {
				outAbs, _ := filepath.Abs(outPath)
				if outAbs == absClean || outPath == cleanPath || strings.HasSuffix(outAbs, cleanPath) {
					m = outMetrics
					found = true
					break
				}
			}
			if found {
				res.Metrics = m
			}
		}
		results = append(results, res)
	}

	return results, nil
}

func DetectLanguage(path string) string {
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

func runESLintBatch(filePaths []string) (map[string]MaintainabilityMetrics, error) {
	args := []string{"eslint", "-f", "json"}
	args = append(args, filePaths...)
	output, exitCode, err := runLintCommand("npx", args...)
	if err != nil && exitCode == 0 {
		return nil, fmt.Errorf("ESLint error: %w", err)
	}

	if exitCode == 0 || exitCode == 1 {
		var metricsMap map[string]MaintainabilityMetrics
		if len(output) > 0 {
			metricsMap, err = parseESLintOutputBatch(output)
			if err != nil && exitCode == 1 {
				return nil, fmt.Errorf("ESLint crashed or encountered a configuration error (exit code 1): %s", strings.TrimSpace(string(output)))
			}
		}
		if metricsMap == nil {
			metricsMap = make(map[string]MaintainabilityMetrics)
		}
		return metricsMap, nil
	}

	return nil, fmt.Errorf("ESLint exited with unexpected code %d: %s", exitCode, strings.TrimSpace(string(output)))
}

func parseESLintOutputBatch(output []byte) (map[string]MaintainabilityMetrics, error) {
	var list []struct {
		FilePath string `json:"filePath"`
		Messages []struct {
			RuleID   string `json:"ruleId"`
			Message  string `json:"message"`
			Line     int    `json:"line"`
			Severity int    `json:"severity"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(output, &list); err != nil {
		return nil, err
	}

	metricsMap := make(map[string]MaintainabilityMetrics)
	reComplexity := regexp.MustCompile(`complexity of (\d+)`)
	reParams := regexp.MustCompile(`has (\d+) parameters`)
	reLines := regexp.MustCompile(`exceeds (\d+) lines`)

	for _, result := range list {
		var metrics MaintainabilityMetrics
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
		metricsMap[result.FilePath] = metrics
	}
	return metricsMap, nil
}

func runPyLintBatch(filePaths []string) (map[string]MaintainabilityMetrics, error) {
	args := []string{"--output-format=json"}
	args = append(args, filePaths...)
	output, exitCode, err := runLintCommand("pylint", args...)
	if err != nil && exitCode == 0 {
		return nil, fmt.Errorf("pylint error: %w", err)
	}

	if exitCode >= 0 {
		var metricsMap map[string]MaintainabilityMetrics
		if len(output) > 0 {
			metricsMap, _ = parsePyLintOutputBatch(output)
		}
		if metricsMap == nil {
			metricsMap = make(map[string]MaintainabilityMetrics)
		}
		if exitCode > 0 && len(metricsMap) == 0 {
			if parseErr := json.Unmarshal(output, &[]interface{}{}); parseErr != nil {
				return nil, fmt.Errorf("pylint crashed or encountered a configuration error (exit code %d): %s", exitCode, strings.TrimSpace(string(output)))
			}
		}
		return metricsMap, nil
	}

	return nil, fmt.Errorf("pylint exited with unexpected code %d: %s", exitCode, strings.TrimSpace(string(output)))
}

func parsePyLintOutputBatch(output []byte) (map[string]MaintainabilityMetrics, error) {
	var list []struct {
		Path    string `json:"path"`
		Message string `json:"message"`
		Symbol  string `json:"symbol"`
	}
	if err := json.Unmarshal(output, &list); err != nil {
		return nil, err
	}

	metricsMap := make(map[string]MaintainabilityMetrics)
	reComplexity := regexp.MustCompile(`McCabe rating is (\d+)`)
	reParams := regexp.MustCompile(`Too many arguments \((\d+)/`)
	reStatements := regexp.MustCompile(`Too many statements \((\d+)/`)

	for _, msg := range list {
		metrics := metricsMap[msg.Path]
		if msg.Symbol == "too-many-statements" {
			var val int
			if matches := reStatements.FindStringSubmatch(msg.Message); matches != nil {
				fmt.Sscanf(matches[1], "%d", &val)
				if val > metrics.FunctionLength {
					metrics.FunctionLength = val
				}
			}
		}
		if matches := reComplexity.FindStringSubmatch(msg.Message); matches != nil {
			updateMetric(&metrics.Complexity, matches[1])
		}
		if matches := reParams.FindStringSubmatch(msg.Message); matches != nil {
			updateMetric(&metrics.ArgumentCount, matches[1])
		}
		metricsMap[msg.Path] = metrics
	}
	return metricsMap, nil
}

func runRuboCopBatch(filePaths []string) (map[string]MaintainabilityMetrics, error) {
	args := []string{"--format", "json"}
	args = append(args, filePaths...)
	output, exitCode, err := runLintCommand("rubocop", args...)
	if err != nil && exitCode == 0 {
		return nil, fmt.Errorf("rubocop error: %w", err)
	}

	if exitCode == 0 || exitCode == 1 {
		var metricsMap map[string]MaintainabilityMetrics
		if len(output) > 0 {
			metricsMap, err = parseRuboCopOutputBatch(output)
			if err != nil && exitCode == 1 {
				return nil, fmt.Errorf("rubocop crashed or encountered a configuration error (exit code 1): %s", strings.TrimSpace(string(output)))
			}
		}
		if metricsMap == nil {
			metricsMap = make(map[string]MaintainabilityMetrics)
		}
		return metricsMap, nil
	}

	return nil, fmt.Errorf("rubocop exited with unexpected code %d: %s", exitCode, strings.TrimSpace(string(output)))
}

func parseRuboCopOutputBatch(output []byte) (map[string]MaintainabilityMetrics, error) {
	var result struct {
		Files []struct {
			Path     string `json:"path"`
			Offenses []struct {
				CopName string `json:"cop_name"`
				Message string `json:"message"`
			} `json:"offenses"`
		} `json:"files"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	metricsMap := make(map[string]MaintainabilityMetrics)
	reVal := regexp.MustCompile(`\[(\d+)/`)

	for _, file := range result.Files {
		var metrics MaintainabilityMetrics
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
		metricsMap[file.Path] = metrics
	}
	return metricsMap, nil
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
