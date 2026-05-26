package sensors

import (
	"fmt"
)

type ProcessDeltaCtx struct {
	Plugins       []Plugin
	ToolByPath    map[string]ConfigParser
	ValidPaths    []FileContext
	OriginalPaths map[string]string
	MetricsMap    map[string][]Violation
}

func processSinglePluginDelta(plugin Plugin, pathsRemaining []FileContext, ctx ProcessDeltaCtx) ([]FileContext, error) {
	pathsForPlugin := filterPathsForPlugin(pathsRemaining, plugin, ctx.ToolByPath)
	if len(pathsForPlugin) == 0 {
		return pathsRemaining, nil
	}

	toolMetrics, err := analyzeInChunks(plugin, pathsForPlugin)
	if err != nil {
		return nil, fmt.Errorf("Orchestration error (%s): %w", plugin.Name(), err)
	}

	if toolMetrics != nil {
		return updateDeltaMetricsMap(UpdateDeltaCtx{
			MetricsMap:     ctx.MetricsMap,
			ToolMetrics:    toolMetrics,
			OriginalPaths:  ctx.OriginalPaths,
			PathsRemaining: pathsRemaining,
			PathsForPlugin: pathsForPlugin,
		}), nil
	}
	return pathsRemaining, nil
}

func processPluginsDelta(ctx ProcessDeltaCtx) ([]FileContext, error) {
	pathsRemaining := ctx.ValidPaths

	for _, plugin := range ctx.Plugins {
		if len(pathsRemaining) == 0 {
			break
		}
		var err error
		pathsRemaining, err = processSinglePluginDelta(plugin, pathsRemaining, ctx)
		if err != nil {
			return nil, err
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
