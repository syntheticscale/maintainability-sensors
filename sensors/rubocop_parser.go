package sensors

// RuboCopConfigParser extracts thresholds from RuboCop configs.
type RuboCopConfigParser struct{}

func (RuboCopConfigParser) Name() string { return "rubocop" }

func (p RuboCopConfigParser) Anchors() []string {
	return []string{".rubocop.yml", "Gemfile"}
}

func (p RuboCopConfigParser) Rules() []ParserRule {
	return []ParserRule{
		{RuleName: "Cyclomatic Complexity", Keys: []string{"CyclomaticComplexity"}, Baseline: BaselineComplexity},
		{RuleName: "Function Length", Keys: []string{"MethodLength"}, Baseline: BaselineFunctionLength},
		{RuleName: "Argument Count", Keys: []string{"ParameterLists"}, Baseline: BaselineArgumentCount},
		{RuleName: "File Length", Keys: []string{"ModuleLength"}, Baseline: BaselineFileLength},
	}
}
