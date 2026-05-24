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
	BaseName       string
	FilePath       string
	Language       string
	IsOrchestrated bool
	Complexity     int
	FunctionLength int
	ArgumentCount  int
	CompClass      string
	LinesClass     string
	ParamsClass    string
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

	fileBase := filepath.Base(res.FilePath)
	filePrompts := getHTMLFilePrompts(data, res)

	compClass, linesClass, paramsClass := getCSSClasses(res)

	data.Rows = append(data.Rows, FileRow{
		BaseName:       fileBase,
		FilePath:       res.FilePath,
		Language:       strings.ToUpper(res.Language),
		IsOrchestrated: true,
		Complexity:     res.Metrics.Complexity,
		FunctionLength: res.Metrics.FunctionLength,
		ArgumentCount:  res.Metrics.ArgumentCount,
		CompClass:      compClass,
		LinesClass:     linesClass,
		ParamsClass:    paramsClass,
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

func getHTMLFilePrompts(data *ReportData, res sensors.OrchestratorResult) []string {
	var filePrompts []string
	if res.Metrics.Complexity > sensors.BaselineComplexity {
		data.TotalViolations++
		filePrompts = append(filePrompts, fmt.Sprintf("Complexity is %d (Max %d limit). Nudge agent to extract nested conditionals into separate helper functions.", res.Metrics.Complexity, sensors.BaselineComplexity))
	}
	if res.Metrics.FunctionLength > sensors.BaselineFunctionLength {
		if len(filePrompts) == 0 {
			data.TotalViolations++
		}
		filePrompts = append(filePrompts, fmt.Sprintf("Function lines is %d (Max %d limit). Nudge agent to modularize this block into separate functional components.", res.Metrics.FunctionLength, sensors.BaselineFunctionLength))
	}
	if res.Metrics.ArgumentCount > sensors.BaselineArgumentCount {
		if len(filePrompts) == 0 {
			data.TotalViolations++
		}
		filePrompts = append(filePrompts, fmt.Sprintf("Parameter count is %d (Max %d limit). Nudge agent to bundle parameters into a structured configuration object.", res.Metrics.ArgumentCount, sensors.BaselineArgumentCount))
	}
	return filePrompts
}

func getCSSClasses(res sensors.OrchestratorResult) (string, string, string) {
	compClass := ""
	if res.Metrics.Complexity > sensors.BaselineComplexity {
		compClass = "text-error font-bold"
	}
	linesClass := ""
	if res.Metrics.FunctionLength > sensors.BaselineFunctionLength {
		linesClass = "text-error font-bold"
	}
	paramsClass := ""
	if res.Metrics.ArgumentCount > sensors.BaselineArgumentCount {
		paramsClass = "text-error font-bold"
	}
	return compClass, linesClass, paramsClass
}
