package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/paulolai/maintainability-sensors/sensors"
	"sync"
	"golang.org/x/sync/errgroup"
)

// Execute runs the main CLI command-line parser.
func Execute() {
	if len(os.Args) < 2 {
		printGeneralUsage()
		os.Exit(0)
	}

	subcommand := os.Args[1]

	switch subcommand {
	case "run":
		runCmd := flag.NewFlagSet("run", flag.ExitOnError)
		jsonOut := runCmd.Bool("json", false, "output result in raw JSON format")
		githubPR := runCmd.Bool("github-pr", false, "post markdown scorecard directly as a GitHub PR comment")
		markdownOut := runCmd.String("markdown-out", "", "write beautiful markdown scorecard to specified file path")
		jsonOutFile := runCmd.String("json-out", "", "write raw JSON metric payload to specified file path")
		htmlOut := runCmd.String("html-out", "", "write beautiful dark-themed HTML scorecard to specified file path")
		_ = runCmd.Parse(os.Args[2:])

		targetPath := "."
		if len(runCmd.Args()) > 0 {
			targetPath = runCmd.Arg(0)
		}

		executeRun(RunOptions{
			TargetPath:  targetPath,
			JSONOutput:  *jsonOut,
			GithubPR:    *githubPR,
			MarkdownOut: *markdownOut,
			JSONOutFile: *jsonOutFile,
			HTMLOut:     *htmlOut,
		})

	case "generate":
		genCmd := flag.NewFlagSet("generate", flag.ExitOnError)
		markdownOut := genCmd.String("markdown-out", "", "write beautiful markdown scorecard to specified file path")
		htmlOut := genCmd.String("html-out", "", "write beautiful dark-themed HTML scorecard to specified file path")
		_ = genCmd.Parse(os.Args[2:])

		if len(genCmd.Args()) < 1 {
			fmt.Fprintln(os.Stderr, "[ERROR] Missing input JSON file path for generate subcommand.")
			os.Exit(1)
		}
		jsonIn := genCmd.Arg(0)
		executeGenerate(jsonIn, *markdownOut, *htmlOut)

	case "bootstrap":
		bootCmd := flag.NewFlagSet("bootstrap", flag.ExitOnError)
		_ = bootCmd.Parse(os.Args[2:])

		targetPath := "."
		if len(bootCmd.Args()) > 0 {
			targetPath = bootCmd.Arg(0)
		}

		executeBootstrap(targetPath)

	case "-h", "--help", "help":
		printGeneralUsage()

	default:
		fmt.Fprintf(os.Stderr, "[ERROR] Unknown subcommand: %s\n\n", subcommand)
		printGeneralUsage()
		os.Exit(1)
	}
}

func printGeneralUsage() {
	fmt.Printf("Maintainability Sensors for Coding Agents CLI 📡\n")
	fmt.Printf("Usage: maintainability-sensors <subcommand> [args]\n\n")
	fmt.Printf("Subcommands:\n")
	fmt.Printf("  run [path]        Scan a specific file or folder for maintainability warnings.\n")
	fmt.Printf("                    Optional flag: --json (outputs raw JSON metric payload to stdout).\n")
	fmt.Printf("                    Optional flag: --github-pr (post markdown scorecard directly as a GitHub PR comment).\n")
	fmt.Printf("                    Optional flag: --markdown-out [file-path] (writes beautiful markdown scorecard to specified file path).\n")
	fmt.Printf("                    Optional flag: --json-out [file-path] (writes raw JSON metric payload to specified file path).\n")
	fmt.Printf("                    Optional flag: --html-out [file-path] (writes beautiful dark-themed HTML scorecard to specified file path).\n")
	fmt.Printf("  generate [json-in] Reconstruct visual reports from a saved JSON scorecard (the Single Source of Truth).\n")
	fmt.Printf("                    Optional flag: --markdown-out [file-path] (writes beautiful markdown scorecard to specified file path).\n")
	fmt.Printf("                    Optional flag: --html-out [file-path] (writes beautiful dark-themed HTML scorecard to specified file path).\n")
	fmt.Printf("  bootstrap [path]  Auto-detect repository language and bootstrap pristine, non-overwriting\n")
	fmt.Printf("                    maintainability configuration files (TS, Python, Go, Java, Ruby, C#).\n\n")
	fmt.Printf("Examples:\n")
	fmt.Printf("  maintainability-sensors run .\n")
	fmt.Printf("  maintainability-sensors run . --markdown-out=report.md --html-out=report.html\n")
	fmt.Printf("  maintainability-sensors run src/api.py --json\n")
	fmt.Printf("  maintainability-sensors generate report.json --html-out=report.html --markdown-out=report.md\n")
	fmt.Printf("  maintainability-sensors bootstrap /path/to/my/project\n")
}

func FindFiles(targetPath string) ([]string, bool, error) {
	cleanPath := filepath.Clean(targetPath)

	info, err := os.Stat(cleanPath)
	if err != nil {
		return nil, false, fmt.Errorf("[ERROR] Path does not exist: %s", targetPath)
	}

	var files []string
	isDir := info.IsDir()

	if !isDir {
		files = append(files, cleanPath)
		return files, false, nil
	}

	err = filepath.Walk(cleanPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "[WARNING] Cannot access %s: %v\n", path, err)
			return nil
		}
		if info.IsDir() {
			dirName := info.Name()
			if dirName == "node_modules" || dirName == ".git" || dirName == "vendor" || dirName == "bin" || dirName == ".cache" || dirName == ".venv" || dirName == "venv" || dirName == "env" {
				return filepath.SkipDir
			}
			return nil
		}
		// Skip files without recognized extension
		ext := filepath.Ext(path)
		if ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" && ext != ".py" && ext != ".go" && ext != ".rb" && ext != ".cs" {
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
	fmt.Printf("\n=========================================\n")
	fmt.Printf(" Maintainability Sensors Report Summary\n")
	fmt.Printf("=========================================\n\n")
	fmt.Printf("%-35s %-12s %-10s %-10s %-10s\n", "File", "Lang", "Complexity", "FuncLines", "MaxParams")
	fmt.Printf("%-35s %-12s %-10s %-10s %-10s\n", "----", "----", "----------", "---------", "---------")

	blindCount := 0
	for _, res := range results {
		fileBase := filepath.Base(res.FilePath)
		if !res.ToolingDetected {
			blindCount++
			fmt.Printf("%-35s %-12s %-10s %-10s %-10s\n", fileBase, res.Language, "BLIND (L0)", "BLIND (L0)", "BLIND (L0)")
		} else {
			fmt.Printf("%-35s %-12s %-10d %-10d %-10d\n", fileBase, res.Language, res.Metrics.Complexity, res.Metrics.FunctionLength, res.Metrics.ArgumentCount)
		}
	}

	if blindCount > 0 {
		fmt.Printf("\n>>> NOTICE: %d files are running BLIND (Level 0) with no static analysis configs.\n", blindCount)
		fmt.Printf("    Run 'maintainability-sensors bootstrap' to automatically establish their guardrails!\n")
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
		fmt.Printf("\n=========================================\n")
		fmt.Printf(" Exceptions Created by AI (Relaxed Constraints)\n")
		fmt.Printf("=========================================\n")
		fmt.Printf("⚠️  The following files have relaxed rules configured in their linters:\n\n")
		for _, excStr := range allExceptions {
			fmt.Println(excStr)
		}
		fmt.Printf("\nNOTE: These relaxed thresholds must be manually verified by a human during code review.\n")
		fmt.Printf("(\"Looking at the exceptions AI created was a good point to start my code review.\")\n")
	}
}

func executeRun(opts RunOptions) {
	files, isDir, err := FindFiles(opts.TargetPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if isDir && len(files) == 0 {
		fmt.Println("No supported source files (TS/JS, Python, Go) found in target directory.")
		return
	}

	results, err := ScanFiles(files, isDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		os.Exit(1)
	}

	if isDir && len(results) == 0 {
		fmt.Println("No supported source files (TS/JS, Python, Go) found in target directory.")
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
		scorecard := GenerateMarkdownScorecard(results)
		fmt.Println("Posting scorecard to GitHub PR...")
		if err := PostGitHubPRComment(scorecard); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to post GitHub PR comment: %v\n", err)
		} else {
			fmt.Println("Successfully posted scorecard comment to GitHub PR!")
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
		fmt.Printf("[%s] %s markdown report to %s\n", strings.ToUpper(opts.ActionVerb), opts.ActionVerb, opts.MarkdownOut)
	}
	if opts.JSONOut != "" {
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		if err := os.WriteFile(opts.JSONOut, data, 0644); err != nil {
			return fmt.Errorf("failed to write JSON scorecard: %w", err)
		}
		fmt.Printf("[%s] %s JSON report to %s\n", strings.ToUpper(opts.ActionVerb), opts.ActionVerb, opts.JSONOut)
	}
	if opts.HTMLOut != "" {
		htmlScorecard := GenerateHTMLScorecard(results)
		if err := os.WriteFile(opts.HTMLOut, []byte(htmlScorecard), 0644); err != nil {
			return fmt.Errorf("failed to write HTML scorecard: %w", err)
		}
		fmt.Printf("[%s] %s HTML report to %s\n", strings.ToUpper(opts.ActionVerb), opts.ActionVerb, opts.HTMLOut)
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

	fmt.Printf("\n=========================================\n")
	fmt.Printf(" Maintainability Sensor Result: %s\n", filepath.Base(res.FilePath))
	fmt.Printf("=========================================\n\n")
	fmt.Printf("File Path:  %s\n", res.FilePath)
	fmt.Printf("Language:   %s\n", strings.ToUpper(res.Language))

	if !res.ToolingDetected {
		fmt.Printf("Status:     RUNNING BLIND (Level 0) ⚠️\n")
		fmt.Printf("Message:    %s\n", res.Message)
		return
	}

	fmt.Printf("Status:     ORCHESTRATED (Level 1+) ✅\n\n")
	fmt.Printf("Maintainability Telemetry:\n")
	fmt.Printf("- Max Cyclomatic Complexity:    %d (Limit: %d)\n", res.Metrics.Complexity, sensors.BaselineComplexity)
	fmt.Printf("- Max Function Line Count:      %d (Limit: %d)\n", res.Metrics.FunctionLength, sensors.BaselineFunctionLength)
	fmt.Printf("- Max Function Parameter Count: %d (Limit: %d)\n", res.Metrics.ArgumentCount, sensors.BaselineArgumentCount)

	// Output specific self-correction guidance blocks (Fowler article style)
	printSelfCorrectionGuidance(res)

	// Display Exceptions if any
	if len(res.Exceptions) > 0 {
		fmt.Printf("\n-----------------------------------------\n")
		fmt.Printf(" Exceptions Created by AI (Relaxed Constraints):\n")
		fmt.Printf("-----------------------------------------\n")
		fmt.Printf("⚠️  The following custom limits are set to relaxed values in the configuration:\n\n")
		for _, exc := range res.Exceptions {
			fmt.Printf("  * %s: Configured Limit is %d (Standard Baseline is %d)\n", exc.RuleName, exc.ConfiguredVal, exc.BaselineVal)
		}
		fmt.Printf("\nNOTE: These relaxed thresholds must be manually verified by a human during code review.\n")
		fmt.Printf("(\"Looking at the exceptions AI created was a good point to start my code review.\")\n")
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
		fmt.Printf("\n-----------------------------------------\n")
		fmt.Printf(" AI Agent Self-Correction Prompts:\n")
		fmt.Printf("-----------------------------------------\n")
		fmt.Printf("Pass the following instructions directly to your AI Coding Agent (Cursor/Claude) to refactor the violations:\n\n")
		for _, g := range guidance {
			fmt.Println(g)
		}
		fmt.Printf("\nIf refactoring is impossible, instruct the agent to suppress the warning with standard inline annotations (e.g. // eslint-disable-next-line -- reason) to keep the baseline clean.\n")
	}
}
