package sensors

// BiomeConfigParser extracts thresholds from Biome configs.
type BiomeConfigParser struct{}

func (BiomeConfigParser) Name() string { return "biome" }

func (p BiomeConfigParser) Anchors() []string {
	return []string{"biome.json", "biome.jsonc"}
}

func (p BiomeConfigParser) Rules() []ParserRule {
	return []ParserRule{
		{RuleName: RuleComplexity, Keys: []string{"complexity"}, Baseline: BaselineComplexity},
		{RuleName: RuleFunctionLength, Keys: []string{"maxLines"}, Baseline: BaselineFunctionLength},
		{RuleName: RuleArgumentCount, Keys: []string{"maxParameters"}, Baseline: BaselineArgumentCount},
	}
}
