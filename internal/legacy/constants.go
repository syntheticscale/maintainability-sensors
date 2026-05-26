package legacy

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
