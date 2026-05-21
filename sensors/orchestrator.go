package sensors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

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

	plugins := GlobalRegistry.GetPlugins(lang)
	if len(plugins) == 0 {
		results := make([]OrchestratorResult, 0, len(validPaths))
		for _, cleanPath := range validPaths {
			results = append(results, OrchestratorResult{
				FilePath:        originalPaths[cleanPath],
				Language:        lang,
				ToolingDetected: false,
			})
		}
		return results, nil
	}

	configAnchors := make(map[string]string)
	toolByPath := make(map[string]ConfigParser)

	for _, cleanPath := range validPaths {
		anchor, parser := detectConfigAndParser(cleanPath, lang)
		if anchor != "" {
			configAnchors[cleanPath] = anchor
			toolByPath[cleanPath] = parser
		}
	}

	metricsMap := make(map[string]MaintainabilityMetrics)
	var batchErrorMsg string

	pathsRemaining := validPaths

	for _, plugin := range plugins {
		if len(pathsRemaining) == 0 {
			break
		}

		pluginName := plugin.Name()
		isNative := strings.HasSuffix(pluginName, "-ast")

		var pathsForPlugin []string
		for _, p := range pathsRemaining {
			if isNative {
				pathsForPlugin = append(pathsForPlugin, p)
			} else {
				if toolByPath[p] != nil && toolByPath[p].Name() == pluginName {
					pathsForPlugin = append(pathsForPlugin, p)
				}
			}
		}

		if len(pathsForPlugin) == 0 {
			continue
		}

		toolMetrics := make(map[string][]Violation)
		var mu sync.Mutex
		var eg errgroup.Group

		for i := 0; i < len(pathsForPlugin); i += 300 {
			start := i
			end := i + 300
			if end > len(pathsForPlugin) {
				end = len(pathsForPlugin)
			}
			chunkPaths := pathsForPlugin[start:end]

			eg.Go(func() error {
				chunkMetrics, err := plugin.Analyze(chunkPaths)
				if err != nil {
					return err
				}

				mu.Lock()
				defer mu.Unlock()
				for k, v := range chunkMetrics {
					toolMetrics[k] = v
				}
				return nil
			})
		}

		analyzeErr := eg.Wait()

		if analyzeErr != nil {
			batchErrorMsg = fmt.Sprintf("Orchestration error (%s): %v", pluginName, analyzeErr)
			continue
		}

		if toolMetrics != nil {
			var nextRemaining []string
			for _, p := range pathsRemaining {
				absP, _ := filepath.Abs(p)
				found := false
				for outPath, outMetrics := range toolMetrics {
					outAbs, _ := filepath.Abs(outPath)
					if outAbs == absP || outPath == p || filepath.Clean(outAbs) == filepath.Clean(p) {
						metrics := metricsMap[p]
						for _, v := range outMetrics {
							switch v.RuleName {
							case "Complexity":
								if v.Value > metrics.Complexity {
									metrics.Complexity = v.Value
								}
							case "FunctionLength":
								if v.Value > metrics.FunctionLength {
									metrics.FunctionLength = v.Value
								}
							case "ArgumentCount":
								if v.Value > metrics.ArgumentCount {
									metrics.ArgumentCount = v.Value
								}
							}
						}
						metricsMap[p] = metrics
						found = true
						break
					}
				}
				if !found {
					isForThisPlugin := false
					for _, pfp := range pathsForPlugin {
						if p == pfp {
							isForThisPlugin = true
							break
						}
					}
					if !isForThisPlugin {
						nextRemaining = append(nextRemaining, p)
					}
				}
			}
			pathsRemaining = nextRemaining
		}
	}

	var results []OrchestratorResult
	for _, cleanPath := range validPaths {
		orig := originalPaths[cleanPath]
		res := OrchestratorResult{
			FilePath: orig,
			Language: lang,
		}

		// Tooling is detected if there is a config anchor OR if any native plugin exists for this language
		hasNativePlugin := false
		for _, pl := range plugins {
			if strings.HasSuffix(pl.Name(), "-ast") {
				hasNativePlugin = true
				break
			}
		}

		if configAnchors[cleanPath] != "" || hasNativePlugin {
			res.ToolingDetected = true
		} else {
			res.ToolingDetected = false
			res.Message = fmt.Sprintf("[WARNING] RUNNING BLIND (Level 0) on '%s'. No local %s static analysis config detected. Run 'bootstrap' command to fix.", filepath.Base(cleanPath), strings.ToUpper(lang))
			fmt.Fprintln(os.Stderr, res.Message)
		}

		if anchor := configAnchors[cleanPath]; anchor != "" {
			res.Exceptions = detectRelaxedLimits(anchor, toolByPath[cleanPath])
		}

		if batchErrorMsg != "" && !res.ToolingDetected { // if failed and blind
			res.Message = batchErrorMsg
		} else if batchErrorMsg != "" && res.ToolingDetected && metricsMap[cleanPath] == (MaintainabilityMetrics{}) {
			res.Message = batchErrorMsg
		} else {
			if m, ok := metricsMap[cleanPath]; ok {
				res.Metrics = m
			}
		}
		results = append(results, res)
	}

	return results, nil
}

// ScanDeltaBatch scans a batch of files for a specific language concurrently and returns raw violations.
func ScanDeltaBatch(filePaths []string, originalPaths map[string]string, lang string) (map[string][]Violation, error) {
	if len(filePaths) == 0 {
		return nil, nil
	}

	validPaths := make([]string, 0, len(filePaths))
	for _, p := range filePaths {
		clean, err := sanitizePath(p)
		if err != nil {
			return nil, err
		}
		validPaths = append(validPaths, clean)
	}

	plugins := GlobalRegistry.GetPlugins(lang)
	if len(plugins) == 0 {
		return nil, nil
	}

	toolByPath := make(map[string]ConfigParser)
	for _, cleanPath := range validPaths {
		anchor, parser := detectConfigAndParser(cleanPath, lang)
		if anchor != "" {
			toolByPath[cleanPath] = parser
		}
	}

	metricsMap := make(map[string][]Violation)
	pathsRemaining := validPaths

	for _, plugin := range plugins {
		if len(pathsRemaining) == 0 {
			break
		}

		pluginName := plugin.Name()
		isNative := strings.HasSuffix(pluginName, "-ast")

		var pathsForPlugin []string
		for _, p := range pathsRemaining {
			if isNative {
				pathsForPlugin = append(pathsForPlugin, p)
			} else {
				if toolByPath[p] != nil && toolByPath[p].Name() == pluginName {
					pathsForPlugin = append(pathsForPlugin, p)
				}
			}
		}

		if len(pathsForPlugin) == 0 {
			continue
		}

		toolMetrics := make(map[string][]Violation)
		var mu sync.Mutex
		var eg errgroup.Group

		for i := 0; i < len(pathsForPlugin); i += 300 {
			start := i
			end := i + 300
			if end > len(pathsForPlugin) {
				end = len(pathsForPlugin)
			}
			chunkPaths := pathsForPlugin[start:end]

			eg.Go(func() error {
				chunkMetrics, err := plugin.Analyze(chunkPaths)
				if err != nil {
					return err
				}

				mu.Lock()
				defer mu.Unlock()
				for k, v := range chunkMetrics {
					toolMetrics[k] = v
				}
				return nil
			})
		}

		if err := eg.Wait(); err != nil {
			return nil, fmt.Errorf("Orchestration error (%s): %w", pluginName, err)
		}

		if toolMetrics != nil {
			var nextRemaining []string
			for _, p := range pathsRemaining {
				absP, _ := filepath.Abs(p)
				found := false
				for outPath, outMetrics := range toolMetrics {
					outAbs, _ := filepath.Abs(outPath)
					if outAbs == absP || outPath == p || filepath.Clean(outAbs) == filepath.Clean(p) {
						origPath := originalPaths[p]
						if origPath == "" {
							origPath = p
						}
						metricsMap[origPath] = append(metricsMap[origPath], outMetrics...)
						found = true
						break
					}
				}
				if !found {
					isForThisPlugin := false
					for _, pfp := range pathsForPlugin {
						if p == pfp {
							isForThisPlugin = true
							break
						}
					}
					if !isForThisPlugin {
						nextRemaining = append(nextRemaining, p)
					}
				}
			}
			pathsRemaining = nextRemaining
		}
	}

	return metricsMap, nil
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

func getParsersForLang(lang string) []ConfigParser {
	switch lang {
	case "typescript", "javascript":
		return []ConfigParser{BiomeConfigParser{}, ESLintConfigParser{}}
	case "python":
		return []ConfigParser{RuffConfigParser{}, PyLintConfigParser{}}
	case "go":
		return []ConfigParser{GoConfigParser{}}
	case "ruby":
		return []ConfigParser{StandardRBConfigParser{}, RuboCopConfigParser{}}
	}
	return nil
}

func detectConfigAndParser(filePath string, lang string) (string, ConfigParser) {
	parsers := getParsersForLang(lang)
	if len(parsers) == 0 {
		return "", nil
	}

	dir := filepath.Dir(filePath)
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}

	for {
		for _, parser := range parsers {
			anchors := parser.Anchors()
			for _, anchor := range anchors {
				p := filepath.Join(absDir, anchor)
				if _, err := os.Stat(p); err == nil {
					return p, parser
				}
			}
		}
		parent := filepath.Dir(absDir)
		if parent == absDir {
			break
		}
		absDir = parent
	}
	return "", nil
}

func detectConfig(filePath string, lang string) string {
	anchor, _ := detectConfigAndParser(filePath, lang)
	return anchor
}

func runLintCommandJSON(name string, target interface{}, args ...string) (int, []byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return 0, nil, fmt.Errorf("%s not found in PATH", name)
		}
		return 0, nil, fmt.Errorf("failed to start %s: %w", name, err)
	}

	decodeErr := json.NewDecoder(stdout).Decode(target)

	err = cmd.Wait()

	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		return 0, stderrBuf.Bytes(), fmt.Errorf("failed to run %s: %w", name, err)
	}

	if decodeErr != nil && decodeErr != io.EOF {
		return exitCode, stderrBuf.Bytes(), fmt.Errorf("failed to decode JSON: %w", decodeErr)
	}

	return exitCode, stderrBuf.Bytes(), nil
}

func updateMetric(metric *int, valStr string) {
	var val int
	fmt.Sscanf(valStr, "%d", &val)
	if val > *metric {
		*metric = val
	}
}

type ESLintPlugin struct{}

func (p ESLintPlugin) Name() string {
	return "eslint"
}

func (p ESLintPlugin) Analyze(filePaths []string) (map[string][]Violation, error) {
    args := []string{"--no-install", "eslint", "-f", "json"}
    args = append(args, "--")
    args = append(args, filePaths...)
	var list []struct {
		FilePath string `json:"filePath"`
		Messages []struct {
			RuleID   string `json:"ruleId"`
			Message  string `json:"message"`
			Line     int    `json:"line"`
			Severity int    `json:"severity"`
		} `json:"messages"`
	}
	exitCode, output, err := runLintCommandJSON("npx", &list, args...)
	if err != nil {
		if exitCode > 0 {
			return nil, fmt.Errorf("ESLint crashed or encountered a configuration error (exit code %d): %v\n%s", exitCode, err, string(output))
		}
		return nil, fmt.Errorf("ESLint error: %w", err)
	}

	if exitCode == 0 || exitCode == 1 {
		metricsMap := make(map[string][]Violation)
		if len(list) > 0 {
			reComplexity := regexp.MustCompile(`complexity of (\d+)`)
			reParameters := regexp.MustCompile(`has (\d+) parameters`)
			reLines := regexp.MustCompile(`exceeds (\d+) lines`)
			reFallback := regexp.MustCompile(`(\d+)`)
			for _, result := range list {
				var violations []Violation
				for _, msg := range result.Messages {
					var val int
					if msg.RuleID == "complexity" {
						if m := reComplexity.FindStringSubmatch(msg.Message); len(m) > 1 {
							fmt.Sscanf(m[1], "%d", &val)
						} else if m := reFallback.FindStringSubmatch(msg.Message); len(m) > 1 {
							fmt.Sscanf(m[1], "%d", &val)
						}
						violations = append(violations, Violation{RuleName: "Complexity", Value: val, StartLine: msg.Line, EndLine: msg.Line, Message: msg.Message})
					} else if msg.RuleID == "max-params" {
						if m := reParameters.FindStringSubmatch(msg.Message); len(m) > 1 {
							fmt.Sscanf(m[1], "%d", &val)
						} else if m := reFallback.FindStringSubmatch(msg.Message); len(m) > 1 {
							fmt.Sscanf(m[1], "%d", &val)
						}
						violations = append(violations, Violation{RuleName: "ArgumentCount", Value: val, StartLine: msg.Line, EndLine: msg.Line, Message: msg.Message})
					} else if msg.RuleID == "max-lines-per-function" {
						if m := reLines.FindStringSubmatch(msg.Message); len(m) > 1 {
							fmt.Sscanf(m[1], "%d", &val)
						} else if m := reFallback.FindStringSubmatch(msg.Message); len(m) > 1 {
							fmt.Sscanf(m[1], "%d", &val)
						}
						violations = append(violations, Violation{RuleName: "FunctionLength", Value: val, StartLine: msg.Line, EndLine: msg.Line, Message: msg.Message})
					}
				}
				metricsMap[result.FilePath] = violations
			}
		}
		return metricsMap, nil
	}

	return nil, fmt.Errorf("ESLint exited with unexpected code %d: %s", exitCode, strings.TrimSpace(string(output)))
}

type PyLintPlugin struct{}

func (p PyLintPlugin) Name() string {
	return "pylint"
}

func (p PyLintPlugin) Analyze(filePaths []string) (map[string][]Violation, error) {
	args := []string{"--output-format=json"}
	args = append(args, "--")
	args = append(args, filePaths...)

	var list []struct {
		Path    string `json:"path"`
		Message string `json:"message"`
		Symbol  string `json:"symbol"`
	}
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
			for _, msg := range list {
				var val int
				var rule string
				if msg.Symbol == "too-many-statements" {
					if strings.Contains(msg.Message, "Too many statements") {
						fmt.Sscanf(msg.Message, "Too many statements (%d/%*d)", &val)
						rule = "FunctionLength"
					}
				} else if msg.Symbol == "too-many-arguments" {
					if strings.Contains(msg.Message, "Too many arguments") {
						fmt.Sscanf(msg.Message, "Too many arguments (%d/%*d)", &val)
						rule = "ArgumentCount"
					}
				} else if msg.Symbol == "too-many-branches" || msg.Symbol == "too-complex" {
					if strings.Contains(msg.Message, "McCabe rating is") {
						fmt.Sscanf(msg.Message, "McCabe rating is %d", &val)
						rule = "Complexity"
					}
				}
				if rule != "" {
					metricsMap[msg.Path] = append(metricsMap[msg.Path], Violation{RuleName: rule, Value: val, Message: msg.Message})
				}
			}
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

type RuboCopPlugin struct{}

func (p RuboCopPlugin) Name() string {
	return "rubocop"
}

func (p RuboCopPlugin) Analyze(filePaths []string) (map[string][]Violation, error) {
	args := []string{"--format", "json"}
	args = append(args, "--")
	args = append(args, filePaths...)

	var result struct {
		Files []struct {
			Path     string `json:"path"`
			Offenses []struct {
				CopName string `json:"cop_name"`
				Message string `json:"message"`
			} `json:"offenses"`
		} `json:"files"`
	}
	exitCode, output, err := runLintCommandJSON("rubocop", &result, args...)
	if err != nil {
		if exitCode > 0 {
			return nil, fmt.Errorf("rubocop crashed or encountered a configuration error (exit code %d): %v\n%s", exitCode, err, string(output))
		}
		return nil, fmt.Errorf("rubocop error: %w", err)
	}

	if exitCode == 0 || exitCode == 1 {
		metricsMap := make(map[string][]Violation)
		if len(result.Files) > 0 {
			reVal := regexp.MustCompile(`\[(\d+)/`)
			for _, file := range result.Files {
				var violations []Violation
				for _, off := range file.Offenses {
					var val int
					if strings.Contains(off.Message, "[") {
						if m := reVal.FindStringSubmatch(off.Message); len(m) > 1 {
							fmt.Sscanf(m[1], "%d", &val)
						}
					}
					if val == 0 {
						continue
					}
					switch off.CopName {
					case "Metrics/CyclomaticComplexity":
						violations = append(violations, Violation{RuleName: "Complexity", Value: val, Message: off.Message})
					case "Metrics/MethodLength":
						violations = append(violations, Violation{RuleName: "FunctionLength", Value: val, Message: off.Message})
					case "Metrics/ParameterLists":
						violations = append(violations, Violation{RuleName: "ArgumentCount", Value: val, Message: off.Message})
					}
				}
				metricsMap[file.Path] = violations
			}
		}
		if exitCode == 1 && len(result.Files) == 0 && len(output) > 0 {
			return nil, fmt.Errorf("rubocop crashed or encountered a configuration error (exit code 1): %s", strings.TrimSpace(string(output)))
		}
		return metricsMap, nil
	}

	return nil, fmt.Errorf("rubocop exited with unexpected code %d: %s", exitCode, strings.TrimSpace(string(output)))
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

func detectRelaxedLimits(configPath string, parser ConfigParser) []RelaxedLimit {
	var exceptions []RelaxedLimit
	if configPath == "" || parser == nil {
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

type RuffPlugin struct{}

func (p RuffPlugin) Name() string {
	return "ruff"
}

func (p RuffPlugin) Analyze(filePaths []string) (map[string][]Violation, error) {
	args := []string{"check", "--output-format=json"}
	args = append(args, "--")
	args = append(args, filePaths...)

	var list []struct {
		Filename string `json:"filename"`
		Message  string `json:"message"`
		Code     string `json:"code"`
	}
	exitCode, output, err := runLintCommandJSON("ruff", &list, args...)
	if err != nil {
		if exitCode > 0 {
			return nil, fmt.Errorf("ruff crashed or encountered a configuration error (exit code %d): %v\n%s", exitCode, err, string(output))
		}
		return nil, fmt.Errorf("ruff error: %w", err)
	}

	if exitCode >= 0 {
		metricsMap := make(map[string][]Violation)
		if len(list) > 0 {
			reVal := regexp.MustCompile(`\((\d+)\s*>`)
			for _, msg := range list {
				var val int

				if strings.Contains(msg.Message, "(") && strings.Contains(msg.Message, ">") {
					if m := reVal.FindStringSubmatch(msg.Message); len(m) > 1 {
						fmt.Sscanf(m[1], "%d", &val)
					}
				}

				var rule string
				if val > 0 {
					if msg.Code == "C901" || strings.HasPrefix(msg.Code, "C90") {
						rule = "Complexity"
					} else if msg.Code == "PLR0915" {
						rule = "FunctionLength"
					} else if msg.Code == "PLR0913" {
						rule = "ArgumentCount"
					}
				} else {
					if msg.Code == "C901" {
						rule = "Complexity"
						val = 1
					}
				}
				if rule != "" {
					metricsMap[msg.Filename] = append(metricsMap[msg.Filename], Violation{RuleName: rule, Value: val, Message: msg.Message})
				}
			}
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

type BiomePlugin struct{}

func (p BiomePlugin) Name() string {
	return "biome"
}

func (p BiomePlugin) Analyze(filePaths []string) (map[string][]Violation, error) {
	args := []string{"lint", "--formatter-enabled=false", "--output-format=json"}
	args = append(args, filePaths...)

	var result struct {
		Diagnostics []struct {
			Category string `json:"category"`
			Location struct {
				Path struct {
					File string `json:"file"`
				} `json:"path"`
			} `json:"location"`
			Description string `json:"description"`
		} `json:"diagnostics"`
	}
	exitCode, output, err := runLintCommandJSON("biome", &result, args...)
	if err != nil {
		if exitCode > 0 {
			return nil, fmt.Errorf("biome crashed or encountered a configuration error (exit code %d): %v\n%s", exitCode, err, string(output))
		}
		return nil, fmt.Errorf("biome error: %w", err)
	}

	if exitCode == 0 || exitCode == 1 {
		metricsMap := make(map[string][]Violation)
		if len(result.Diagnostics) > 0 {
			reVal := regexp.MustCompile(`(\d+)`)
			for _, diag := range result.Diagnostics {
				path := diag.Location.Path.File
				if path == "" {
					continue
				}

				var val int
				if strings.Contains(diag.Category, "complexity") || strings.Contains(diag.Description, "complexity") {
					if m := reVal.FindStringSubmatch(diag.Description); len(m) > 1 {
						fmt.Sscanf(m[1], "%d", &val)
					}
					if val == 0 {
						val = 2
					} // Fallback if parse fails but issue exists
					metricsMap[path] = append(metricsMap[path], Violation{RuleName: "Complexity", Value: val, Message: diag.Description})
				}
				if strings.Contains(diag.Category, "maxParameters") || strings.Contains(diag.Description, "parameters") {
					if m := reVal.FindStringSubmatch(diag.Description); len(m) > 1 {
						fmt.Sscanf(m[1], "%d", &val)
					}
					if val == 0 {
						val = 2
					} // Fallback
					metricsMap[path] = append(metricsMap[path], Violation{RuleName: "ArgumentCount", Value: val, Message: diag.Description})
				}
			}
		}
		if exitCode == 1 && len(metricsMap) == 0 && len(output) > 0 {
			return nil, fmt.Errorf("biome crashed or encountered a configuration error (exit code 1): %s", strings.TrimSpace(string(output)))
		}
		return metricsMap, nil
	}

	return nil, fmt.Errorf("biome exited with unexpected code %d: %s", exitCode, strings.TrimSpace(string(output)))
}

type StandardRBPlugin struct{}

func (p StandardRBPlugin) Name() string {
	return "standardrb"
}

func (p StandardRBPlugin) Analyze(filePaths []string) (map[string][]Violation, error) {
	args := []string{"--format", "json"}
	args = append(args, "--")
	args = append(args, filePaths...)

	var result struct {
		Files []struct {
			Path     string `json:"path"`
			Offenses []struct {
				CopName string `json:"cop_name"`
				Message string `json:"message"`
			} `json:"offenses"`
		} `json:"files"`
	}
	exitCode, output, err := runLintCommandJSON("standardrb", &result, args...)
	if err != nil {
		if exitCode > 0 {
			return nil, fmt.Errorf("standardrb crashed or encountered a configuration error (exit code %d): %v\n%s", exitCode, err, string(output))
		}
		return nil, fmt.Errorf("standardrb error: %w", err)
	}

	if exitCode == 0 || exitCode == 1 {
		metricsMap := make(map[string][]Violation)
		if len(result.Files) > 0 {
			for _, file := range result.Files {
				var violations []Violation
				for _, off := range file.Offenses {
					var val int
					if strings.Contains(off.Message, "[") {
						fmt.Sscanf(off.Message, "%*[^[][%d/%*d]", &val)
					}
					if val == 0 {
						continue
					}

					switch off.CopName {
					case "Metrics/CyclomaticComplexity":
						violations = append(violations, Violation{RuleName: "Complexity", Value: val, Message: off.Message})
					case "Metrics/MethodLength":
						violations = append(violations, Violation{RuleName: "FunctionLength", Value: val, Message: off.Message})
					case "Metrics/ParameterLists":
						violations = append(violations, Violation{RuleName: "ArgumentCount", Value: val, Message: off.Message})
					}
				}
				metricsMap[file.Path] = violations
			}
		}
		if exitCode == 1 && len(result.Files) == 0 && len(output) > 0 {
			return nil, fmt.Errorf("standardrb crashed or encountered a configuration error (exit code 1): %s", strings.TrimSpace(string(output)))
		}
		return metricsMap, nil
	}

	return nil, fmt.Errorf("standardrb exited with unexpected code %d: %s", exitCode, strings.TrimSpace(string(output)))
}
