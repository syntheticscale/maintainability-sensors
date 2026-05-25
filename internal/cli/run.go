package cli

import (
	"fmt"
	"os"
	"sync"

	"github.com/syntheticscale/maintainability-sensors/internal/sensors"
	"golang.org/x/sync/errgroup"
)

type RunOptions struct {
	TargetPath  string
	JSONOutput  bool
	GithubPR    bool
	MarkdownOut string
	JSONOutFile string
	HTMLOut     string
}

func executeRun(opts RunOptions) {
	files, isDir, err := FindFiles(opts.TargetPath)
	if err != nil {
		logLn(LogLevelError, err)
		os.Exit(1)
	}

	if isDir && len(files) == 0 {
	logLn(LogLevelWarn, "No supported source files (TS/JS, Python, Go) found in target directory.")
	return
	}

	results, err := ScanFiles(files, isDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		os.Exit(1)
	}

	if isDir && len(results) == 0 {
		logLn(LogLevelWarn, "No supported source files (TS/JS, Python, Go) found in target directory.")
		return
	}

	hasV := FormatResultsCLI(results, opts.JSONOutput, isDir)

	postGitHubResults(results, opts.GithubPR)

	saveReportsAndExit(results, opts, hasV)
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

			fileContexts := make([]sensors.FileContext, len(files))
			for i, f := range files {
				fileContexts[i] = sensors.FileContext{Path: f}
			}

			res, err := sensors.OrchestratedScanBatch(fileContexts, lang)
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

func saveReportsAndExit(results []sensors.OrchestratorResult, opts RunOptions, hasV bool) {
	err := writeReports(results, ReportOptions{
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

func postGitHubResults(results []sensors.OrchestratorResult, forcePR bool) {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		scorecard := GenerateMarkdownScorecard(results)
		if err := WriteGitHubStepSummary(scorecard); err != nil {
			fmt.Fprintf(os.Stderr, "[WARNING] Failed to write GitHub Step Summary: %v\n", err)
		}
	}

	isCI_PR := os.Getenv("GITHUB_TOKEN") != "" && (os.Getenv("GITHUB_EVENT_PATH") != "" || os.Getenv("GITHUB_REF") != "")
	if forcePR || isCI_PR {
		logLn(LogLevelInfo, "Posting inline review to GitHub PR...")
		if err := PostGitHubReview(results); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to post GitHub inline review: %v\n", err)
		} else {
			logLn(LogLevelInfo, "Successfully posted inline review to GitHub PR!")
		}
	}
}
