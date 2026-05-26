package sensors

const (
	BaselineComplexity          = 8
	BaselineCognitiveComplexity = 8
	BaselineFunctionLength      = 50
	BaselineArgumentCount       = 4
	BaselineFileLength          = 300 // Not enforced for Go native AST parsing; Go metrics are function-oriented.
	BaselineCaseLength          = 10
)

const (
	MaxFileSize           = 2 * 1024 * 1024  // 2MB file size limit for scanning
	MaxJSONFileSize       = 10 * 1024 * 1024 // 10MB JSON file size limit
	FallbackLimit         = 999999           // Fallback limit when no rule is matched
	UntrackedFileEndLine  = 999999999        // Sentinel end line for untracked files
	PluginChunkSize       = 300              // Number of files per plugin invocation chunk
	FallbackEndLineOffset = 100              // Offset added to msg.Line when EndLine is zero
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
