package sensors

// StandardRBConfigParser extracts thresholds from StandardRB configs.
type StandardRBConfigParser struct{}

func (StandardRBConfigParser) Name() string { return "standardrb" }

func (p StandardRBConfigParser) Anchors() []string {
	return []string{".standard.yml"}
}

func (p StandardRBConfigParser) Rules() []ParserRule {
	return []ParserRule{
		{RuleName: RuleComplexity, Keys: []string{"CyclomaticComplexity"}, Baseline: BaselineComplexity},
		{RuleName: RuleFunctionLength, Keys: []string{"MethodLength"}, Baseline: BaselineFunctionLength},
		{RuleName: RuleArgumentCount, Keys: []string{"ParameterLists"}, Baseline: BaselineArgumentCount},
		{RuleName: RuleFileLength, Keys: []string{"ModuleLength"}, Baseline: BaselineFileLength},
	}
}
