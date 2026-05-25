package cli

import (
	"fmt"
	"os"

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
	return executeRun(RunOptions{
		TargetPath:  c.Path,
		JSONOutput:  c.Json,
		GithubPR:    c.GithubPr,
		MarkdownOut: c.MarkdownOut,
		JSONOutFile: c.JsonOut,
		HTMLOut:     c.HtmlOut,
	})
}

type generateCmd struct {
	JsonIn      string `arg:"" help:"Input JSON file path."`
	MarkdownOut string `help:"Write beautiful markdown scorecard to specified file path."`
	HtmlOut     string `help:"Write beautiful dark-themed HTML scorecard to specified file path."`
}

func (c *generateCmd) Run() error {
	return executeGenerate(c.JsonIn, c.MarkdownOut, c.HtmlOut)
}

type bootstrapCmd struct {
	Path           string `arg:"" optional:"" default:"." help:"Target path to bootstrap."`
	WithWarnPolicy bool   `optional:"" name:"with-warn-policy" help:"Generate a .maintainability-sensors.yml with default-severity: warn."`
}

func (c *bootstrapCmd) Run() error {
	return executeBootstrap(c.Path, c.WithWarnPolicy)
}

type CheckDiffCmd struct {
	TargetBranch    string   `optional:"" default:"HEAD" help:"Target branch to diff against."`
	TargetPath      string   `arg:"" optional:"" default:"." help:"Target path to diff."`
	Config          string   `optional:"" help:"Path to .maintainability-sensors.yml config file."`
	DefaultSeverity string   `optional:"" help:"Default severity level for rules not explicitly configured (error|warn|ignore). Defaults to error."`
	Severity        []string `optional:"" name:"severity" help:"Per-rule severity overrides (format: Rule:level)."`
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
