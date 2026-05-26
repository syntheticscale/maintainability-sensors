package legacy

// PyLintConfigParser extracts thresholds from PyLint / flake8 configs.
type PyLintConfigParser struct{}

func (PyLintConfigParser) Name() string { return "pylint" }

func (p PyLintConfigParser) Anchors() []string {
	return []string{"pyproject.toml", ".pylintrc", "setup.cfg", "tox.ini"}
}

func (p PyLintConfigParser) Rules() []ParserRule {
	return []ParserRule{
		{RuleName: RuleComplexity, Keys: []string{"max-complexity"}, Baseline: BaselineComplexity},
		{RuleName: RuleFunctionLength, Keys: []string{"max-statements"}, Baseline: BaselineFunctionLength},
		{RuleName: RuleArgumentCount, Keys: []string{"max-args"}, Baseline: BaselineArgumentCount},
		{RuleName: RuleFileLength, Keys: []string{"max-module-lines"}, Baseline: BaselineFileLength},
	}
}
