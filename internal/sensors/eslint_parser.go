package sensors

// ESLintConfigParser extracts thresholds from ESLint / ts-standard configs.
type ESLintConfigParser struct{}

func (ESLintConfigParser) Name() string { return "eslint" }

func (p ESLintConfigParser) Anchors() []string {
	return []string{
		"package.json",
		".eslintrc.json",
		".eslintrc.js",
		".eslintrc.yaml",
		".eslintrc.yml",
		"eslint.config.js",
		"eslint.config.mjs",
	}
}

func (p ESLintConfigParser) Rules() []ParserRule {
	return []ParserRule{
		{RuleName: RuleComplexity, Keys: []string{"complexity"}, Baseline: BaselineComplexity},
		{RuleName: RuleFunctionLength, Keys: []string{"max-lines-per-function"}, Baseline: BaselineFunctionLength},
		{RuleName: RuleArgumentCount, Keys: []string{"max-params"}, Baseline: BaselineArgumentCount},
		{RuleName: RuleFileLength, Keys: []string{"max-lines"}, Baseline: BaselineFileLength},
	}
}
