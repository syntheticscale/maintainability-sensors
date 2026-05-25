package sensors

import (
	"fmt"
	"regexp"
	"strings"
)

type BiomePlugin struct{}

func (p BiomePlugin) Name() string {
	return "biome"
}

type BiomeLocation struct {
	Path struct {
		File string `json:"file"`
	} `json:"path"`
	Span struct {
		Start int `json:"start"`
		End   int `json:"end"`
	} `json:"span"`
}

type BiomeDiagnostic struct {
	Category    string        `json:"category"`
	Location    BiomeLocation `json:"location"`
	Description string        `json:"description"`
}

type BiomeResult struct {
	Diagnostics []BiomeDiagnostic `json:"diagnostics"`
}

func extractBiomeComplexity(desc string, reVal *regexp.Regexp) (string, int) {
	var val int
	if m := reVal.FindStringSubmatch(desc); len(m) > 1 {
		fmt.Sscanf(m[1], "%d", &val)
	}
	if val == 0 {
		val = 2
	}
	return RuleComplexity, val
}

func extractBiomeMaxParameters(desc string, reVal *regexp.Regexp) (string, int) {
	var val int
	if m := reVal.FindStringSubmatch(desc); len(m) > 1 {
		fmt.Sscanf(m[1], "%d", &val)
	}
	if val == 0 {
		val = 2
	}
	return RuleArgumentCount, val
}

func extractBiomeRuleAndVal(diag BiomeDiagnostic, reVal *regexp.Regexp) (string, int) {
	isComplexity := strings.Contains(diag.Category, "complexity") || strings.Contains(diag.Description, "complexity")
	if isComplexity {
		return extractBiomeComplexity(diag.Description, reVal)
	}
	isParams := strings.Contains(diag.Category, "maxParameters") || strings.Contains(diag.Description, "parameters")
	if isParams {
		return extractBiomeMaxParameters(diag.Description, reVal)
	}
	return "", 0
}

func parseSingleBiomeDiagnostic(diag BiomeDiagnostic, reVal *regexp.Regexp, fileViolations *[]Violation) {
	rule, val := extractBiomeRuleAndVal(diag, reVal)
	if rule == "" {
		return
	}

	startLine := diag.Location.Span.Start
	endLine := diag.Location.Span.End
	if endLine == 0 {
		endLine = startLine + FallbackEndLineOffset
	}

	*fileViolations = append(*fileViolations, Violation{RuleName: rule, Value: val, StartLine: startLine, EndLine: endLine, Message: diag.Description})
}

//nolint:gocognit,cyclop // Highly cohesive mapping logic, splitting hurts readability
func parseBiomeMessages(diagnostics []BiomeDiagnostic) map[string][]Violation {
	metricsMap := make(map[string][]Violation)
	reVal := regexp.MustCompile(`(\d+)`)
	for _, diag := range diagnostics {
		path := diag.Location.Path.File
		if path == "" {
			continue
		}
		var violations []Violation
		parseSingleBiomeDiagnostic(diag, reVal, &violations)
		if len(violations) > 0 {
			metricsMap[path] = append(metricsMap[path], violations...)
		}
	}
	return metricsMap
}

func processBiomeAnalyzeResult(exitCode int, result BiomeResult, output []byte) (map[string][]Violation, error) {
	if exitCode == 0 || exitCode == 1 {
		metricsMap := make(map[string][]Violation)
		if len(result.Diagnostics) > 0 {
			metricsMap = parseBiomeMessages(result.Diagnostics)
		}
		if exitCode == 1 && len(metricsMap) == 0 && len(output) > 0 {
			return nil, fmt.Errorf("biome crashed or encountered a configuration error (exit code 1): %s", strings.TrimSpace(string(output)))
		}
		return metricsMap, nil
	}

	return nil, fmt.Errorf("biome exited with unexpected code %d: %s", exitCode, strings.TrimSpace(string(output)))
}

func (p BiomePlugin) Analyze(files []FileContext) (map[string][]Violation, error) {
	var filePaths []string
	for _, f := range files {
		filePaths = append(filePaths, f.Path)
	}

	args := []string{"lint", "--formatter-enabled=false", "--output-format=json"}
	args = append(args, "--")
	args = append(args, filePaths...)

	var result BiomeResult
	exitCode, output, err := runLintCommandJSON("biome", &result, args...)
	if err != nil {
		if exitCode > 0 {
			return nil, fmt.Errorf("biome crashed or encountered a configuration error (exit code %d): %v\n%s", exitCode, err, string(output))
		}
		return nil, fmt.Errorf("biome error: %w", err)
	}

	return processBiomeAnalyzeResult(exitCode, result, output)
}
