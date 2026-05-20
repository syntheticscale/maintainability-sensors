package sensors

// GoConfigParser extracts thresholds from golangci-lint configs.
type GoConfigParser struct{}

func (GoConfigParser) Name() string { return "golangci" }

func (p GoConfigParser) Anchors() []string {
	return []string{".golangci.yml", "golangci.yml"}
}

func (p GoConfigParser) Rules() []ParserRule {
	return []ParserRule{
		{RuleName: "Cyclomatic Complexity", Keys: []string{"min-complexity"}, Baseline: BaselineComplexity},
		{RuleName: "Function Length", Keys: []string{"lines"}, Baseline: BaselineFunctionLength},
	}
}
