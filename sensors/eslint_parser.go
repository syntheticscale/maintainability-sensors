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
		{RuleName: "Cyclomatic Complexity", Keys: []string{"complexity"}, Baseline: BaselineComplexity},
		{RuleName: "Function Length", Keys: []string{"max-lines-per-function"}, Baseline: BaselineFunctionLength},
		{RuleName: "Argument Count", Keys: []string{"max-params"}, Baseline: BaselineArgumentCount},
		{RuleName: "File Length", Keys: []string{"max-lines"}, Baseline: BaselineFileLength},
	}
}
