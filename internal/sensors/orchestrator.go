package sensors

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

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

// OrchestratedScan scans a specific file. It is a convenience wrapper over OrchestratedScanBatch.
func OrchestratedScan(filePath string) (OrchestratorResult, error) {
	lang := DetectLanguage(filePath)
	if lang == "" {
		return OrchestratorResult{FilePath: filePath}, fmt.Errorf("unsupported or unrecognized language file: %s", filePath)
	}
	results, err := OrchestratedScanBatch([]FileContext{{Path: filePath}}, lang)
	if err != nil {
		return OrchestratorResult{FilePath: filePath, Language: lang}, err
	}
	if len(results) > 0 {
		return results[0], nil
	}
	return OrchestratorResult{FilePath: filePath, Language: lang}, nil
}

func processPluginsMetrics(plugins []Plugin, toolByPath map[string]ConfigParser, validFiles []FileContext, metricsMap map[string]MaintainabilityMetrics) (string, []FileContext) {
	var batchErrorMsg string
	pathsRemaining := validFiles

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
func OrchestratedScanBatch(files []FileContext, lang string) ([]OrchestratorResult, error) {
	if len(files) == 0 {
		return nil, nil
	}

	validFiles, originalPaths, err := sanitizeAndMapFileContexts(files)
	if err != nil {
		return nil, err
	}

	validPaths := make([]string, len(validFiles))
	for i, f := range validFiles {
		validPaths[i] = f.Path
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
	batchErrorMsg, _ := processPluginsMetrics(plugins, toolByPath, validFiles, metricsMap)

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
	ValidPaths    []FileContext
	OriginalPaths map[string]string
	MetricsMap    map[string][]Violation
}

func processPluginsDelta(ctx ProcessDeltaCtx) ([]FileContext, error) {
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
func ScanDeltaBatch(files []FileContext, originalPaths map[string]string, lang string) (map[string][]Violation, error) {
	if len(files) == 0 {
		return nil, nil
	}

	validPaths := make([]FileContext, 0, len(files))
	validPathsStr := make([]string, 0, len(files))
	for _, p := range files {
		clean, err := sanitizePath(p.Path)
		if err != nil {
			return nil, err
		}
		validPaths = append(validPaths, FileContext{Path: clean, Content: p.Content})
		validPathsStr = append(validPathsStr, clean)
	}

	plugins := GlobalRegistry.GetPlugins(lang)
	if len(plugins) == 0 {
		return nil, nil
	}

	_, toolByPath := findConfigAndParsers(validPathsStr, lang)

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

func filterPathsForPlugin(pathsRemaining []FileContext, plugin Plugin, toolByPath map[string]ConfigParser) []FileContext {
	pluginName := plugin.Name()
	isNative := strings.HasSuffix(pluginName, "-ast")
	var pathsForPlugin []FileContext
	for _, p := range pathsRemaining {
		if isNative {
			pathsForPlugin = append(pathsForPlugin, p)
		} else {
			if toolByPath[p.Path] != nil && toolByPath[p.Path].Name() == pluginName {
				pathsForPlugin = append(pathsForPlugin, p)
			}
		}
	}
	return pathsForPlugin
}

func analyzeInChunks(plugin Plugin, pathsForPlugin []FileContext) (map[string][]Violation, error) {
	toolMetrics := make(map[string][]Violation)
	var mu sync.Mutex
	eg, _ := errgroup.WithContext(context.Background())

	for i := 0; i < len(pathsForPlugin); i += PluginChunkSize {
		start := i
		end := i + PluginChunkSize
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
	PathsRemaining []FileContext
	PathsForPlugin []FileContext
}

func isPathForPlugin(p string, pathsForPlugin []FileContext) bool {
	for _, pfp := range pathsForPlugin {
		if p == pfp.Path {
			return true
		}
	}
	return false
}

func updateSingleMetric(v Violation, metrics *MaintainabilityMetrics) {
	switch v.RuleName {
	case RuleComplexity:
		if v.Value > metrics.Complexity {
			metrics.Complexity = v.Value
		}
	case RuleCognitiveComplexity:
		if v.Value > metrics.CognitiveComplexity {
			metrics.CognitiveComplexity = v.Value
		}
	case RuleFunctionLength:
		if v.Value > metrics.FunctionLength {
			metrics.FunctionLength = v.Value
		}
	case RuleArgumentCount:
		if v.Value > metrics.ArgumentCount {
			metrics.ArgumentCount = v.Value
		}
	case RuleCaseBlockLength:
		if v.Value > metrics.MaxCaseLength {
			metrics.MaxCaseLength = v.Value
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

func updateMetricsMap(ctx UpdateMetricsCtx) []FileContext {
	var nextRemaining []FileContext
	for _, f := range ctx.PathsRemaining {
		p := f.Path
		absP, _ := filepath.Abs(p)
		found := findAndUpdateMetrics(p, absP, ctx)
		if !found {
			if !isPathForPlugin(p, ctx.PathsForPlugin) {
				nextRemaining = append(nextRemaining, f)
			}
		}
	}
	return nextRemaining
}

type UpdateDeltaCtx struct {
	MetricsMap     map[string][]Violation
	ToolMetrics    map[string][]Violation
	OriginalPaths  map[string]string
	PathsRemaining []FileContext
	PathsForPlugin []FileContext
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

func updateDeltaMetricsMap(ctx UpdateDeltaCtx) []FileContext {
	var nextRemaining []FileContext
	for _, f := range ctx.PathsRemaining {
		p := f.Path
		absP, _ := filepath.Abs(p)
		found := findAndUpdateDeltaMetrics(p, absP, ctx)
		if !found {
			if !isPathForPlugin(p, ctx.PathsForPlugin) {
				nextRemaining = append(nextRemaining, f)
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

