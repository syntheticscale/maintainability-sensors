package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/alecthomas/kong"
	"github.com/syntheticscale/maintainability-sensors/sensors"
	"golang.org/x/sync/errgroup"
)

var cli struct {
	Run       runCmd       `cmd:"" help:"Scan a specific file or folder for maintainability warnings."`
	Generate  generateCmd  `cmd:"" help:"Reconstruct visual reports from a saved JSON scorecard (the Single Source of Truth)."`
	Bootstrap bootstrapCmd `cmd:"" help:"Auto-detect repository language and bootstrap pristine, non-overwriting maintainability configuration files (TS, Python, Go, Java, Ruby, C#)."`
	CheckDiff CheckDiffCmd `cmd:"" name:"check-diff" help:"Check maintainability delta against a target branch."`
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
	Path string `arg:"" optional:"" default:"." help:"Target path to bootstrap."`
}

func (c *bootstrapCmd) Run() error {
	executeBootstrap(c.Path)
	return nil
}

type CheckDiffCmd struct {
	TargetBranch string `optional:"" default:"HEAD" help:"Target branch to diff against."`
	TargetPath   string `arg:"" optional:"" default:"." help:"Target path to diff."`
}

func (c *CheckDiffCmd) Run() error {
	modifiedLines, err := sensors.GetModifiedLines(c.TargetBranch, c.TargetPath)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to get modified lines: %v", err)
	}

	files, _, err := FindFiles(c.TargetPath)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to find files: %v", err)
	}

	hasDeltaViolations := false

	absModifiedLines := make(map[string][]sensors.LineRange)
	absTargetDir, _ := filepath.Abs(c.TargetPath)

	for relPath, ranges := range modifiedLines {
		absPath := filepath.Clean(filepath.Join(absTargetDir, relPath))
		absModifiedLines[absPath] = ranges
	}

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

	for lang, langFiles := range groups {
		if len(langFiles) == 0 {
			continue
		}

		violationsMap, err := sensors.ScanDeltaBatch(langFiles, originalPaths, lang)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[WARNING] Delta scan failed for %s: %v\n", lang, err)
			continue
		}

		for file, violations := range violationsMap {
			absPath, err := filepath.Abs(filepath.Clean(file))
			if err != nil {
				continue
			}

			ranges, hasRanges := absModifiedLines[absPath]
			if !hasRanges {
				continue
			}

			for _, v := range violations {
				overlaps := false
				for _, r := range ranges {
										if v.StartLine <= r.End && v.EndLine >= r.Start {
						overlaps = true
						break
					}
				}
				if overlaps {
					fmt.Fprintf(os.Stderr, "AI WARNING: %s:%d - %s - %s\n", file, v.StartLine, v.RuleName, v.Message)
					hasDeltaViolations = true
				}
			}
		}
	}

	if hasDeltaViolations {
		return fmt.Errorf("Delta violations found")
	}

	fmt.Fprintln(os.Stderr, "Delta clean.")
	return nil
}

// Execute runs the main CLI command-line parser.
func Execute() {
	ctx := kong.Parse(&cli,
		kong.Name("maintainability-sensors"),
		kong.Description("Maintainability Sensors for Coding Agents CLI 📡\n\nExamples:\n  maintainability-sensors run .\n  maintainability-sensors run . --markdown-out=report.md --html-out=report.html\n  maintainability-sensors run src/api.py --json\n  maintainability-sensors generate report.json --html-out=report.html --markdown-out=report.md\n  maintainability-sensors bootstrap /path/to/my/project"),
		kong.UsageOnError(),
	)
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
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

	var files []string
	isDir := info.IsDir()

	if !isDir {
		absPath, err := filepath.Abs(cleanPath)
		if err == nil {
			if resolvedPath, err := filepath.EvalSymlinks(absPath); err == nil {
				absPath = resolvedPath
			}
			if strings.HasPrefix(absPath, absTargetDir) {
				files = append(files, cleanPath)
			}
		}
		return files, false, nil
	}

	err = filepath.WalkDir(cleanPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "[WARNING] Cannot access %s: %v\n", path, err)
			return nil
		}
		if d.IsDir() {
			dirName := d.Name()
			if dirName == "node_modules" || dirName == ".git" || dirName == "vendor" || dirName == "bin" || dirName == ".cache" || dirName == ".venv" || dirName == "venv" || dirName == "env" {
				return filepath.SkipDir
			}
			return nil
		}

		info, err := d.Info()
		if err != nil {
		        fmt.Fprintf(os.Stderr, "[WARNING] Cannot get info for %s: %v\n", path, err)
		        return nil
		}
		if !info.Mode().IsRegular() {
		        return nil
		}
		if info.Size() > 2*1024*1024 {			fmt.Fprintf(os.Stderr, "[WARNING] Skipping file %s: exceeds 2MB limit\n", path)
			return nil
		}

		// Skip files without recognized extension
		ext := filepath.Ext(path)
		if ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" && ext != ".py" && ext != ".go" && ext != ".rb" && ext != ".cs" {
			return nil
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil
		}
		if resolvedPath, err := filepath.EvalSymlinks(absPath); err == nil {
			absPath = resolvedPath
		}
		if !strings.HasPrefix(absPath, absTargetDir) {
			return nil
		}

		files = append(files, path)
		return nil
	})

	if err != nil {
		return nil, true, fmt.Errorf("[ERROR] Directory scan failed: %v", err)
	}

	return files, true, nil
}

func ScanFiles(filePaths []string, isDir bool) ([]sensors.OrchestratorResult, error) {
	groups := make(map[string][]string)
	for _, p := range filePaths {
		lang := sensors.DetectLanguage(p)
		groups[lang] = append(groups[lang], p)
	}

	var allResults []sensors.OrchestratorResult
	var mu sync.Mutex
	var g errgroup.Group

	for lang, files := range groups {
		lang, files := lang, files
		g.Go(func() error {
			if lang == "" {
				for _, f := range files {
					if !isDir {
						return fmt.Errorf("unsupported or unrecognized language file: %s", f)
					}
					fmt.Fprintf(os.Stderr, "[WARNING] Scan failed for %s: unsupported or unrecognized language file: %s\n", f, f)
				}
				return nil
			}

			res, err := sensors.OrchestratedScanBatch(files, lang)
			if err != nil {
				if !isDir {
					return fmt.Errorf("Scan failed: %v", err)
				}
				fmt.Fprintf(os.Stderr, "[WARNING] Scan failed for language %s: %v\n", lang, err)
				return nil
			}

			mu.Lock()
			allResults = append(allResults, res...)
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return allResults, nil
}

type RunOptions struct {
	TargetPath  string
	JSONOutput  bool
	GithubPR    bool
	MarkdownOut string
	JSONOutFile string
	HTMLOut     string
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
	limitComplexity := sensors.BaselineComplexity
	limitLength := sensors.BaselineFunctionLength
	limitArgs := sensors.BaselineArgumentCount

	for _, exc := range res.Exceptions {
		if exc.RuleName == "Cyclomatic Complexity" {
			limitComplexity = exc.ConfiguredVal
		} else if exc.RuleName == "Function Length" {
			limitLength = exc.ConfiguredVal
		} else if exc.RuleName == "Argument Count" {
			limitArgs = exc.ConfiguredVal
		}
	}

	return res.Metrics.Complexity > limitComplexity ||
		res.Metrics.FunctionLength > limitLength ||
		res.Metrics.ArgumentCount > limitArgs
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
	fmt.Fprintf(os.Stderr, "%-35s %-12s %-10s %-10s %-10s\n", "File", "Lang", "Complexity", "FuncLines", "MaxParams")
	fmt.Fprintf(os.Stderr, "%-35s %-12s %-10s %-10s %-10s\n", "----", "----", "----------", "---------", "---------")

	blindCount := 0
	for _, res := range results {
		fileBase := filepath.Base(res.FilePath)
		if !res.ToolingDetected {
			blindCount++
			fmt.Fprintf(os.Stderr, "%-35s %-12s %-10s %-10s %-10s\n", fileBase, res.Language, "BLIND (L0)", "BLIND (L0)", "BLIND (L0)")
		} else {
			fmt.Fprintf(os.Stderr, "%-35s %-12s %-10d %-10d %-10d\n", fileBase, res.Language, res.Metrics.Complexity, res.Metrics.FunctionLength, res.Metrics.ArgumentCount)
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
		fmt.Fprintf(os.Stderr, " Exceptions Created by AI (Relaxed Constraints)\n")
		fmt.Fprintf(os.Stderr, "=========================================\n")
		fmt.Fprintf(os.Stderr, "⚠️  The following files have relaxed rules configured in their linters:\n\n")
		for _, excStr := range allExceptions {
			fmt.Fprintln(os.Stderr, excStr)
		}
		fmt.Fprintf(os.Stderr, "\nNOTE: These relaxed thresholds must be manually verified by a human during code review.\n")
		fmt.Fprintf(os.Stderr, "(\"Looking at the exceptions AI created was a good point to start my code review.\")\n")
	}
}

func executeRun(opts RunOptions) {
	files, isDir, err := FindFiles(opts.TargetPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if isDir && len(files) == 0 {
		fmt.Fprintln(os.Stderr, "No supported source files (TS/JS, Python, Go) found in target directory.")
		return
	}

	results, err := ScanFiles(files, isDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		os.Exit(1)
	}

	if isDir && len(results) == 0 {
		fmt.Fprintln(os.Stderr, "No supported source files (TS/JS, Python, Go) found in target directory.")
		return
	}

	hasV := FormatResultsCLI(results, opts.JSONOutput, isDir)

	if os.Getenv("GITHUB_ACTIONS") == "true" {
		scorecard := GenerateMarkdownScorecard(results)
		if err := WriteGitHubStepSummary(scorecard); err != nil {
			fmt.Fprintf(os.Stderr, "[WARNING] Failed to write GitHub Step Summary: %v\n", err)
		}
	}

	isCI_PR := os.Getenv("GITHUB_TOKEN") != "" && (os.Getenv("GITHUB_EVENT_PATH") != "" || os.Getenv("GITHUB_REF") != "")
	if opts.GithubPR || isCI_PR {
		fmt.Fprintln(os.Stderr, "Posting inline review to GitHub PR...")
		if err := PostGitHubReview(results); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to post GitHub inline review: %v\n", err)
		} else {
			fmt.Fprintln(os.Stderr, "Successfully posted inline review to GitHub PR!")
		}
	}

	err = writeReports(results, ReportOptions{
		MarkdownOut: opts.MarkdownOut,
		JSONOut:     opts.JSONOutFile,
		HTMLOut:     opts.HTMLOut,
		ActionVerb:  "Saved",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		os.Exit(1)
	}

	if hasV {
		os.Exit(1)
	}
}

func executeBootstrap(targetPath string) {
	err := sensors.BootstrapRepo(targetPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Bootstrap failed: %v\n", err)
		os.Exit(1)
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

func executeGenerate(jsonIn string, markdownOut string, htmlOut string) {
	data, err := os.ReadFile(jsonIn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to read JSON input file: %v\n", err)
		os.Exit(1)
	}

	var results []sensors.OrchestratorResult
	if err := json.Unmarshal(data, &results); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to parse JSON input scorecard: %v\n", err)
		os.Exit(1)
	}

	for i, res := range results {
		if res.FilePath == "" {
			fmt.Fprintf(os.Stderr, "[ERROR] Validation failed: Missing 'file_path' in result at index %d\n", i)
			os.Exit(1)
		}
		if res.Language == "" {
			fmt.Fprintf(os.Stderr, "[ERROR] Validation failed: Missing 'language' in result at index %d\n", i)
			os.Exit(1)
		}
	}

	hasV := false
	for _, res := range results {
		if hasViolations(res) {
			hasV = true
			break
		}
	}

	if err := writeReports(results, ReportOptions{
		MarkdownOut: markdownOut,
		HTMLOut:     htmlOut,
		ActionVerb:  "Generated",
	}); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		os.Exit(1)
	}

	if hasV {
		os.Exit(1)
	}
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
	fmt.Fprintf(os.Stderr, "Maintainability Telemetry:\n")
	fmt.Fprintf(os.Stderr, "- Max Cyclomatic Complexity:    %d (Limit: %d)\n", res.Metrics.Complexity, sensors.BaselineComplexity)
	fmt.Fprintf(os.Stderr, "- Max Function Line Count:      %d (Limit: %d)\n", res.Metrics.FunctionLength, sensors.BaselineFunctionLength)
	fmt.Fprintf(os.Stderr, "- Max Function Parameter Count: %d (Limit: %d)\n", res.Metrics.ArgumentCount, sensors.BaselineArgumentCount)

	// Output specific self-correction guidance blocks (Fowler article style)
	printSelfCorrectionGuidance(res)

	// Display Exceptions if any
	if len(res.Exceptions) > 0 {
		fmt.Fprintf(os.Stderr, "\n-----------------------------------------\n")
		fmt.Fprintf(os.Stderr, " Exceptions Created by AI (Relaxed Constraints):\n")
		fmt.Fprintf(os.Stderr, "-----------------------------------------\n")
		fmt.Fprintf(os.Stderr, "⚠️  The following custom limits are set to relaxed values in the configuration:\n\n")
		for _, exc := range res.Exceptions {
			fmt.Fprintf(os.Stderr, "  * %s: Configured Limit is %d (Standard Baseline is %d)\n", exc.RuleName, exc.ConfiguredVal, exc.BaselineVal)
		}
		fmt.Fprintf(os.Stderr, "\nNOTE: These relaxed thresholds must be manually verified by a human during code review.\n")
		fmt.Fprintf(os.Stderr, "(\"Looking at the exceptions AI created was a good point to start my code review.\")\n")
	}
}

func printSelfCorrectionGuidance(res sensors.OrchestratorResult) {
	hasViolation := false
	var guidance []string

	limitComplexity := sensors.BaselineComplexity
	limitLength := sensors.BaselineFunctionLength
	limitArgs := sensors.BaselineArgumentCount

	for _, exc := range res.Exceptions {
		if exc.RuleName == "Cyclomatic Complexity" {
			limitComplexity = exc.ConfiguredVal
		} else if exc.RuleName == "Function Length" {
			limitLength = exc.ConfiguredVal
		} else if exc.RuleName == "Argument Count" {
			limitArgs = exc.ConfiguredVal
		}
	}

	if res.Metrics.Complexity > limitComplexity {
		hasViolation = true
		guidance = append(guidance, fmt.Sprintf("  * Complexity is %d (Max %d). Nudge coding agent to extract nested conditionals into separate, single-responsibility helper functions.", res.Metrics.Complexity, limitComplexity))
	}
	if res.Metrics.FunctionLength > limitLength {
		hasViolation = true
		guidance = append(guidance, fmt.Sprintf("  * Function lines is %d (Max %d). Nudge coding agent to modularize this block into separate functional components.", res.Metrics.FunctionLength, limitLength))
	}
	if res.Metrics.ArgumentCount > limitArgs {
		hasViolation = true
		guidance = append(guidance, fmt.Sprintf("  * Parameter count is %d (Max %d). Nudge coding agent to bundle parameters into a single structured configuration object.", res.Metrics.ArgumentCount, limitArgs))
	}

	if hasViolation {
		fmt.Fprintf(os.Stderr, "\n-----------------------------------------\n")
		fmt.Fprintf(os.Stderr, " AI Agent Self-Correction Prompts:\n")
		fmt.Fprintf(os.Stderr, "-----------------------------------------\n")
		fmt.Fprintf(os.Stderr, "AI WARNING: Refactor these violations:\n\n")
		for _, g := range guidance {
			fmt.Fprintln(os.Stderr, g)
		}
		fmt.Fprintf(os.Stderr, "\nIf refactoring is impossible, AI WARNING: suppress the warning with standard inline annotations (e.g. // eslint-disable-next-line -- reason).\n")
	}
}
	for _, g := range guidance {
			fmt.Fprintln(os.Stderr, g)
		}
		fmt.Fprintf(os.Stderr, "\nIf refactoring is impossible, AI WARNING: suppress the warning with standard inline annotations (e.g. // eslint-disable-next-line -- reason).\n")
	}
}
