package sensors

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
