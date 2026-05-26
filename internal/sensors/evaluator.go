package sensors

import (
	"fmt"
)

// EvaluatedViolation represents a single detected violation with standard guidance.
type EvaluatedViolation struct {
	RuleName     string `json:"rule_name"`
	ActualValue  int    `json:"actual_value"`
	ThresholdVal int    `json:"threshold_val"`
	Guidance     string `json:"guidance"`
}

// EvaluatedSummary holds the result along with computed violations.
type EvaluatedSummary struct {
	Result        OrchestratorResult   `json:"result"`
	Violations    []EvaluatedViolation `json:"violations"`
	HasViolations bool                 `json:"has_violations"`
}

// EffectiveLimits represents the resolved threshold limits after checking exceptions.
type EffectiveLimits struct {
	Complexity          int
	CognitiveComplexity int
	FunctionLength      int
	ArgumentCount       int
	MaxCaseLength       int
}

// GetEffectiveLimits determines the final limits applied to the file.
func GetEffectiveLimits(res OrchestratorResult) EffectiveLimits {
	limits := EffectiveLimits{
		Complexity:          BaselineComplexity,
		CognitiveComplexity: BaselineCognitiveComplexity,
		FunctionLength:      BaselineFunctionLength,
		ArgumentCount:       BaselineArgumentCount,
		MaxCaseLength:       BaselineCaseLength,
	}
	for _, exc := range res.Exceptions {
		switch exc.RuleName {
		case RuleComplexity:
			limits.Complexity = exc.ConfiguredVal
		case RuleCognitiveComplexity:
			limits.CognitiveComplexity = exc.ConfiguredVal
		case RuleFunctionLength:
			limits.FunctionLength = exc.ConfiguredVal
		case RuleArgumentCount:
			limits.ArgumentCount = exc.ConfiguredVal
		case RuleCaseBlockLength:
			limits.MaxCaseLength = exc.ConfiguredVal
		}
	}
	return limits
}

// Evaluate processes an OrchestratorResult and returns an EvaluatedSummary with violations.
func Evaluate(res OrchestratorResult) EvaluatedSummary {
	summary := EvaluatedSummary{
		Result: res,
	}
	if !res.ToolingDetected {
		return summary
	}

	limits := GetEffectiveLimits(res)

	summary.Violations = append(summary.Violations, checkComplexity(res.Metrics, limits)...)
	summary.Violations = append(summary.Violations, checkCognitiveComplexity(res.Metrics, limits)...)
	summary.Violations = append(summary.Violations, checkFunctionLength(res.Metrics, limits)...)
	summary.Violations = append(summary.Violations, checkArgumentCount(res.Metrics, limits)...)
	summary.Violations = append(summary.Violations, checkMaxCaseLength(res.Metrics, limits)...)

	summary.HasViolations = len(summary.Violations) > 0
	return summary
}

func checkComplexity(metrics MaintainabilityMetrics, limits EffectiveLimits) []EvaluatedViolation {
	if metrics.Complexity > limits.Complexity {
		return []EvaluatedViolation{{
			RuleName:     RuleComplexity,
			ActualValue:  metrics.Complexity,
			ThresholdVal: limits.Complexity,
			Guidance:     fmt.Sprintf("Complexity is %d (Max %d). Extract nested conditionals into separate, single-responsibility helper functions.", metrics.Complexity, limits.Complexity),
		}}
	}
	return nil
}

func checkCognitiveComplexity(metrics MaintainabilityMetrics, limits EffectiveLimits) []EvaluatedViolation {
	if metrics.CognitiveComplexity > limits.CognitiveComplexity {
		return []EvaluatedViolation{{
			RuleName:     RuleCognitiveComplexity,
			ActualValue:  metrics.CognitiveComplexity,
			ThresholdVal: limits.CognitiveComplexity,
			Guidance:     fmt.Sprintf("Cognitive Complexity is %d (Max %d). Flatten deeply nested control flow and return early.", metrics.CognitiveComplexity, limits.CognitiveComplexity),
		}}
	}
	return nil
}

func checkFunctionLength(metrics MaintainabilityMetrics, limits EffectiveLimits) []EvaluatedViolation {
	if metrics.FunctionLength > limits.FunctionLength {
		return []EvaluatedViolation{{
			RuleName:     RuleFunctionLength,
			ActualValue:  metrics.FunctionLength,
			ThresholdVal: limits.FunctionLength,
			Guidance:     fmt.Sprintf("Function lines is %d (Max %d). Modularize this block into separate functional components.", metrics.FunctionLength, limits.FunctionLength),
		}}
	}
	return nil
}

func checkArgumentCount(metrics MaintainabilityMetrics, limits EffectiveLimits) []EvaluatedViolation {
	if metrics.ArgumentCount > limits.ArgumentCount {
		return []EvaluatedViolation{{
			RuleName:     RuleArgumentCount,
			ActualValue:  metrics.ArgumentCount,
			ThresholdVal: limits.ArgumentCount,
			Guidance:     fmt.Sprintf("Parameter count is %d (Max %d). Bundle parameters into a single structured configuration object.", metrics.ArgumentCount, limits.ArgumentCount),
		}}
	}
	return nil
}

func checkMaxCaseLength(metrics MaintainabilityMetrics, limits EffectiveLimits) []EvaluatedViolation {
	if metrics.MaxCaseLength > limits.MaxCaseLength {
		return []EvaluatedViolation{{
			RuleName:     RuleCaseBlockLength,
			ActualValue:  metrics.MaxCaseLength,
			ThresholdVal: limits.MaxCaseLength,
			Guidance:     fmt.Sprintf("Case block lines is %d (Max %d). Extract the case logic into a well-named method.", metrics.MaxCaseLength, limits.MaxCaseLength),
		}}
	}
	return nil
}

// EvaluateAll processes multiple OrchestratorResults.
func EvaluateAll(results []OrchestratorResult) []EvaluatedSummary {
	var summaries []EvaluatedSummary
	for _, res := range results {
		summaries = append(summaries, Evaluate(res))
	}
	return summaries
}
