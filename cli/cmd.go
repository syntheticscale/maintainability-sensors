package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/paulolai/maintainability-sensors/sensors"
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
		_ = runCmd.Parse(os.Args[2:])

		targetPath := "."
		if len(runCmd.Args()) > 0 {
			targetPath = runCmd.Arg(0)
		}

		executeRun(targetPath, *jsonOut, *githubPR, *markdownOut)

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
	fmt.Printf("                    Optional flag: --json (outputs raw JSON metric payload).\n")
	fmt.Printf("                    Optional flag: --github-pr (post markdown scorecard directly as a GitHub PR comment).\n")
	fmt.Printf("                    Optional flag: --markdown-out [file-path] (writes beautiful markdown scorecard to specified file path).\n")
	fmt.Printf("  bootstrap [path]  Auto-detect repository language and bootstrap pristine, non-overwriting\n")
	fmt.Printf("                    maintainability configuration files (TS, Python, Go, Java).\n\n")
	fmt.Printf("Examples:\n")
	fmt.Printf("  maintainability-sensors run .\n")
	fmt.Printf("  maintainability-sensors run . --markdown-out=report.md\n")
	fmt.Printf("  maintainability-sensors run src/api.py --json\n")
	fmt.Printf("  maintainability-sensors bootstrap /path/to/my/project\n")
}

func executeRun(targetPath string, jsonOutput bool, githubPR bool, markdownOut string) {
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		absPath = targetPath
	}

	info, err := os.Stat(absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Path does not exist: %s\n", targetPath)
		os.Exit(1)
	}

	var results []sensors.OrchestratorResult

	if !info.IsDir() {
		// Single File Scan
		res, err := sensors.OrchestratedScan(absPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Scan failed: %v\n", err)
			os.Exit(1)
		}
		results = append(results, res)
		printScanResult(res, jsonOutput)
	} else {
		// Directory Scan
		err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			// Skip typical build / dependency folders
			if strings.Contains(path, "node_modules") || strings.Contains(path, ".git") || strings.Contains(path, "vendor") || strings.Contains(path, "bin") {
				return nil
			}
			// Skip files without recognized extension
			ext := filepath.Ext(path)
			if ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" && ext != ".py" && ext != ".go" {
				return nil
			}

			res, err := sensors.OrchestratedScan(path)
			if err == nil {
				results = append(results, res)
			}
			return nil
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Directory scan failed: %v\n", err)
			os.Exit(1)
		}

		if len(results) == 0 {
			fmt.Println("No supported source files (TS/JS, Python, Go) found in target directory.")
			return
		}

		if jsonOutput {
			data, _ := json.MarshalIndent(results, "", "  ")
			fmt.Println(string(data))
		} else {
			// Pretty print summary table
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

			// Display directory scan exceptions!
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
	}

	// Post results/summary to GitHub if active/triggered
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		scorecard := GenerateMarkdownScorecard(results)
		if err := WriteGitHubStepSummary(scorecard); err != nil {
			fmt.Fprintf(os.Stderr, "[WARNING] Failed to write GitHub Step Summary: %v\n", err)
		}
	}

	isCI_PR := os.Getenv("GITHUB_TOKEN") != "" && (os.Getenv("GITHUB_EVENT_PATH") != "" || os.Getenv("GITHUB_REF") != "")
	if githubPR || isCI_PR {
		scorecard := GenerateMarkdownScorecard(results)
		fmt.Println("Posting scorecard to GitHub PR...")
		if err := PostGitHubPRComment(scorecard); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to post GitHub PR comment: %v\n", err)
		} else {
			fmt.Println("Successfully posted scorecard comment to GitHub PR!")
		}
	}

	if markdownOut != "" {
		scorecard := GenerateMarkdownScorecard(results)
		err := os.WriteFile(markdownOut, []byte(scorecard), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to write markdown scorecard: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[SUCCESS] Saved markdown report to %s\n", markdownOut)
	}
}

func executeBootstrap(targetPath string) {
	err := sensors.BootstrapRepo(targetPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Bootstrap failed: %v\n", err)
		os.Exit(1)
	}
}

func printScanResult(res sensors.OrchestratorResult, jsonOutput bool) {
	if jsonOutput {
		data, _ := json.MarshalIndent(res, "", "  ")
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
	fmt.Printf("- Max Cyclomatic Complexity:    %d (Limit: 8)\n", res.Metrics.Complexity)
	fmt.Printf("- Max Function Line Count:      %d (Limit: 50)\n", res.Metrics.FunctionLength)
	fmt.Printf("- Max Function Parameter Count: %d (Limit: 4)\n", res.Metrics.ArgumentCount)

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

	if res.Metrics.Complexity > 8 {
		hasViolation = true
		guidance = append(guidance, fmt.Sprintf("  * Complexity is %d (Max 8). Nudge coding agent to extract nested conditionals into separate, single-responsibility helper functions.", res.Metrics.Complexity))
	}
	if res.Metrics.FunctionLength > 50 {
		hasViolation = true
		guidance = append(guidance, fmt.Sprintf("  * Function lines is %d (Max 50). Nudge coding agent to modularize this block into separate functional components.", res.Metrics.FunctionLength))
	}
	if res.Metrics.ArgumentCount > 4 {
		hasViolation = true
		guidance = append(guidance, fmt.Sprintf("  * Parameter count is %d (Max 4). Nudge coding agent to bundle parameters into a single structured configuration object.", res.Metrics.ArgumentCount))
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
