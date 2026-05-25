package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/syntheticscale/maintainability-sensors/internal/sensors"
)

func executeGenerate(jsonIn string, markdownOut string, htmlOut string) {
	results, err := parseJSONScorecard(jsonIn)
	if err != nil {
		logf(LogLevelError, "[ERROR] %v\n", err)
		os.Exit(1)
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
		logf(LogLevelError, "[ERROR] %v\n", err)
		os.Exit(1)
	}

	if hasV {
		os.Exit(1)
	}
}

func validateScorecardResults(results []sensors.OrchestratorResult) error {
	for i, res := range results {
		if res.FilePath == "" {
			return fmt.Errorf("Validation failed: Missing 'file_path' in result at index %d", i)
		}
		if res.Language == "" {
			return fmt.Errorf("Validation failed: Missing 'language' in result at index %d", i)
		}
	}
	return nil
}

func parseJSONScorecard(jsonIn string) ([]sensors.OrchestratorResult, error) {
	if info, err := os.Stat(jsonIn); err == nil && (!info.Mode().IsRegular() || info.Size() > sensors.MaxJSONFileSize) {
		return nil, fmt.Errorf("JSON input file is too large or not a regular file (limit 10MB)")
	}
	data, err := os.ReadFile(jsonIn)
	if err != nil {
		return nil, fmt.Errorf("Failed to read JSON input file: %v", err)
	}

	var results []sensors.OrchestratorResult
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, fmt.Errorf("Failed to parse JSON input scorecard: %v", err)
	}

	if err := validateScorecardResults(results); err != nil {
		return nil, err
	}
	return results, nil
}
