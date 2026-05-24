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
	Complexity          int `json:"complexity"`
	CognitiveComplexity int `json:"cognitive_complexity"`
	FunctionLength      int `json:"function_length"`
	ArgumentCount       int `json:"argument_count"`
	MaxCaseLength       int `json:"max_case_length"`
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

func processPluginsMetrics(plugins []Plugin, toolByPath map[string]ConfigParser, validPaths []string, metricsMap map[string]MaintainabilityMetrics) (string, []string) {
	var batchErrorMsg string
	pathsRemaining := validPaths

	for _, plugin := range plugins {
		if len(pathsRemaining) == 0 {
			break
		}

		pathsForPlugin := filterPathsForPlugin(pathsRemaining, plugin, toolByPath)
		if len(pathsForPlugin) == 0 {
			continue
		}

		toolMetrics, analyzeErr := analyzeInChunks(plugin, pathsForPlugin)
		if analyzeErr != nil {
			batchErrorMsg = fmt.Sprintf("Orchestration error (%s): %v", plugin.Name(), analyzeErr)
			continue
		}

		if toolMetrics != nil {
			pathsRemaining = updateMetricsMap(UpdateMetricsCtx{
				MetricsMap:     metricsMap,
				ToolMetrics:    toolMetrics,
				PathsRemaining: pathsRemaining,
				PathsForPlugin: pathsForPlugin,
			})
		}
	}
	return batchErrorMsg, pathsRemaining
}

// OrchestratedScanBatch scans a batch of files for a specific language concurrently.
func OrchestratedScanBatch(filePaths []string, lang string) ([]OrchestratorResult, error) {
	if len(filePaths) == 0 {
		return nil, nil
	}

	validPaths, originalPaths, err := sanitizeAndMapPaths(filePaths)
	if err != nil {
		return nil, err
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

	configAnchors, toolByPath := findConfigAndParsers(validPaths, lang)

	metricsMap := make(map[string]MaintainabilityMetrics)
	batchErrorMsg, _ := processPluginsMetrics(plugins, toolByPath, validPaths, metricsMap)

	return buildOrchestratorResults(BatchContext{
		ValidPaths:    validPaths,
		OriginalPaths: originalPaths,
		Lang:          lang,
		Plugins:       plugins,
		ConfigAnchors: configAnchors,
		ToolByPath:    toolByPath,
		MetricsMap:    metricsMap,
		BatchErrorMsg: batchErrorMsg,
	}), nil
}

type ProcessDeltaCtx struct {
	Plugins       []Plugin
	ToolByPath    map[string]ConfigParser
	ValidPaths    []string
	OriginalPaths map[string]string
	MetricsMap    map[string][]Violation
}

func processPluginsDelta(ctx ProcessDeltaCtx) ([]string, error) {
	pathsRemaining := ctx.ValidPaths

	for _, plugin := range ctx.Plugins {
		if len(pathsRemaining) == 0 {
			break
		}

		pathsForPlugin := filterPathsForPlugin(pathsRemaining, plugin, ctx.ToolByPath)
		if len(pathsForPlugin) == 0 {
			continue
		}

		toolMetrics, err := analyzeInChunks(plugin, pathsForPlugin)
		if err != nil {
			return nil, fmt.Errorf("Orchestration error (%s): %w", plugin.Name(), err)
		}

		if toolMetrics != nil {
			pathsRemaining = updateDeltaMetricsMap(UpdateDeltaCtx{
				MetricsMap:     ctx.MetricsMap,
				ToolMetrics:    toolMetrics,
				OriginalPaths:  ctx.OriginalPaths,
				PathsRemaining: pathsRemaining,
				PathsForPlugin: pathsForPlugin,
			})
		}
	}
	return pathsRemaining, nil
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

	_, toolByPath := findConfigAndParsers(validPaths, lang)

	metricsMap := make(map[string][]Violation)
	_, err := processPluginsDelta(ProcessDeltaCtx{
		Plugins:       plugins,
		ToolByPath:    toolByPath,
		ValidPaths:    validPaths,
		OriginalPaths: originalPaths,
		MetricsMap:    metricsMap,
	})
	if err != nil {
		return nil, err
	}

	return metricsMap, nil
}


func sanitizeAndMapPaths(filePaths []string) ([]string, map[string]string, error) {
	validPaths := make([]string, 0, len(filePaths))
	originalPaths := make(map[string]string)

	for _, p := range filePaths {
		clean, err := sanitizePath(p)
		if err != nil {
			return nil, nil, err
		}
		abs, err := filepath.Abs(clean)
		if err == nil {
			originalPaths[abs] = p
		}
		originalPaths[clean] = p
		validPaths = append(validPaths, clean)
	}
	return validPaths, originalPaths, nil
}

func filterPathsForPlugin(pathsRemaining []string, plugin Plugin, toolByPath map[string]ConfigParser) []string {
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
	return pathsForPlugin
}

func analyzeInChunks(plugin Plugin, pathsForPlugin []string) (map[string][]Violation, error) {
	toolMetrics := make(map[string][]Violation)
	var mu sync.Mutex
	eg, _ := errgroup.WithContext(context.Background())

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
		return nil, err
	}
	return toolMetrics, nil
}

type UpdateMetricsCtx struct {
	MetricsMap     map[string]MaintainabilityMetrics
	ToolMetrics    map[string][]Violation
	PathsRemaining []string
	PathsForPlugin []string
}

func pathsMatch(p, absP, outPath string) bool {
	outAbs, _ := filepath.Abs(outPath)
	return outAbs == absP || outPath == p || filepath.Clean(outAbs) == filepath.Clean(p)
}

func isPathForPlugin(p string, pathsForPlugin []string) bool {
	for _, pfp := range pathsForPlugin {
		if p == pfp {
			return true
		}
	}
	return false
}

func updateSingleMetric(v Violation, metrics *MaintainabilityMetrics) {
	switch v.RuleName {
	case "Complexity":
		if v.Value > metrics.Complexity {
			metrics.Complexity = v.Value
		}
	case "CognitiveComplexity":
		if v.Value > metrics.CognitiveComplexity {
			metrics.CognitiveComplexity = v.Value
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

func updateMetricsForPath(p string, outMetrics []Violation, metricsMap map[string]MaintainabilityMetrics) {
	metrics := metricsMap[p]
	for _, v := range outMetrics {
		updateSingleMetric(v, &metrics)
	}
	metricsMap[p] = metrics
}

func findAndUpdateMetrics(p string, absP string, ctx UpdateMetricsCtx) bool {
	for outPath, outMetrics := range ctx.ToolMetrics {
		if pathsMatch(p, absP, outPath) {
			updateMetricsForPath(p, outMetrics, ctx.MetricsMap)
			return true
		}
	}
	return false
}

func updateMetricsMap(ctx UpdateMetricsCtx) []string {
	var nextRemaining []string
	for _, p := range ctx.PathsRemaining {
		absP, _ := filepath.Abs(p)
		found := findAndUpdateMetrics(p, absP, ctx)
		if !found {
			if !isPathForPlugin(p, ctx.PathsForPlugin) {
				nextRemaining = append(nextRemaining, p)
			}
		}
	}
	return nextRemaining
}

type UpdateDeltaCtx struct {
	MetricsMap     map[string][]Violation
	ToolMetrics    map[string][]Violation
	OriginalPaths  map[string]string
	PathsRemaining []string
	PathsForPlugin []string
}

func findAndUpdateDeltaMetrics(p string, absP string, ctx UpdateDeltaCtx) bool {
	for outPath, outMetrics := range ctx.ToolMetrics {
		if pathsMatch(p, absP, outPath) {
			origPath := ctx.OriginalPaths[p]
			if origPath == "" {
				origPath = p
			}
			ctx.MetricsMap[origPath] = append(ctx.MetricsMap[origPath], outMetrics...)
			return true
		}
	}
	return false
}

func updateDeltaMetricsMap(ctx UpdateDeltaCtx) []string {
	var nextRemaining []string
	for _, p := range ctx.PathsRemaining {
		absP, _ := filepath.Abs(p)
		found := findAndUpdateDeltaMetrics(p, absP, ctx)
		if !found {
			if !isPathForPlugin(p, ctx.PathsForPlugin) {
				nextRemaining = append(nextRemaining, p)
			}
		}
	}
	return nextRemaining
}

type BatchContext struct {
	ValidPaths    []string
	OriginalPaths map[string]string
	Lang          string
	Plugins       []Plugin
	ConfigAnchors map[string]string
	ToolByPath    map[string]ConfigParser
	MetricsMap    map[string]MaintainabilityMetrics
	BatchErrorMsg string
}

func hasNativePlugin(plugins []Plugin) bool {
	for _, pl := range plugins {
		if strings.HasSuffix(pl.Name(), "-ast") {
			return true
		}
	}
	return false
}

func populateResultMessage(ctx BatchContext, res *OrchestratorResult, cleanPath string) {
	if ctx.BatchErrorMsg != "" && !res.ToolingDetected {
		res.Message = ctx.BatchErrorMsg
	} else if ctx.BatchErrorMsg != "" && res.ToolingDetected && ctx.MetricsMap[cleanPath] == (MaintainabilityMetrics{}) {
		res.Message = ctx.BatchErrorMsg
	} else {
		if m, ok := ctx.MetricsMap[cleanPath]; ok {
			res.Metrics = m
		}
	}
}

func buildSingleResult(ctx BatchContext, cleanPath string) OrchestratorResult {
	orig := ctx.OriginalPaths[cleanPath]
	res := OrchestratorResult{
		FilePath: orig,
		Language: ctx.Lang,
	}

	if ctx.ConfigAnchors[cleanPath] != "" || hasNativePlugin(ctx.Plugins) {
		res.ToolingDetected = true
	} else {
		res.ToolingDetected = false
		res.Message = fmt.Sprintf("[WARNING] RUNNING BLIND (Level 0) on '%s'. No local %s static analysis config detected. Run 'bootstrap' command to fix.", filepath.Base(cleanPath), strings.ToUpper(ctx.Lang))
		fmt.Fprintln(os.Stderr, res.Message)
	}

	if anchor := ctx.ConfigAnchors[cleanPath]; anchor != "" {
		res.Exceptions = DetectRelaxedLimits(anchor, ctx.ToolByPath[cleanPath])
	}

	populateResultMessage(ctx, &res, cleanPath)
	return res
}

func buildOrchestratorResults(ctx BatchContext) []OrchestratorResult {
	var results []OrchestratorResult
	for _, cleanPath := range ctx.ValidPaths {
		results = append(results, buildSingleResult(ctx, cleanPath))
	}
	return results
}

func findConfigAndParsers(validPaths []string, lang string) (map[string]string, map[string]ConfigParser) {
	configAnchors := make(map[string]string)
	toolByPath := make(map[string]ConfigParser)

	for _, cleanPath := range validPaths {
		anchor, parser := DetectConfigAndParser(cleanPath, lang)
		if anchor != "" {
			configAnchors[cleanPath] = anchor
			toolByPath[cleanPath] = parser
		}
	}
	return configAnchors, toolByPath
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

func findConfigInDir(absDir string, parsers []ConfigParser) (string, ConfigParser) {
	for _, parser := range parsers {
		anchors := parser.Anchors()
		for _, anchor := range anchors {
			p := filepath.Join(absDir, anchor)
			if _, err := os.Stat(p); err == nil {
				return p, parser
			}
		}
	}
	return "", nil
}

func DetectConfigAndParser(filePath string, lang string) (string, ConfigParser) {
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
		if p, parser := findConfigInDir(absDir, parsers); p != "" {
			return p, parser
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
	anchor, _ := DetectConfigAndParser(filePath, lang)
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

type ESLintMessage struct {
	RuleID   string `json:"ruleId"`
	Message  string `json:"message"`
	Line     int    `json:"line"`
	EndLine  int    `json:"endLine"`
	Severity int    `json:"severity"`
}

type ESLintResult struct {
	FilePath string          `json:"filePath"`
	Messages []ESLintMessage `json:"messages"`
}

func extractESLintValue(msg string, primaryRe *regexp.Regexp, fallbackRe *regexp.Regexp) int {
	var val int
	if m := primaryRe.FindStringSubmatch(msg); len(m) > 1 {
		fmt.Sscanf(m[1], "%d", &val)
	} else if m := fallbackRe.FindStringSubmatch(msg); len(m) > 1 {
		fmt.Sscanf(m[1], "%d", &val)
	}
	return val
}

//nolint:gocognit,cyclop // Highly cohesive mapping logic for linters. Splitting this hurts readability.
func parseESLintMessages(messages []ESLintMessage) []Violation {
	var violations []Violation
	reComplexity := regexp.MustCompile(`complexity of (\d+)`)
	reParameters := regexp.MustCompile(`has (\d+) parameters`)
	reLines := regexp.MustCompile(`exceeds (\d+) lines`)
	reFallback := regexp.MustCompile(`(\d+)`)

	for _, msg := range messages {
		endLine := msg.EndLine
		if endLine == 0 {
			endLine = msg.Line + 100
		}
		if msg.RuleID == "complexity" {
			val := extractESLintValue(msg.Message, reComplexity, reFallback)
			violations = append(violations, Violation{RuleName: "Complexity", Value: val, StartLine: msg.Line, EndLine: endLine, Message: msg.Message})
		} else if msg.RuleID == "max-params" {
			val := extractESLintValue(msg.Message, reParameters, reFallback)
			violations = append(violations, Violation{RuleName: "ArgumentCount", Value: val, StartLine: msg.Line, EndLine: endLine, Message: msg.Message})
		} else if msg.RuleID == "max-lines-per-function" {
			val := extractESLintValue(msg.Message, reLines, reFallback)
			violations = append(violations, Violation{RuleName: "FunctionLength", Value: val, StartLine: msg.Line, EndLine: endLine, Message: msg.Message})
		}
	}
	return violations
}

func processESLintAnalyzeResult(exitCode int, list []ESLintResult, output []byte) (map[string][]Violation, error) {
	if exitCode == 0 || exitCode == 1 || exitCode == 2 {
		metricsMap := make(map[string][]Violation)
		if len(list) > 0 {
			for _, result := range list {
				violations := parseESLintMessages(result.Messages)
				if len(violations) > 0 {
					metricsMap[result.FilePath] = violations
				}
			}
		}
		return metricsMap, nil
	}

	return nil, fmt.Errorf("ESLint exited with unexpected code %d: %s", exitCode, strings.TrimSpace(string(output)))
}

func (p ESLintPlugin) Analyze(filePaths []string) (map[string][]Violation, error) {
    args := []string{"--no-install", "eslint", "-f", "json"}
    args = append(args, "--")
    args = append(args, filePaths...)
	var list []ESLintResult
	exitCode, output, err := runLintCommandJSON("npx", &list, args...)
	if err != nil {
		if exitCode > 0 {
			return nil, fmt.Errorf("ESLint crashed or encountered a configuration error (exit code %d): %v\n%s", exitCode, err, string(output))
		}
		return nil, fmt.Errorf("ESLint error: %w", err)
	}

	return processESLintAnalyzeResult(exitCode, list, output)
}

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
			endLine := msg.EndLine
			if endLine == 0 {
				endLine = msg.Line + 100
			}
			metricsMap[msg.Path] = append(metricsMap[msg.Path], Violation{RuleName: rule, Value: val, StartLine: msg.Line, EndLine: endLine, Message: msg.Message})
		}
	}
	return metricsMap
}

func (p PyLintPlugin) Analyze(filePaths []string) (map[string][]Violation, error) {
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

type RuboCopPlugin struct{}

func (p RuboCopPlugin) Name() string {
	return "rubocop"
}

type RuboCopLocation struct {
	Line int `json:"line"`
	LastLine int `json:"last_line"`
}

type RuboCopOffense struct {
	CopName string `json:"cop_name"`
	Message string `json:"message"`
	Location RuboCopLocation `json:"location"`
}

type RuboCopFile struct {
	Path     string `json:"path"`
	Offenses []RuboCopOffense `json:"offenses"`
}

type RuboCopResult struct {
	Files []RuboCopFile `json:"files"`
}

//nolint:gocognit,cyclop // Highly cohesive mapping logic, splitting hurts readability
func parseSingleRuboCopOffense(off RuboCopOffense, reVal *regexp.Regexp, fileViolations *[]Violation) {
	var val int
	if strings.Contains(off.Message, "[") {
		if m := reVal.FindStringSubmatch(off.Message); len(m) > 1 {
			fmt.Sscanf(m[1], "%d", &val)
		}
	}
	if val == 0 {
		return
	}

	endLine := off.Location.LastLine
	if endLine == 0 {
		endLine = off.Location.Line + 100
	}

	switch off.CopName {
	case "Metrics/CyclomaticComplexity":
		*fileViolations = append(*fileViolations, Violation{RuleName: "Complexity", Value: val, StartLine: off.Location.Line, EndLine: endLine, Message: off.Message})
	case "Metrics/MethodLength":
		*fileViolations = append(*fileViolations, Violation{RuleName: "FunctionLength", Value: val, StartLine: off.Location.Line, EndLine: endLine, Message: off.Message})
	case "Metrics/ParameterLists":
		*fileViolations = append(*fileViolations, Violation{RuleName: "ArgumentCount", Value: val, StartLine: off.Location.Line, EndLine: endLine, Message: off.Message})
	}
}

//nolint:gocognit,cyclop // Highly cohesive mapping logic, splitting hurts readability
func parseRuboCopMessages(files []RuboCopFile) map[string][]Violation {
	metricsMap := make(map[string][]Violation)
	reVal := regexp.MustCompile(`\[(\d+)/`)
	for _, file := range files {
		var violations []Violation
		for _, off := range file.Offenses {
			parseSingleRuboCopOffense(off, reVal, &violations)
		}
		if len(violations) > 0 {
			metricsMap[file.Path] = violations
		}
	}
	return metricsMap
}

func processRuboCopAnalyzeResult(exitCode int, result RuboCopResult, output []byte) (map[string][]Violation, error) {
	if exitCode == 0 || exitCode == 1 {
		metricsMap := make(map[string][]Violation)
		if len(result.Files) > 0 {
			metricsMap = parseRuboCopMessages(result.Files)
		}
		if exitCode == 1 && len(result.Files) == 0 && len(output) > 0 {
			return nil, fmt.Errorf("rubocop crashed or encountered a configuration error (exit code 1): %s", strings.TrimSpace(string(output)))
		}
		return metricsMap, nil
	}

	return nil, fmt.Errorf("rubocop exited with unexpected code %d: %s", exitCode, strings.TrimSpace(string(output)))
}

func (p RuboCopPlugin) Analyze(filePaths []string) (map[string][]Violation, error) {
	args := []string{"--format", "json"}
	args = append(args, "--")
	args = append(args, filePaths...)

	var result RuboCopResult
	exitCode, output, err := runLintCommandJSON("rubocop", &result, args...)
	if err != nil {
		if exitCode > 0 {
			return nil, fmt.Errorf("rubocop crashed or encountered a configuration error (exit code %d): %v\n%s", exitCode, err, string(output))
		}
		return nil, fmt.Errorf("rubocop error: %w", err)
	}

	return processRuboCopAnalyzeResult(exitCode, result, output)
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

func isValidConfigFile(configPath string) bool {
	info, err := os.Stat(configPath)
	return err == nil && info.Mode().IsRegular() && info.Size() <= 2*1024*1024
}

func DetectRelaxedLimits(configPath string, parser ConfigParser) []RelaxedLimit {
	var exceptions []RelaxedLimit
	if configPath == "" || parser == nil || !isValidConfigFile(configPath) {
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

func checkLintExecutionError(name string, exitCode int, output []byte, err error) error {
	if err != nil {
		if exitCode > 0 {
			return fmt.Errorf("%s crashed or encountered a configuration error (exit code %d): %v\n%s", name, exitCode, err, string(output))
		}
		return fmt.Errorf("%s error: %w", name, err)
	}
	return nil
}

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
		return "Complexity"
	}
	if *val > 0 {
		if msg.Code == "PLR0915" {
			return "FunctionLength"
		} else if msg.Code == "PLR0913" {
			return "ArgumentCount"
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
			endLine = msg.Location.Row + 100
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

func (p RuffPlugin) Analyze(filePaths []string) (map[string][]Violation, error) {
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

type BiomePlugin struct{}

func (p BiomePlugin) Name() string {
	return "biome"
}

type BiomeLocation struct {
	Path struct {
		File string `json:"file"`
	} `json:"path"`
	Span struct {
		Start int `json:"start"`
		End int `json:"end"`
	} `json:"span"`
}

type BiomeDiagnostic struct {
	Category string `json:"category"`
	Location BiomeLocation `json:"location"`
	Description string `json:"description"`
}

type BiomeResult struct {
	Diagnostics []BiomeDiagnostic `json:"diagnostics"`
}

func parseSingleStandardRBOffense(off StandardRBOffense, fileViolations *[]Violation) {
	var val int
	if strings.Contains(off.Message, "[") {
		fmt.Sscanf(off.Message, "%*[^[][%d/%*d]", &val)
	}
	if val == 0 {
		return
	}

	endLine := off.Location.LastLine
	if endLine == 0 {
		endLine = off.Location.Line + 100
	}

	switch off.CopName {
	case "Metrics/CyclomaticComplexity":
		*fileViolations = append(*fileViolations, Violation{RuleName: "Complexity", Value: val, StartLine: off.Location.Line, EndLine: endLine, Message: off.Message})
	case "Metrics/MethodLength":
		*fileViolations = append(*fileViolations, Violation{RuleName: "FunctionLength", Value: val, StartLine: off.Location.Line, EndLine: endLine, Message: off.Message})
	case "Metrics/ParameterLists":
		*fileViolations = append(*fileViolations, Violation{RuleName: "ArgumentCount", Value: val, StartLine: off.Location.Line, EndLine: endLine, Message: off.Message})
	}
}

//nolint:gocognit,cyclop // Highly cohesive mapping logic, splitting hurts readability
func parseStandardRBMessages(files []StandardRBFile) map[string][]Violation {
	metricsMap := make(map[string][]Violation)
	for _, file := range files {
		var violations []Violation
		for _, off := range file.Offenses {
			parseSingleStandardRBOffense(off, &violations)
		}
		if len(violations) > 0 {
			metricsMap[file.Path] = violations
		}
	}
	return metricsMap
}

func extractBiomeComplexity(desc string, reVal *regexp.Regexp) (string, int) {
	var val int
	if m := reVal.FindStringSubmatch(desc); len(m) > 1 {
		fmt.Sscanf(m[1], "%d", &val)
	}
	if val == 0 {
		val = 2
	}
	return "Complexity", val
}

func extractBiomeMaxParameters(desc string, reVal *regexp.Regexp) (string, int) {
	var val int
	if m := reVal.FindStringSubmatch(desc); len(m) > 1 {
		fmt.Sscanf(m[1], "%d", &val)
	}
	if val == 0 {
		val = 2
	}
	return "ArgumentCount", val
}

func extractBiomeRuleAndVal(diag BiomeDiagnostic, reVal *regexp.Regexp) (string, int) {
	isComplexity := strings.Contains(diag.Category, "complexity") || strings.Contains(diag.Description, "complexity")
	if isComplexity {
		return extractBiomeComplexity(diag.Description, reVal)
	}
	isParams := strings.Contains(diag.Category, "maxParameters") || strings.Contains(diag.Description, "parameters")
	if isParams {
		return extractBiomeMaxParameters(diag.Description, reVal)
	}
	return "", 0
}

func parseSingleBiomeDiagnostic(diag BiomeDiagnostic, reVal *regexp.Regexp, fileViolations *[]Violation) {
	rule, val := extractBiomeRuleAndVal(diag, reVal)
	if rule == "" {
		return
	}

	startLine := diag.Location.Span.Start
	endLine := diag.Location.Span.End
	if endLine == 0 {
		endLine = startLine + 100
	}

	*fileViolations = append(*fileViolations, Violation{RuleName: rule, Value: val, StartLine: startLine, EndLine: endLine, Message: diag.Description})
}

//nolint:gocognit,cyclop // Highly cohesive mapping logic, splitting hurts readability
func parseBiomeMessages(diagnostics []BiomeDiagnostic) map[string][]Violation {
	metricsMap := make(map[string][]Violation)
	reVal := regexp.MustCompile(`(\d+)`)
	for _, diag := range diagnostics {
		path := diag.Location.Path.File
		if path == "" {
			continue
		}
		var violations []Violation
		parseSingleBiomeDiagnostic(diag, reVal, &violations)
		if len(violations) > 0 {
			metricsMap[path] = append(metricsMap[path], violations...)
		}
	}
	return metricsMap
}

func processBiomeAnalyzeResult(exitCode int, result BiomeResult, output []byte) (map[string][]Violation, error) {
	if exitCode == 0 || exitCode == 1 {
		metricsMap := make(map[string][]Violation)
		if len(result.Diagnostics) > 0 {
			metricsMap = parseBiomeMessages(result.Diagnostics)
		}
		if exitCode == 1 && len(metricsMap) == 0 && len(output) > 0 {
			return nil, fmt.Errorf("biome crashed or encountered a configuration error (exit code 1): %s", strings.TrimSpace(string(output)))
		}
		return metricsMap, nil
	}

	return nil, fmt.Errorf("biome exited with unexpected code %d: %s", exitCode, strings.TrimSpace(string(output)))
}

func (p BiomePlugin) Analyze(filePaths []string) (map[string][]Violation, error) {
	args := []string{"lint", "--formatter-enabled=false", "--output-format=json"}
	args = append(args, "--")
	args = append(args, filePaths...)

	var result BiomeResult
	exitCode, output, err := runLintCommandJSON("biome", &result, args...)
	if err != nil {
		if exitCode > 0 {
			return nil, fmt.Errorf("biome crashed or encountered a configuration error (exit code %d): %v\n%s", exitCode, err, string(output))
		}
		return nil, fmt.Errorf("biome error: %w", err)
	}

	return processBiomeAnalyzeResult(exitCode, result, output)
}

type StandardRBPlugin struct{}

func (p StandardRBPlugin) Name() string {
	return "standardrb"
}

type StandardRBLocation struct {
	Line int `json:"line"`
	LastLine int `json:"last_line"`
}

type StandardRBOffense struct {
	CopName string `json:"cop_name"`
	Message string `json:"message"`
	Location StandardRBLocation `json:"location"`
}

type StandardRBFile struct {
	Path     string `json:"path"`
	Offenses []StandardRBOffense `json:"offenses"`
}

type StandardRBResult struct {
	Files []StandardRBFile `json:"files"`
}

func processStandardRBAnalyzeResult(exitCode int, result StandardRBResult, output []byte) (map[string][]Violation, error) {
	if exitCode == 0 || exitCode == 1 {
		metricsMap := make(map[string][]Violation)
		if len(result.Files) > 0 {
			metricsMap = parseStandardRBMessages(result.Files)
		}
		if exitCode == 1 && len(result.Files) == 0 && len(output) > 0 {
			return nil, fmt.Errorf("standardrb crashed or encountered a configuration error (exit code 1): %s", strings.TrimSpace(string(output)))
		}
		return metricsMap, nil
	}

	return nil, fmt.Errorf("standardrb exited with unexpected code %d: %s", exitCode, strings.TrimSpace(string(output)))
}

func (p StandardRBPlugin) Analyze(filePaths []string) (map[string][]Violation, error) {
	args := []string{"--format", "json"}
	args = append(args, "--")
	args = append(args, filePaths...)

	var result StandardRBResult
	exitCode, output, err := runLintCommandJSON("standardrb", &result, args...)
	if execErr := checkLintExecutionError("standardrb", exitCode, output, err); execErr != nil {
		return nil, execErr
	}

	return processStandardRBAnalyzeResult(exitCode, result, output)
}
