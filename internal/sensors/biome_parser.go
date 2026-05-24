package sensors

// BiomeConfigParser extracts thresholds from Biome configs.
type BiomeConfigParser struct{}

func (BiomeConfigParser) Name() string { return "biome" }

func (p BiomeConfigParser) Anchors() []string {
	return []string{"biome.json", "biome.jsonc"}
}

func (p BiomeConfigParser) Rules() []ParserRule {
	return []ParserRule{
		{RuleName: "Cyclomatic Complexity", Keys: []string{"complexity"}, Baseline: BaselineComplexity},
		{RuleName: "Function Length", Keys: []string{"maxLines"}, Baseline: BaselineFunctionLength},
		{RuleName: "Argument Count", Keys: []string{"maxParameters"}, Baseline: BaselineArgumentCount},
	}
}
