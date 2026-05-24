package sensors

// RuffConfigParser extracts thresholds from Ruff configs.
type RuffConfigParser struct{}

func (RuffConfigParser) Name() string { return "ruff" }

func (p RuffConfigParser) Anchors() []string {
	return []string{"ruff.toml", ".ruff.toml", "pyproject.toml"}
}

func (p RuffConfigParser) Rules() []ParserRule {
	return []ParserRule{
		{RuleName: "Cyclomatic Complexity", Keys: []string{"max-complexity"}, Baseline: BaselineComplexity},
		{RuleName: "Function Length", Keys: []string{"max-statements"}, Baseline: BaselineFunctionLength},
		{RuleName: "Argument Count", Keys: []string{"max-args"}, Baseline: BaselineArgumentCount},
	}
}
