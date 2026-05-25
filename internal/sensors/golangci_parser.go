package sensors

// GoConfigParser extracts thresholds from golangci-lint configs.
type GoConfigParser struct{}

func (GoConfigParser) Name() string { return "golangci" }

func (p GoConfigParser) Anchors() []string {
	return []string{".golangci.yml", "golangci.yml"}
}

func (p GoConfigParser) Rules() []ParserRule {
	return []ParserRule{
		{RuleName: RuleComplexity, Keys: []string{"min-complexity"}, Baseline: BaselineComplexity},
		{RuleName: RuleFunctionLength, Keys: []string{"lines"}, Baseline: BaselineFunctionLength},
		{RuleName: RuleArgumentCount, Keys: []string{"argument-limit"}, Baseline: BaselineArgumentCount}, // from revive
	}
}
