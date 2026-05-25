package cli

//nolint // maintainability: highly cohesive logic

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

func executeRun(opts RunOptions) error {
	files, isDir, err := FindFiles(opts.TargetPath)
	if err != nil {
		logLn(LogLevelError, err)
		return fmt.Errorf("failed to find files: %v", err)
	}

	if isDir && len(files) == 0 {
		logLn(LogLevelWarn, "No supported source files (TS/JS, Python, Go) found in target directory.")
		return nil
	}

	results, err := ScanFiles(files, isDir)
	if err != nil {
		logf(LogLevelError, "[ERROR] %v\n", err)
		return fmt.Errorf("scan failed: %v", err)
	}

	if isDir && len(results) == 0 {
		logLn(LogLevelWarn, "No supported source files (TS/JS, Python, Go) found in target directory.")
		return nil
	}

	hasV := FormatResultsCLI(results, opts.JSONOutput, isDir)

	postGitHubResults(results, opts.GithubPR)

	return saveReportsAndExit(results, opts, hasV)
}

func groupFiles(filePaths []string) map[string][]string {
	groups := make(map[string][]string)
	for _, p := range filePaths {
		lang := sensors.DetectLanguage(p)
		groups[lang] = append(groups[lang], p)
	}
	return groups
}

func handleUnknownLanguage(files []string, isDir bool) error {
	for _, f := range files {
		if !isDir {
			return fmt.Errorf("unsupported or unrecognized language file: %s", f)
		}
		logf(LogLevelWarn, "[WARNING] Scan failed for %s: unsupported or unrecognized language file: %s\n", f, f)
	}
	return nil
}

func processLanguageGroup(lang string, files []string, isDir bool) ([]sensors.OrchestratorResult, error) {
	if lang == "" {
		return nil, handleUnknownLanguage(files, isDir)
	}

	fileContexts := make([]sensors.FileContext, len(files))
	for i, f := range files {
		fileContexts[i] = sensors.FileContext{Path: f}
	}

	res, err := sensors.OrchestratedScanBatch(fileContexts, lang)
	if err != nil {
		if !isDir {
			return nil, fmt.Errorf("Scan failed: %v", err)
		}
		logf(LogLevelWarn, "[WARNING] Scan failed for language %s: %v\n", lang, err)
		return nil, nil
	}
	return res, nil
}

func ScanFiles(filePaths []string, isDir bool) ([]sensors.OrchestratorResult, error) {
	groups := groupFiles(filePaths)

	var allResults []sensors.OrchestratorResult
	var mu sync.Mutex
	var g errgroup.Group

	for lang, files := range groups {
		lang, files := lang, files
		g.Go(func() error {
			res, err := processLanguageGroup(lang, files, isDir)
			if err != nil {
				return err
			}
			if len(res) > 0 {
				mu.Lock()
				allResults = append(allResults, res...)
				mu.Unlock()
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return allResults, nil
}

func saveReportsAndExit(results []sensors.OrchestratorResult, opts RunOptions, hasV bool) error {
	err := writeReports(results, ReportOptions{
		MarkdownOut: opts.MarkdownOut,
		JSONOut:     opts.JSONOutFile,
		HTMLOut:     opts.HTMLOut,
		ActionVerb:  "Saved",
	})
	if err != nil {
		logf(LogLevelError, "[ERROR] %v\n", err)
		return fmt.Errorf("failed to save reports: %v", err)
	}

	if hasV {
		return fmt.Errorf("maintainability violations detected")
	}

	return nil
}

func postGitHubStepSummary(results []sensors.OrchestratorResult) {
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		return
	}
	scorecard := GenerateMarkdownScorecard(results)
	if err := WriteGitHubStepSummary(scorecard); err != nil {
		logf(LogLevelWarn, "[WARNING] Failed to write GitHub Step Summary: %v\n", err)
	}
}

func isCIAndPR() bool {
	if os.Getenv("GITHUB_TOKEN") == "" {
		return false
	}
	if os.Getenv("GITHUB_EVENT_PATH") != "" {
		return true
	}
	if os.Getenv("GITHUB_REF") != "" {
		return true
	}
	return false
}

func postGitHubResults(results []sensors.OrchestratorResult, forcePR bool) {
	postGitHubStepSummary(results)

	if !forcePR && !isCIAndPR() {
		return
	}

	logLn(LogLevelInfo, "Posting inline review to GitHub PR...")
	if err := PostGitHubPRComment(results); err != nil {		logf(LogLevelError, "[ERROR] Failed to post GitHub inline review: %v\n", err)
	} else {
		logLn(LogLevelInfo, "Successfully posted inline review to GitHub PR!")
	}
}
