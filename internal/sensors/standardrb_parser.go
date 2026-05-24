package sensors

// StandardRBConfigParser extracts thresholds from StandardRB configs.
type StandardRBConfigParser struct{}

func (StandardRBConfigParser) Name() string { return "standardrb" }

func (p StandardRBConfigParser) Anchors() []string {
	return []string{".standard.yml"}
}

func (p StandardRBConfigParser) Rules() []ParserRule {
	return []ParserRule{
		{RuleName: "Cyclomatic Complexity", Keys: []string{"CyclomaticComplexity"}, Baseline: BaselineComplexity},
		{RuleName: "Function Length", Keys: []string{"MethodLength"}, Baseline: BaselineFunctionLength},
		{RuleName: "Argument Count", Keys: []string{"ParameterLists"}, Baseline: BaselineArgumentCount},
		{RuleName: "File Length", Keys: []string{"ModuleLength"}, Baseline: BaselineFileLength},
	}
}
