package sensors

// PyLintConfigParser extracts thresholds from PyLint / flake8 configs.
type PyLintConfigParser struct{}

func (PyLintConfigParser) Name() string { return "pylint" }

func (p PyLintConfigParser) Anchors() []string {
	return []string{"pyproject.toml", ".pylintrc", "setup.cfg", "tox.ini"}
}

func (p PyLintConfigParser) Rules() []ParserRule {
	return []ParserRule{
		{RuleName: "Cyclomatic Complexity", Keys: []string{"max-complexity"}, Baseline: BaselineComplexity},
		{RuleName: "Function Length", Keys: []string{"max-statements"}, Baseline: BaselineFunctionLength},
		{RuleName: "Argument Count", Keys: []string{"max-args"}, Baseline: BaselineArgumentCount},
		{RuleName: "File Length", Keys: []string{"max-module-lines"}, Baseline: BaselineFileLength},
	}
}
