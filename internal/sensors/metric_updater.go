package sensors

import (
	"path/filepath"
)

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

func updateIfGreater(current *int, value int) {
	if value > *current {
		*current = value
	}
}

func updateSingleMetric(v Violation, metrics *MaintainabilityMetrics) {
	switch v.RuleName {
	case RuleComplexity:
		updateIfGreater(&metrics.Complexity, v.Value)
	case RuleCognitiveComplexity:
		updateIfGreater(&metrics.CognitiveComplexity, v.Value)
	case RuleFunctionLength:
		updateIfGreater(&metrics.FunctionLength, v.Value)
	case RuleArgumentCount:
		updateIfGreater(&metrics.ArgumentCount, v.Value)
	case RuleCaseBlockLength:
		updateIfGreater(&metrics.MaxCaseLength, v.Value)
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
