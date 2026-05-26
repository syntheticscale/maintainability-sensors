package cli

//nolint // maintainability: highly cohesive test

import (
	"path/filepath"
	"testing"

	"github.com/syntheticscale/maintainability-sensors/internal/sensors"
)

func intPtr(i int) *int {
	return &i
}

func TestProcessViolationsMap(t *testing.T) {
	absPath1, _ := filepath.Abs("file1.go")
	absPath2, _ := filepath.Abs("file2.py")

	policy := &CheckDiffPolicy{
		DefaultSeverity: SeverityError,
		Rules: map[string]RulePolicy{
			sensors.RuleComplexity:     {Name: sensors.RuleComplexity, Threshold: intPtr(10), Severity: SeverityWarn},
			sensors.RuleFunctionLength: {Name: sensors.RuleFunctionLength, Threshold: intPtr(50), Severity: SeverityError},
			sensors.RuleArgumentCount:  {Name: sensors.RuleArgumentCount, Threshold: intPtr(5), Severity: SeverityError},
		},
	}

	violationsMap := map[string][]sensors.Violation{
		"file1.go": {
			{RuleName: sensors.RuleComplexity, Value: 15, StartLine: 10, EndLine: 20, Message: "too complex"},
			{RuleName: sensors.RuleFunctionLength, Value: 60, StartLine: 30, EndLine: 100, Message: "too long"},
			{RuleName: sensors.RuleComplexity, Value: 12, StartLine: 150, EndLine: 160, Message: "too complex but not overlapping"},
		},
		"file2.py": {
			{RuleName: sensors.RuleArgumentCount, Value: 6, StartLine: 5, EndLine: 5, Message: "too many args"},
		},
	}

	absModifiedLines := map[string][]sensors.LineRange{
		absPath1: {
			{Start: 1, End: 50},
		},
		absPath2: {
			{Start: 1, End: 10},
		},
	}

	hasErrors, warnings := processViolationsMap(violationsMap, absModifiedLines, policy)

	if !hasErrors {
		t.Errorf("processViolationsMap() expected hasErrors = true due to RuleFunctionLength")
	}
	
	if len(warnings) != 1 {
		t.Errorf("processViolationsMap() expected 1 warning, got %d", len(warnings))
	} else {
		expectedWarning := "REFACTORING PROMPT: file1.go:10 - " + sensors.RuleComplexity + " - too complex"
		if warnings[0] != expectedWarning {
			t.Errorf("processViolationsMap() warning = %q, want %q", warnings[0], expectedWarning)
		}
	}
}
