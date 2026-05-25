package sensors

// RuboCopConfigParser extracts thresholds from RuboCop configs.
type RuboCopConfigParser struct{}

func (RuboCopConfigParser) Name() string { return "rubocop" }

func (p RuboCopConfigParser) Anchors() []string {
	return []string{".rubocop.yml", "Gemfile"}
}

func (p RuboCopConfigParser) Rules() []ParserRule {
	return []ParserRule{
		{RuleName: RuleComplexity, Keys: []string{"CyclomaticComplexity"}, Baseline: BaselineComplexity},
		{RuleName: RuleFunctionLength, Keys: []string{"MethodLength"}, Baseline: BaselineFunctionLength},
		{RuleName: RuleArgumentCount, Keys: []string{"ParameterLists"}, Baseline: BaselineArgumentCount},
		{RuleName: RuleFileLength, Keys: []string{"ModuleLength"}, Baseline: BaselineFileLength},
	}
}
