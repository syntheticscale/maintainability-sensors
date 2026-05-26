package sensors

import (
	"testing"
)

func TestEvaluate(t *testing.T) {
	res := OrchestratorResult{
		ToolingDetected: true,
		Metrics: MaintainabilityMetrics{
			Complexity:          BaselineComplexity + 1,
			CognitiveComplexity: BaselineCognitiveComplexity - 1,
		},
	}

	summary := Evaluate(res)

	if !summary.HasViolations {
		t.Error("expected hasViolations to be true")
	}

	if len(summary.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(summary.Violations))
	}

	v := summary.Violations[0]
	if v.RuleName != RuleComplexity {
		t.Errorf("expected rule %s, got %s", RuleComplexity, v.RuleName)
	}
	if v.ActualValue != BaselineComplexity+1 {
		t.Errorf("expected actual %d, got %d", BaselineComplexity+1, v.ActualValue)
	}
	if v.ThresholdVal != BaselineComplexity {
		t.Errorf("expected threshold %d, got %d", BaselineComplexity, v.ThresholdVal)
	}
	if v.Guidance == "" {
		t.Error("expected guidance string, got empty")
	}
}
