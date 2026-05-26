package sensors

// BiomeConfigParser extracts thresholds from Biome configs.
type BiomeConfigParser struct{}

func (BiomeConfigParser) Name() string { return "biome" }

func (p BiomeConfigParser) Anchors() []string {
	return []string{"biome.json", "biome.jsonc"}
}

func (p BiomeConfigParser) Rules() []ParserRule {
	return []ParserRule{
		{RuleName: RuleComplexity, Keys: []string{"complexity", "max-complexity"}, Baseline: BaselineComplexity},
		{RuleName: RuleFunctionLength, Keys: []string{"max-lines-per-function", "max-statements"}, Baseline: BaselineFunctionLength},
		{RuleName: RuleArgumentCount, Keys: []string{"max-params", "max-args"}, Baseline: BaselineArgumentCount},
	}
}

// ESLintConfigParser extracts thresholds from ESLint / ts-standard configs.
type ESLintConfigParser struct{}

func (ESLintConfigParser) Name() string { return "eslint" }

func (p ESLintConfigParser) Anchors() []string {
	return []string{
		".eslintrc", ".eslintrc.json", ".eslintrc.js", ".eslintrc.cjs", ".eslintrc.yaml", ".eslintrc.yml",
		"eslint.config.js", "eslint.config.mjs", "eslint.config.cjs",
		"package.json",
	}
}

func (p ESLintConfigParser) Rules() []ParserRule {
	return []ParserRule{
		{RuleName: RuleComplexity, Keys: []string{"complexity", "@typescript-eslint/complexity"}, Baseline: BaselineComplexity},
		{RuleName: RuleFunctionLength, Keys: []string{"max-lines-per-function", "@typescript-eslint/max-lines-per-function"}, Baseline: BaselineFunctionLength},
		{RuleName: RuleArgumentCount, Keys: []string{"max-params", "@typescript-eslint/max-params"}, Baseline: BaselineArgumentCount},
		{RuleName: RuleFileLength, Keys: []string{"max-lines"}, Baseline: BaselineFileLength},
	}
}

// PyLintConfigParser extracts thresholds from PyLint / flake8 configs.
type PyLintConfigParser struct{}

func (PyLintConfigParser) Name() string { return "pylint" }

func (p PyLintConfigParser) Anchors() []string {
	return []string{".pylintrc", "pylintrc", "pyproject.toml", "setup.cfg", ".flake8", "tox.ini"}
}

func (p PyLintConfigParser) Rules() []ParserRule {
	return []ParserRule{
		{RuleName: RuleComplexity, Keys: []string{"max-complexity"}, Baseline: BaselineComplexity},
		{RuleName: RuleFunctionLength, Keys: []string{"max-statements"}, Baseline: BaselineFunctionLength},
		{RuleName: RuleArgumentCount, Keys: []string{"max-args"}, Baseline: BaselineArgumentCount},
		{RuleName: RuleFileLength, Keys: []string{"max-module-lines"}, Baseline: BaselineFileLength},
	}
}

// RuboCopConfigParser extracts thresholds from RuboCop configs.
type RuboCopConfigParser struct{}

func (RuboCopConfigParser) Name() string { return "rubocop" }

func (p RuboCopConfigParser) Anchors() []string {
	return []string{".rubocop.yml"}
}

func (p RuboCopConfigParser) Rules() []ParserRule {
	return []ParserRule{
		{RuleName: RuleComplexity, Keys: []string{"Metrics/CyclomaticComplexity", "Metrics/PerceivedComplexity"}, Baseline: BaselineComplexity},
		{RuleName: RuleFunctionLength, Keys: []string{"Metrics/MethodLength"}, Baseline: BaselineFunctionLength},
		{RuleName: RuleArgumentCount, Keys: []string{"Metrics/ParameterLists"}, Baseline: BaselineArgumentCount},
		{RuleName: RuleFileLength, Keys: []string{"Metrics/ModuleLength"}, Baseline: BaselineFileLength},
	}
}

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
		{RuleName: RuleFileLength, Keys: []string{"max-module-lines"}, Baseline: BaselineFileLength},
	}
}

// StandardRBConfigParser extracts thresholds from StandardRB configs.
type StandardRBConfigParser struct{}

func (StandardRBConfigParser) Name() string { return "standardrb" }

func (p StandardRBConfigParser) Anchors() []string {
	return []string{".standard.yml"}
}

func (p StandardRBConfigParser) Rules() []ParserRule {
	// StandardRB rules are generally fixed, but some overrides can be extracted
	return []ParserRule{
		{RuleName: RuleComplexity, Keys: []string{"Metrics/CyclomaticComplexity"}, Baseline: BaselineComplexity},
		{RuleName: RuleFunctionLength, Keys: []string{"Metrics/MethodLength"}, Baseline: BaselineFunctionLength},
		{RuleName: RuleArgumentCount, Keys: []string{"Metrics/ParameterLists"}, Baseline: BaselineArgumentCount},
	}
}
