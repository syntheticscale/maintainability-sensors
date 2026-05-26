package sensors

import "github.com/syntheticscale/maintainability-sensors/internal/plugin/protocol"

const (
	BaselineComplexity          = protocol.BaselineComplexity
	BaselineCognitiveComplexity = protocol.BaselineCognitiveComplexity
	BaselineFunctionLength      = protocol.BaselineFunctionLength
	BaselineArgumentCount       = protocol.BaselineArgumentCount
	BaselineFileLength          = protocol.BaselineFileLength
	BaselineCaseLength          = protocol.BaselineCaseLength
)

const (
	MaxFileSize           = 2 * 1024 * 1024
	MaxJSONFileSize       = 10 * 1024 * 1024
	FallbackLimit         = 999999
	UntrackedFileEndLine  = 999999999
	PluginChunkSize       = 300
	FallbackEndLineOffset = protocol.FallbackEndLineOffset
)

const (
	RuleComplexity          = protocol.RuleComplexity
	RuleCognitiveComplexity = protocol.RuleCognitiveComplexity
	RuleFunctionLength      = protocol.RuleFunctionLength
	RuleArgumentCount       = protocol.RuleArgumentCount
	RuleCaseBlockLength     = protocol.RuleCaseBlockLength
	RuleFileLength          = protocol.RuleFileLength
)
