package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/syntheticscale/maintainability-sensors/internal/sensors"
)

func isTrueViolation(v sensors.Violation, policy *CheckDiffPolicy) bool {
	limit := getThresholdForRule(policy, v.RuleName)
	return v.Value > limit
}

func hasOverlap(v sensors.Violation, ranges []sensors.LineRange) bool {
	for _, r := range ranges {
		if v.StartLine <= r.End && v.EndLine >= r.Start {
			return true
		}
	}
	return false
}

func mapModifiedLinesToAbsPaths(modifiedLines map[string][]sensors.LineRange, targetPath string) map[string][]sensors.LineRange {
	absModifiedLines := make(map[string][]sensors.LineRange)
	absTargetDir, _ := filepath.Abs(targetPath)

	for relPath, ranges := range modifiedLines {
		absPath := filepath.Clean(filepath.Join(absTargetDir, relPath))
		absModifiedLines[absPath] = ranges
	}
	return absModifiedLines
}

func groupFilesByLanguage(files []string, absModifiedLines map[string][]sensors.LineRange) (map[string][]string, map[string]string) {
	groups := make(map[string][]string)
	originalPaths := make(map[string]string)

	for _, p := range files {
		cleanPath := filepath.Clean(p)
		absPath, err := filepath.Abs(cleanPath)
		if err != nil {
			continue
		}

		if _, ok := absModifiedLines[absPath]; ok {
			lang := sensors.DetectLanguage(p)
			if lang != "" {
				groups[lang] = append(groups[lang], p)
				originalPaths[cleanPath] = p
				originalPaths[absPath] = p
			}
		}
	}
	return groups, originalPaths
}

type ViolationCtx struct {
	File       string
	Violations []sensors.Violation
	Ranges     []sensors.LineRange
	Policy     *CheckDiffPolicy
	HasErrors  *bool
	Warnings   *[]string
}

func handleViolationMessage(msg string, isErr bool, ctx *ViolationCtx) {
	fmt.Fprintf(os.Stderr, "%s\n", msg)
	if isErr {
		*ctx.HasErrors = true
	} else {
		*ctx.Warnings = append(*ctx.Warnings, msg)
	}
}

func processSingleViolationFile(ctx ViolationCtx) {
	for _, v := range ctx.Violations {
		if !isTrueViolation(v, ctx.Policy) {
			continue
		}
		if !hasOverlap(v, ctx.Ranges) {
			continue
		}

		isErr, msg := formatViolationMessage(v, ctx.File, ctx.Policy)
		if msg == "" {
			continue
		}
		handleViolationMessage(msg, isErr, &ctx)
	}
}

//nolint:gocognit // maintainability: highly cohesive validation logic
func processViolationsMap(violationsMap map[string][]sensors.Violation, absModifiedLines map[string][]sensors.LineRange, policy *CheckDiffPolicy) (bool, []string) {
	hasErrors := false
	var warnings []string

	for file, violations := range violationsMap {
		absPath, err := filepath.Abs(filepath.Clean(file))
		if err != nil {
			continue
		}

		ranges, hasRanges := absModifiedLines[absPath]
		if !hasRanges {
			continue
		}

		processSingleViolationFile(ViolationCtx{
			File:       file,
			Violations: violations,
			Ranges:     ranges,
			Policy:     policy,
			HasErrors:  &hasErrors,
			Warnings:   &warnings,
		})
	}
	return hasErrors, warnings
}

func formatViolationMessage(v sensors.Violation, file string, policy *CheckDiffPolicy) (bool, string) {
	sev := getSeverityForRule(policy, v.RuleName)
	if sev == SeverityIgnore {
		return false, ""
	}
	msg := fmt.Sprintf("REFACTORING PROMPT: %s:%d - %s - %s", file, v.StartLine, v.RuleName, v.Message)
	if sev == SeverityWarn {
		return false, msg
	}
	return true, msg
}

func processDeltaGroupForLang(lang string, langFiles []string, originalPaths map[string]string, absModifiedLines map[string][]sensors.LineRange, policy *CheckDiffPolicy) bool {
	if len(langFiles) == 0 {
		return false
	}

	fileContexts := make([]sensors.FileContext, len(langFiles))
	for i, f := range langFiles {
		fileContexts[i] = sensors.FileContext{Path: f}
	}

	violationsMap, err := sensors.ScanDeltaBatch(fileContexts, originalPaths, lang)
	if err != nil {
		logf(LogLevelWarn, "[WARNING] Delta scan failed for %s: %v\n", lang, err)
		return false
	}

	hasErrs, _ := processViolationsMap(violationsMap, absModifiedLines, policy)
	return hasErrs
}

func processDeltaGroups(groups map[string][]string, originalPaths map[string]string, absModifiedLines map[string][]sensors.LineRange, policy *CheckDiffPolicy) bool {
	hasErrors := false
	for lang, langFiles := range groups {
		if processDeltaGroupForLang(lang, langFiles, originalPaths, absModifiedLines, policy) {
			hasErrors = true
		}
	}
	return hasErrors
}
