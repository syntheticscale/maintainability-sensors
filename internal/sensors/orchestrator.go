package sensors

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
)

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

func processSinglePluginMetrics(plugin Plugin, toolByPath map[string]ConfigParser, pathsRemaining []FileContext, metricsMap map[string]MaintainabilityMetrics) ([]FileContext, string) {
	pathsForPlugin := filterPathsForPlugin(pathsRemaining, plugin, toolByPath)
	if len(pathsForPlugin) == 0 {
		return pathsRemaining, ""
	}

	toolMetrics, analyzeErr := analyzeInChunks(plugin, pathsForPlugin)
	if analyzeErr != nil {
		return pathsRemaining, fmt.Sprintf("Orchestration error (%s): %v", plugin.Name(), analyzeErr)
	}

	if toolMetrics != nil {
		return updateMetricsMap(UpdateMetricsCtx{
			MetricsMap:     metricsMap,
			ToolMetrics:    toolMetrics,
			PathsRemaining: pathsRemaining,
			PathsForPlugin: pathsForPlugin,
		}), ""
	}

	return pathsRemaining, ""
}

func processPluginsMetrics(plugins []Plugin, toolByPath map[string]ConfigParser, validFiles []FileContext, metricsMap map[string]MaintainabilityMetrics) (string, []FileContext) {
	var batchErrorMsg string
	pathsRemaining := validFiles

	for _, plugin := range plugins {
		if len(pathsRemaining) == 0 {
			break
		}

		var errMsg string
		pathsRemaining, errMsg = processSinglePluginMetrics(plugin, toolByPath, pathsRemaining, metricsMap)
		if errMsg != "" {
			batchErrorMsg = errMsg
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
