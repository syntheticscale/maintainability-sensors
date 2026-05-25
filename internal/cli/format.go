package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/syntheticscale/maintainability-sensors/internal/sensors"
)

type ReportOptions struct {
	MarkdownOut string
	JSONOut     string
	HTMLOut     string
	ActionVerb  string
}

func hasViolations(res sensors.OrchestratorResult) bool {
	if !res.ToolingDetected {
		return false
	}
	limits := getEffectiveLimits(res)
	return res.Metrics.Complexity > limits.Complexity ||
		res.Metrics.CognitiveComplexity > limits.CognitiveComplexity ||
		res.Metrics.FunctionLength > limits.FunctionLength ||
		res.Metrics.ArgumentCount > limits.ArgumentCount ||
		res.Metrics.MaxCaseLength > limits.MaxCaseLength
}

func FormatResultsCLI(results []sensors.OrchestratorResult, jsonOutput bool, isDir bool) bool {
	hasV := false
	for _, res := range results {
		if hasViolations(res) {
			hasV = true
			break
		}
	}

	if !isDir {
		if len(results) > 0 {
			printScanResult(results[0], jsonOutput)
		}
		return hasV
	}

	if jsonOutput {
		printJSONResults(results)
		return hasV
	}

	printSummaryTable(results)
	printExceptionsList(results)
	return hasV
}

func printJSONResults(results []sensors.OrchestratorResult) {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		logf(LogLevelError, "[ERROR] Failed to marshal JSON: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

func printSummaryTable(results []sensors.OrchestratorResult) {
	fmt.Fprintf(os.Stderr, "\n=========================================\n")
	fmt.Fprintf(os.Stderr, " Maintainability Sensors Report Summary\n")
	fmt.Fprintf(os.Stderr, "=========================================\n\n")
	fmt.Fprintf(os.Stderr, "%-35s %-12s %-10s %-10s %-10s %-10s %-10s\n", "File", "Lang", "Complexity", "CogCmplx", "FuncLines", "MaxParams", "MaxCaseLn")
	fmt.Fprintf(os.Stderr, "%-35s %-12s %-10s %-10s %-10s %-10s %-10s\n", "----", "----", "----------", "--------", "---------", "---------", "---------")

	blindCount := 0
	for _, res := range results {
		fileBase := filepath.Base(res.FilePath)
		if !res.ToolingDetected {
			blindCount++
			fmt.Fprintf(os.Stderr, "%-35s %-12s %-10s %-10s %-10s %-10s %-10s\n", fileBase, res.Language, "BLIND (L0)", "BLIND (L0)", "BLIND (L0)", "BLIND (L0)", "BLIND (L0)")
		} else {
			fmt.Fprintf(os.Stderr, "%-35s %-12s %-10d %-10d %-10d %-10d %-10d\n", fileBase, res.Language, res.Metrics.Complexity, res.Metrics.CognitiveComplexity, res.Metrics.FunctionLength, res.Metrics.ArgumentCount, res.Metrics.MaxCaseLength)
		}
	}

	if blindCount > 0 {
		fmt.Fprintf(os.Stderr, "\n>>> NOTICE: %d files are running BLIND (Level 0) with no static analysis configs.\n", blindCount)
		fmt.Fprintf(os.Stderr, "    Run 'maintainability-sensors bootstrap' to automatically establish their guardrails!\n")
	}
}

func printExceptionsList(results []sensors.OrchestratorResult) {
	var allExceptions []string
	for _, res := range results {
		if len(res.Exceptions) > 0 {
			var details []string
			for _, exc := range res.Exceptions {
				details = append(details, fmt.Sprintf("%s (%d vs baseline %d)", exc.RuleName, exc.ConfiguredVal, exc.BaselineVal))
			}
			allExceptions = append(allExceptions, fmt.Sprintf("  * %s: %s", filepath.Base(res.FilePath), strings.Join(details, ", ")))
		}
	}

	if len(allExceptions) > 0 {
		fmt.Fprintf(os.Stderr, "\n=========================================\n")
		fmt.Fprintf(os.Stderr, " Configured Exceptions (Relaxed Constraints)\n")
		fmt.Fprintf(os.Stderr, "=========================================\n")
		fmt.Fprintf(os.Stderr, "⚠️  The following files have relaxed rules configured in their linters:\n\n")
		for _, excStr := range allExceptions {
			logLn(LogLevelWarn, excStr)
		}
		fmt.Fprintf(os.Stderr, "\nNOTE: These relaxed thresholds must be manually verified by a human during code review.\n")
		fmt.Fprintf(os.Stderr, "(\"Looking at the exceptions AI created was a good point to start my code review.\")\n")
	}
}

func writeReports(results []sensors.OrchestratorResult, opts ReportOptions) error {
	if opts.MarkdownOut != "" {
		scorecard := GenerateMarkdownScorecard(results)
		if err := os.WriteFile(opts.MarkdownOut, []byte(scorecard), 0644); err != nil {
			return fmt.Errorf("failed to write markdown scorecard: %w", err)
		}
		logf(LogLevelInfo, "[%s] %s markdown report to %s\n", strings.ToUpper(opts.ActionVerb), opts.ActionVerb, opts.MarkdownOut)
	}
	if opts.JSONOut != "" {
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		if err := os.WriteFile(opts.JSONOut, data, 0644); err != nil {
			return fmt.Errorf("failed to write JSON scorecard: %w", err)
		}
		logf(LogLevelInfo, "[%s] %s JSON report to %s\n", strings.ToUpper(opts.ActionVerb), opts.ActionVerb, opts.JSONOut)
	}
	if opts.HTMLOut != "" {
		htmlScorecard := GenerateHTMLScorecard(results)
		if err := os.WriteFile(opts.HTMLOut, []byte(htmlScorecard), 0644); err != nil {
			return fmt.Errorf("failed to write HTML scorecard: %w", err)
		}
		logf(LogLevelInfo, "[%s] %s HTML report to %s\n", strings.ToUpper(opts.ActionVerb), opts.ActionVerb, opts.HTMLOut)
	}
	return nil
}

func printScanResult(res sensors.OrchestratorResult, jsonOutput bool) {
	if jsonOutput {
		data, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			logf(LogLevelError, "[ERROR] Failed to marshal JSON: %v\n", err)
			return
		}
		fmt.Println(string(data))
		return
	}

	fmt.Fprintf(os.Stderr, "\n=========================================\n")
	fmt.Fprintf(os.Stderr, " Maintainability Sensor Result: %s\n", filepath.Base(res.FilePath))
	fmt.Fprintf(os.Stderr, "=========================================\n\n")
	fmt.Fprintf(os.Stderr, "File Path:  %s\n", res.FilePath)
	fmt.Fprintf(os.Stderr, "Language:   %s\n", strings.ToUpper(res.Language))

	if !res.ToolingDetected {
		fmt.Fprintf(os.Stderr, "Status:     RUNNING BLIND (Level 0) ⚠️\n")
		fmt.Fprintf(os.Stderr, "Message:    %s\n", res.Message)
		return
	}

	fmt.Fprintf(os.Stderr, "Status:     ORCHESTRATED (Level 1+) ✅\n\n")
	limits := getEffectiveLimits(res)
	fmt.Fprintf(os.Stderr, "Maintainability Telemetry:\n")
	fmt.Fprintf(os.Stderr, "- Max Cyclomatic Complexity:    %d (Limit: %d)\n", res.Metrics.Complexity, limits.Complexity)
	fmt.Fprintf(os.Stderr, "- Max Cognitive Complexity:     %d (Limit: %d)\n", res.Metrics.CognitiveComplexity, limits.CognitiveComplexity)
	fmt.Fprintf(os.Stderr, "- Max Function Line Count:      %d (Limit: %d)\n", res.Metrics.FunctionLength, limits.FunctionLength)
	fmt.Fprintf(os.Stderr, "- Max Function Parameter Count: %d (Limit: %d)\n", res.Metrics.ArgumentCount, limits.ArgumentCount)
	fmt.Fprintf(os.Stderr, "- Max Switch Case Line Count:   %d (Limit: %d)\n", res.Metrics.MaxCaseLength, limits.MaxCaseLength)

	// Output specific self-correction guidance blocks (Fowler article style)
	printSelfCorrectionGuidance(res)

	// Display Exceptions if any
	if len(res.Exceptions) > 0 {
		fmt.Fprintf(os.Stderr, "\n-----------------------------------------\n")
		fmt.Fprintf(os.Stderr, " Configured Exceptions (Relaxed Constraints):\n")
		fmt.Fprintf(os.Stderr, "-----------------------------------------\n")
		fmt.Fprintf(os.Stderr, "⚠️  The following custom limits are set to relaxed values in the configuration:\n\n")
		for _, exc := range res.Exceptions {
			fmt.Fprintf(os.Stderr, "  * %s: Configured Limit is %d (Standard Baseline is %d)\n", exc.RuleName, exc.ConfiguredVal, exc.BaselineVal)
		}
		fmt.Fprintf(os.Stderr, "\nNOTE: These relaxed thresholds must be manually verified by a human during code review.\n")
		fmt.Fprintf(os.Stderr, "(\"Looking at the exceptions AI created was a good point to start my code review.\")\n")
	}
}

func getSuppressionExample(lang string) string {
	switch lang {
	case "python":
		return "# pylint: disable=... or # noqa"
	case "go":
		return "//nolint:..."
	case "ruby":
		return "# rubocop:disable ..."
	case "javascript", "typescript":
		return "// eslint-disable-next-line ..."
	case "csharp":
		return "#pragma warning disable ..."
	case "java":
		return "@SuppressWarnings(\"...\")"
	default:
		return "// disable-linter-rule ..."
	}
}

func printSelfCorrectionGuidance(res sensors.OrchestratorResult) {
	var guidance []string
	limits := getEffectiveLimits(res)

	if res.Metrics.Complexity > limits.Complexity {
		guidance = append(guidance, fmt.Sprintf("  * Complexity is %d (Max %d). Extract nested conditionals into separate, single-responsibility helper functions.", res.Metrics.Complexity, limits.Complexity))
	}
	if res.Metrics.CognitiveComplexity > limits.CognitiveComplexity {
		guidance = append(guidance, fmt.Sprintf("  * Cognitive Complexity is %d (Max %d). Flatten deeply nested control flow and return early.", res.Metrics.CognitiveComplexity, limits.CognitiveComplexity))
	}
	if res.Metrics.FunctionLength > limits.FunctionLength {
		guidance = append(guidance, fmt.Sprintf("  * Function lines is %d (Max %d). Modularize this block into separate functional components.", res.Metrics.FunctionLength, limits.FunctionLength))
	}
	if res.Metrics.ArgumentCount > limits.ArgumentCount {
		guidance = append(guidance, fmt.Sprintf("  * Parameter count is %d (Max %d). Bundle parameters into a single structured configuration object.", res.Metrics.ArgumentCount, limits.ArgumentCount))
	}
	if res.Metrics.MaxCaseLength > limits.MaxCaseLength {
		guidance = append(guidance, fmt.Sprintf("  * Case block lines is %d (Max %d). Extract the case logic into a well-named method.", res.Metrics.MaxCaseLength, limits.MaxCaseLength))
	}

	if len(guidance) > 0 {
		fmt.Fprintf(os.Stderr, "\n-----------------------------------------\n")
		fmt.Fprintf(os.Stderr, " Actionable Refactoring Prompts:\n")
		fmt.Fprintf(os.Stderr, "-----------------------------------------\n")
		fmt.Fprintf(os.Stderr, "REFACTORING PROMPT: Refactor these violations:\n\n")
		for _, g := range guidance {
			logLn(LogLevelWarn, g)
		}
		suppressionExample := getSuppressionExample(res.Language)
		fmt.Fprintf(os.Stderr, "\nIf refactoring is impossible, REFACTORING PROMPT: suppress the warning with standard inline annotations (e.g. %s).\n", suppressionExample)
	}
}
