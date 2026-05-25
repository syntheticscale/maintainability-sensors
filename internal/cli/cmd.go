package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/syntheticscale/maintainability-sensors/internal/lsp"
	"github.com/syntheticscale/maintainability-sensors/internal/sensors"
)

var cli struct {
	Quiet     bool         `short:"q" help:"Suppress non-critical diagnostic output (stderr)."`
	Run       runCmd       `cmd:"" help:"Scan a specific file or folder for maintainability warnings."`
	Generate  generateCmd  `cmd:"" help:"Reconstruct visual reports from a saved JSON scorecard (the Single Source of Truth)."`
	Bootstrap bootstrapCmd `cmd:"" help:"Auto-detect repository language and bootstrap pristine, non-overwriting maintainability configuration files (TS, Python, Go, Java, Ruby, C#)."`
	CheckDiff CheckDiffCmd `cmd:"" name:"check-diff" help:"Check maintainability delta against a target branch."`
	Lsp       lspCmd       `cmd:"" help:"Start the Language Server Protocol (LSP) wrapper."`
}

type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

func logf(level LogLevel, format string, a ...interface{}) {
	if cli.Quiet && level < LogLevelWarn {
		return
	}
	fmt.Fprintf(os.Stderr, format, a...)
}

func logLn(level LogLevel, a ...interface{}) {
	if cli.Quiet && level < LogLevelWarn {
		return
	}
	fmt.Fprintln(os.Stderr, a...)
}

type runCmd struct {
	Path        string `arg:"" optional:"" default:"." help:"Target path to scan (file or directory)."`
	Json        bool   `help:"Output result in raw JSON format."`
	GithubPr    bool   `help:"Post markdown scorecard directly as a GitHub PR comment."`
	MarkdownOut string `help:"Write beautiful markdown scorecard to specified file path."`
	JsonOut     string `help:"Write raw JSON metric payload to specified file path."`
	HtmlOut     string `help:"Write beautiful dark-themed HTML scorecard to specified file path."`
}

func (c *runCmd) Run() error {
	executeRun(RunOptions{
		TargetPath:  c.Path,
		JSONOutput:  c.Json,
		GithubPR:    c.GithubPr,
		MarkdownOut: c.MarkdownOut,
		JSONOutFile: c.JsonOut,
		HTMLOut:     c.HtmlOut,
	})
	return nil
}

type generateCmd struct {
	JsonIn      string `arg:"" help:"Input JSON file path."`
	MarkdownOut string `help:"Write beautiful markdown scorecard to specified file path."`
	HtmlOut     string `help:"Write beautiful dark-themed HTML scorecard to specified file path."`
}

func (c *generateCmd) Run() error {
	executeGenerate(c.JsonIn, c.MarkdownOut, c.HtmlOut)
	return nil
}

type bootstrapCmd struct {
	Path           string `arg:"" optional:"" default:"." help:"Target path to bootstrap."`
	WithWarnPolicy bool   `optional:"" name:"with-warn-policy" help:"Generate a .maintainability-sensors.yml with default-severity: warn."`
}

func (c *bootstrapCmd) Run() error {
	executeBootstrap(c.Path, c.WithWarnPolicy)
	return nil
}

type CheckDiffCmd struct {
	TargetBranch    string   `optional:"" default:"HEAD" help:"Target branch to diff against."`
	TargetPath      string   `arg:"" optional:"" default:"." help:"Target path to diff."`
	Config          string   `optional:"" help:"Path to .maintainability-sensors.yml config file."`
	DefaultSeverity string   `optional:"" help:"Default severity level for rules not explicitly configured (error|warn|ignore). Defaults to error."`
	Severity        []string `optional:"" name:"severity" help:"Per-rule severity overrides (format: Rule:level)."`
}

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

func processSingleViolationFile(ctx ViolationCtx) {
	for _, v := range ctx.Violations {
		if !isTrueViolation(v, ctx.Policy) || !hasOverlap(v, ctx.Ranges) {
			continue
		}

		isErr, msg := formatViolationMessage(v, ctx.File, ctx.Policy)
		if msg != "" {
			fmt.Fprintf(os.Stderr, "%s\n", msg)
			if isErr {
				*ctx.HasErrors = true
			} else {
				*ctx.Warnings = append(*ctx.Warnings, msg)
			}
		}
	}
}

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

func (c *CheckDiffCmd) Run() error {
	policy, err := loadCheckDiffPolicy(c)
	if err != nil {
		return err
	}

	modifiedLines, err := sensors.GetModifiedLines(c.TargetBranch, c.TargetPath)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get modified lines: %v", err)
	}

	files, _, err := FindFiles(c.TargetPath)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to find files: %v", err)
	}

	absModifiedLines := mapModifiedLinesToAbsPaths(modifiedLines, c.TargetPath)
	groups, originalPaths := groupFilesByLanguage(files, absModifiedLines)

	hasErrors := processDeltaGroups(groups, originalPaths, absModifiedLines, policy)

	if hasErrors {
		return fmt.Errorf("Delta violations found")
	}

	logLn(LogLevelInfo, "Delta clean.")
	return nil
}

type lspCmd struct{}

func (c *lspCmd) Run() error {
	return lsp.StartServer()
}

func loadCheckDiffPolicy(c *CheckDiffCmd) (*CheckDiffPolicy, error) {
	policy, err := LoadPolicy(c.Config, c.DefaultSeverity, c.Severity)
	if err != nil {
		return nil, fmt.Errorf("[ERROR] Failed to load policy: %v", err)
	}

	if c.Config == "" {
		foundConfig := findConfigFile(c.TargetPath)
		if foundConfig != "" {
			policy, err = LoadPolicy(foundConfig, c.DefaultSeverity, c.Severity)
			if err != nil {
				return nil, fmt.Errorf("[ERROR] Failed to load policy: %v", err)
			}
		}
	}
	return policy, nil
}

func processDeltaGroups(groups map[string][]string, originalPaths map[string]string, absModifiedLines map[string][]sensors.LineRange, policy *CheckDiffPolicy) bool {
	hasErrors := false
	for lang, langFiles := range groups {
		if len(langFiles) == 0 {
			continue
		}

		fileContexts := make([]sensors.FileContext, len(langFiles))
		for i, f := range langFiles {
			fileContexts[i] = sensors.FileContext{Path: f}
		}

		violationsMap, err := sensors.ScanDeltaBatch(fileContexts, originalPaths, lang)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[WARNING] Delta scan failed for %s: %v\n", lang, err)
			continue
		}

		if langErrs, _ := processViolationsMap(violationsMap, absModifiedLines, policy); langErrs {
			hasErrors = true
		}
	}
	return hasErrors
}

// Execute runs the main CLI command-line parser.
func Execute() {
	ctx := kong.Parse(&cli,
		kong.Name("maintainability-sensors"),
		kong.Description("Maintainability Sensors for Coding Agents CLI 📡\n\nExamples:\n  maintainability-sensors check-diff\n  maintainability-sensors run .\n  maintainability-sensors run . --markdown-out=report.md --html-out=report.html\n  maintainability-sensors run src/api.py --json\n  maintainability-sensors generate report.json --html-out=report.html --markdown-out=report.md\n  maintainability-sensors bootstrap /path/to/my/project\n  maintainability-sensors -q run ."),
		kong.UsageOnError(),
	)
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}

func isSkippedDir(dirName string) bool {
	switch dirName {
	case "node_modules", ".git", "vendor", "bin", ".cache", ".venv", "venv", "env":
		return true
	}
	return false
}

func isValidExtension(ext string) bool {
	switch ext {
	case ".ts", ".tsx", ".js", ".jsx", ".py", ".go", ".rb", ".cs", ".java":
		return true
	}
	return false
}

func checkWalkDirPath(path string, absTargetDir string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	if resolvedPath, err := filepath.EvalSymlinks(absPath); err == nil {
		absPath = resolvedPath
	}
	if !strings.HasPrefix(absPath, absTargetDir) {
		return ""
	}
	return path
}

func processWalkDirFile(path string, d os.DirEntry, absTargetDir string) (string, error) {
	if d.IsDir() {
		if isSkippedDir(d.Name()) {
			return "", filepath.SkipDir
		}
		return "", nil
	}

	info, err := d.Info()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[WARNING] Cannot get info for %s: %v\n", path, err)
		return "", nil
	}
	if !info.Mode().IsRegular() {
		return "", nil
	}
	if info.Size() > 2*1024*1024 {
		fmt.Fprintf(os.Stderr, "[WARNING] Skipping file %s: exceeds 2MB limit\n", path)
		return "", nil
	}

	if !isValidExtension(filepath.Ext(path)) {
		return "", nil
	}

	return checkWalkDirPath(path, absTargetDir), nil
}

func resolveSingleFile(cleanPath string, absTargetDir string) []string {
	var files []string
	absPath, err := filepath.Abs(cleanPath)
	if err == nil {
		if resolvedPath, err := filepath.EvalSymlinks(absPath); err == nil {
			absPath = resolvedPath
		}
		if strings.HasPrefix(absPath, absTargetDir+string(filepath.Separator)) || absPath == absTargetDir {
			files = append(files, cleanPath)
		}
	}
	return files
}

func FindFiles(targetPath string) ([]string, bool, error) {
	cleanPath := filepath.Clean(targetPath)

	absTargetDir, err := filepath.Abs(cleanPath)
	if err != nil {
		return nil, false, fmt.Errorf("[ERROR] Failed to get absolute path of target directory: %v", err)
	}

	if resolvedTargetDir, err := filepath.EvalSymlinks(absTargetDir); err == nil {
		absTargetDir = resolvedTargetDir
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return nil, false, fmt.Errorf("[ERROR] Path does not exist: %s", targetPath)
	}

	isDir := info.IsDir()
	if !isDir {
		return resolveSingleFile(cleanPath, absTargetDir), false, nil
	}

	var files []string
	err = filepath.WalkDir(cleanPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "[WARNING] Cannot access %s: %v\n", path, err)
			return nil
		}
		file, walkErr := processWalkDirFile(path, d, absTargetDir)
		if file != "" {
			files = append(files, file)
		}
		return walkErr
	})

	if err != nil {
		return nil, true, fmt.Errorf("[ERROR] Directory scan failed: %v", err)
	}

	return files, true, nil
}

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
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to marshal JSON: %v\n", err)
		os.Exit(1)
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
		fmt.Fprintf(os.Stderr, "[%s] %s markdown report to %s\n", strings.ToUpper(opts.ActionVerb), opts.ActionVerb, opts.MarkdownOut)
	}
	if opts.JSONOut != "" {
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		if err := os.WriteFile(opts.JSONOut, data, 0644); err != nil {
			return fmt.Errorf("failed to write JSON scorecard: %w", err)
		}
		fmt.Fprintf(os.Stderr, "[%s] %s JSON report to %s\n", strings.ToUpper(opts.ActionVerb), opts.ActionVerb, opts.JSONOut)
	}
	if opts.HTMLOut != "" {
		htmlScorecard := GenerateHTMLScorecard(results)
		if err := os.WriteFile(opts.HTMLOut, []byte(htmlScorecard), 0644); err != nil {
			return fmt.Errorf("failed to write HTML scorecard: %w", err)
		}
		fmt.Fprintf(os.Stderr, "[%s] %s HTML report to %s\n", strings.ToUpper(opts.ActionVerb), opts.ActionVerb, opts.HTMLOut)
	}
	return nil
}

func printScanResult(res sensors.OrchestratorResult, jsonOutput bool) {
	if jsonOutput {
		data, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to marshal JSON: %v\n", err)
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
