package sensors

const (
	BaselineComplexity          = 8
	BaselineCognitiveComplexity = 8
	BaselineFunctionLength      = 50
	BaselineArgumentCount       = 4
	BaselineFileLength          = 300 // Not enforced for Go native AST parsing; Go metrics are function-oriented.
	BaselineCaseLength          = 10
)

// Canonical rule names used for exceptions and metrics.
const (
	RuleComplexity          = "Complexity"
	RuleCognitiveComplexity = "CognitiveComplexity"
	RuleFunctionLength      = "FunctionLength"
	RuleArgumentCount       = "ArgumentCount"
	RuleCaseBlockLength     = "CaseBlockLength"
	RuleFileLength          = "FileLength"
)
