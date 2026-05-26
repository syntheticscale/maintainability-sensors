package cli

import (
	_ "embed"
	"fmt"
	"html/template"
	"path/filepath"
	"strings"

	"github.com/syntheticscale/maintainability-sensors/internal/sensors"
)

//go:embed templates/report.html
var reportHTML string

var reportTemplate = template.Must(template.New("report").Parse(reportHTML))

// ReportData holds pre-computed data for the HTML report template.
type ReportData struct {
	TotalFiles        int
	OrchestratedCount int
	BlindCount        int
	TotalViolations   int
	TotalExceptions   int
	HasViolations     bool
	HasExceptions     bool
	Rows              []FileRow
	FilePrompts       []FilePromptData
	FileExceptions    []FileExceptionData
}

// FileRow represents a single row in the scorecard table.
type FileRow struct {
	BaseName            string
	FilePath            string
	Language            string
	IsOrchestrated      bool
	Complexity          int
	CognitiveComplexity int
	FunctionLength      int
	ArgumentCount       int
	MaxCaseLength       int
	CompClass           string
	CogClass            string
	LinesClass          string
	ParamsClass         string
	CaseClass           string
}

// FilePromptData holds prompts for a file with violations.
type FilePromptData struct {
	BaseName   string
	Prompts    []string
	FullPrompt string
}

// FileExceptionData holds exceptions for a file.
type FileExceptionData struct {
	BaseName   string
	Exceptions []sensors.RelaxedLimit
}

// GenerateHTMLScorecard compiles a self-contained, beautifully styled, dark-themed HTML report.
func GenerateHTMLScorecard(results []sensors.OrchestratorResult) string {
	data := ReportData{
		TotalFiles: len(results),
	}

	for _, res := range results {
		if res.ToolingDetected {
			processOrchestratedResult(&data, res)
		} else {
			processBlindResult(&data, res)
		}
	}

	data.HasViolations = data.TotalViolations > 0
	data.HasExceptions = data.TotalExceptions > 0

	var buf strings.Builder
	if err := reportTemplate.Execute(&buf, data); err != nil {
		return fmt.Sprintf("<!-- Template execution error: %v -->", err)
	}

	return buf.String()
}

func processBlindResult(data *ReportData, res sensors.OrchestratorResult) {
	data.BlindCount++
	data.Rows = append(data.Rows, FileRow{
		BaseName:       filepath.Base(res.FilePath),
		FilePath:       res.FilePath,
		Language:       strings.ToUpper(res.Language),
		IsOrchestrated: false,
	})
}

func processOrchestratedResult(data *ReportData, res sensors.OrchestratorResult) {
	data.OrchestratedCount++
	data.TotalExceptions += len(res.Exceptions)

	summary := sensors.Evaluate(res)
	fileBase := filepath.Base(res.FilePath)
	filePrompts := getHTMLFilePrompts(data, summary)

	compClass, cogClass, linesClass, paramsClass, caseClass := getCSSClasses(summary)

	data.Rows = append(data.Rows, FileRow{
		BaseName:            fileBase,
		FilePath:            res.FilePath,
		Language:            strings.ToUpper(res.Language),
		IsOrchestrated:      true,
		Complexity:          res.Metrics.Complexity,
		CognitiveComplexity: res.Metrics.CognitiveComplexity,
		FunctionLength:      res.Metrics.FunctionLength,
		ArgumentCount:       res.Metrics.ArgumentCount,
		MaxCaseLength:       res.Metrics.MaxCaseLength,
		CompClass:           compClass,
		CogClass:            cogClass,
		LinesClass:          linesClass,
		ParamsClass:         paramsClass,
		CaseClass:           caseClass,
	})

	if len(filePrompts) > 0 {
		data.FilePrompts = append(data.FilePrompts, FilePromptData{
			BaseName:   fileBase,
			Prompts:    filePrompts,
			FullPrompt: fmt.Sprintf("Refactor %s. Violations:\n%s", fileBase, strings.Join(filePrompts, "\n")),
		})
	}

	if len(res.Exceptions) > 0 {
		data.FileExceptions = append(data.FileExceptions, FileExceptionData{
			BaseName:   fileBase,
			Exceptions: res.Exceptions,
		})
	}
}

func getHTMLFilePrompts(data *ReportData, summary sensors.EvaluatedSummary) []string {
	var filePrompts []string
	for _, v := range summary.Violations {
		data.TotalViolations++
		filePrompts = append(filePrompts, v.Guidance)
	}
	return filePrompts
}

func getCSSClasses(summary sensors.EvaluatedSummary) (string, string, string, string, string) {
	compClass := ""
	cogClass := ""
	linesClass := ""
	paramsClass := ""
	caseClass := ""

	for _, v := range summary.Violations {
		switch v.RuleName {
		case sensors.RuleComplexity:
			compClass = "text-error font-bold"
		case sensors.RuleCognitiveComplexity:
			cogClass = "text-error font-bold"
		case sensors.RuleFunctionLength:
			linesClass = "text-error font-bold"
		case sensors.RuleArgumentCount:
			paramsClass = "text-error font-bold"
		case sensors.RuleCaseBlockLength:
			caseClass = "text-error font-bold"
		}
	}

	return compClass, cogClass, linesClass, paramsClass, caseClass
}
