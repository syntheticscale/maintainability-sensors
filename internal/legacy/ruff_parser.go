package legacy

// RuffConfigParser extracts thresholds from Ruff configs.
type RuffConfigParser struct{}

func (RuffConfigParser) Name() string { return "ruff" }

func (p RuffConfigParser) Anchors() []string {
	return []string{"ruff.toml", ".ruff.toml", "pyproject.toml"}
}

func (p RuffConfigParser) Rules() []ParserRule {
	return []ParserRule{
		{RuleName: RuleComplexity, Keys: []string{"max-complexity"}, Baseline: BaselineComplexity},
		{RuleName: RuleFunctionLength, Keys: []string{"max-statements"}, Baseline: BaselineFunctionLength},
		{RuleName: RuleArgumentCount, Keys: []string{"max-args"}, Baseline: BaselineArgumentCount},
	}
}
