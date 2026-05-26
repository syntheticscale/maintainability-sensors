package sensors

//nolint // maintainability: highly cohesive test

//nolint // maintainability: highly cohesive logic

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"


)

// ─── ConfigParser Implementations ───

func TestConfigParsers_Anchors(t *testing.T) {
	cases := []struct {
		name     string
		parser   ConfigParser
		expected []string
	}{
		{
			name:     "legacy.ESLintConfigParser",
			parser:   ESLintConfigParser{},
			expected: []string{".eslintrc", ".eslintrc.json", ".eslintrc.js", ".eslintrc.cjs", ".eslintrc.yaml", ".eslintrc.yml", "eslint.config.js", "eslint.config.mjs", "eslint.config.cjs", "package.json"},
		},
		{
			name:     "legacy.PyLintConfigParser",
			parser:   PyLintConfigParser{},
			expected: []string{".pylintrc", "pylintrc", "pyproject.toml", "setup.cfg", ".flake8", "tox.ini"},
		},
		{
			name:     "GoConfigParser",
			parser:   GoConfigParser{},
			expected: []string{".golangci.yml", "golangci.yml"},
		},
		{
			name:     "legacy.RuboCopConfigParser",
			parser:   RuboCopConfigParser{},
			expected: []string{".rubocop.yml"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.parser.Anchors()
			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("%s.Anchors() = %v, want %v", tc.name, got, tc.expected)
			}
		})
	}
}

func verifyParserRules(t *testing.T, parser ConfigParser, expected []ParserRule) {
	got := parser.Rules()
	if len(got) != len(expected) {
		t.Fatalf("%s.Rules() returned %d rules, want %d", parser.Name(), len(got), len(expected))
	}
	for i := range got {
		if got[i].RuleName != expected[i].RuleName {
			t.Errorf("rule %d RuleName = %q, want %q", i, got[i].RuleName, expected[i].RuleName)
		}
		if !reflect.DeepEqual(got[i].Keys, expected[i].Keys) {
			t.Errorf("rule %d Keys = %v, want %v", i, got[i].Keys, expected[i].Keys)
		}
		if got[i].Baseline != expected[i].Baseline {
			t.Errorf("rule %d Baseline = %d, want %d", i, got[i].Baseline, expected[i].Baseline)
		}
	}
}

func TestConfigParsers_Rules_ESLint(t *testing.T) {
	expected := []ParserRule{
		{RuleName: RuleComplexity, Keys: []string{"complexity", "@typescript-eslint/complexity"}, Baseline: BaselineComplexity},
		{RuleName: RuleFunctionLength, Keys: []string{"max-lines-per-function", "@typescript-eslint/max-lines-per-function"}, Baseline: BaselineFunctionLength},
		{RuleName: RuleArgumentCount, Keys: []string{"max-params", "@typescript-eslint/max-params"}, Baseline: BaselineArgumentCount},
		{RuleName: RuleFileLength, Keys: []string{"max-lines"}, Baseline: BaselineFileLength},
	}
	verifyParserRules(t, ESLintConfigParser{}, expected)
}

func TestConfigParsers_Rules_PyLint(t *testing.T) {
	expected := []ParserRule{
		{RuleName: RuleComplexity, Keys: []string{"max-complexity"}, Baseline: BaselineComplexity},
		{RuleName: RuleFunctionLength, Keys: []string{"max-statements"}, Baseline: BaselineFunctionLength},
		{RuleName: RuleArgumentCount, Keys: []string{"max-args"}, Baseline: BaselineArgumentCount},
		{RuleName: RuleFileLength, Keys: []string{"max-module-lines"}, Baseline: BaselineFileLength},
	}
	verifyParserRules(t, PyLintConfigParser{}, expected)
}

func TestConfigParsers_Rules_Go(t *testing.T) {
	expected := []ParserRule{
		{RuleName: RuleComplexity, Keys: []string{"min-complexity"}, Baseline: BaselineComplexity},
		{RuleName: RuleFunctionLength, Keys: []string{"lines"}, Baseline: BaselineFunctionLength},
		{RuleName: RuleArgumentCount, Keys: []string{"argument-limit"}, Baseline: BaselineArgumentCount},
	}
	verifyParserRules(t, GoConfigParser{}, expected)
}

func TestConfigParsers_Rules_RuboCop(t *testing.T) {
	expected := []ParserRule{
		{RuleName: RuleComplexity, Keys: []string{"Metrics/CyclomaticComplexity", "Metrics/PerceivedComplexity"}, Baseline: BaselineComplexity},
		{RuleName: RuleFunctionLength, Keys: []string{"Metrics/MethodLength"}, Baseline: BaselineFunctionLength},
		{RuleName: RuleArgumentCount, Keys: []string{"Metrics/ParameterLists"}, Baseline: BaselineArgumentCount},
		{RuleName: RuleFileLength, Keys: []string{"Metrics/ModuleLength"}, Baseline: BaselineFileLength},
	}
	verifyParserRules(t, RuboCopConfigParser{}, expected)
}

// ─── detectConfig ───

func TestDetectConfig_FindsConfigInSameDir(t *testing.T) {
	tempDir := t.TempDir()

	configPath := filepath.Join(tempDir, ".eslintrc.json")
	if err := os.WriteFile(configPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	filePath := filepath.Join(tempDir, "index.ts")
	if err := os.WriteFile(filePath, []byte("const x = 1;\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	got, _ := DetectConfigAndParser(filePath, "typescript")
	if got != configPath {
		t.Errorf("DetectConfigAndParser(%q, \"typescript\") = %q, want %q", filePath, got, configPath)
	}
}

func TestDetectConfig_WalksUpParentDirs(t *testing.T) {
	tempDir := t.TempDir()
	parentDir := filepath.Join(tempDir, "parent")
	subDir := filepath.Join(parentDir, "child")

	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	configPath := filepath.Join(parentDir, ".eslintrc.json")
	if err := os.WriteFile(configPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	filePath := filepath.Join(subDir, "index.ts")
	if err := os.WriteFile(filePath, []byte("const x = 1;\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	got, _ := DetectConfigAndParser(filePath, "typescript")
	if got != configPath {
		t.Errorf("DetectConfigAndParser(%q, \"typescript\") = %q, want %q", filePath, got, configPath)
	}
}

func TestDetectConfig_ReturnsEmptyWhenNoConfig(t *testing.T) {
	tempDir := t.TempDir()

	filePath := filepath.Join(tempDir, "index.ts")
	if err := os.WriteFile(filePath, []byte("const x = 1;\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	got, _ := DetectConfigAndParser(filePath, "typescript")
	if got != "" {
		t.Errorf("DetectConfigAndParser(%q, \"typescript\") = %q, want \"\"", filePath, got)
	}
}

func TestDetectConfig_UnsupportedLanguageReturnsEmpty(t *testing.T) {
	tempDir := t.TempDir()

	filePath := filepath.Join(tempDir, "index.rs")
	if err := os.WriteFile(filePath, []byte("fn main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	got, _ := DetectConfigAndParser(filePath, "rust")
	if got != "" {
		t.Errorf("DetectConfigAndParser(%q, \"rust\") = %q, want \"\"", filePath, got)
	}
}

// ─── findAllConfigVals ───

func TestFindAllConfigVals_JSON(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		key      string
		expected []int
	}{
		{
			name:     "top-level primitive",
			content:  `{"complexity": 12}`,
			key:      "complexity",
			expected: []int{12},
		},
		{
			name:     "nested object",
			content:  `{"rules": {"complexity": ["error", 15]}}`,
			key:      "complexity",
			expected: []int{15},
		},
		{
			name:     "array of numbers",
			content:  `{"thresholds": [10, 20, 30]}`,
			key:      "thresholds",
			expected: []int{10, 20, 30},
		},
		{
			name:     "multiple occurrences",
			content:  `{"a": {"complexity": 5}, "b": {"complexity": 10}}`,
			key:      "complexity",
			expected: []int{5, 10},
		},
		{
			name:     "missing key returns empty",
			content:  `{"other": 42}`,
			key:      "complexity",
			expected: nil,
		},
		{
			name:     "invalid JSON returns empty",
			content:  `not json at all`,
			key:      "complexity",
			expected: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := findAllConfigVals(tc.content, tc.key, ".json")
			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("findAllConfigValsJSON(%q, %q) = %v, want %v", tc.content, tc.key, got, tc.expected)
			}
		})
	}
}

func TestFindAllConfigVals_NonJSON(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		key      string
		expected []int
	}{
		{
			name:     "INI-style key=value",
			content:  "max-args=7\n",
			key:      "max-args",
			expected: []int{7},
		},
		{
			name:     "YAML-style with spaces",
			content:  "  min-complexity: 15\n",
			key:      "min-complexity",
			expected: []int{15},
		},
		{
			name:     "line with comments is ignored",
			content:  "# max-args=99\n",
			key:      "max-args",
			expected: nil,
		},
		{
			name:     " multiple occurrences",
			content:  "complexity: 5\ncomplexity: 10\n",
			key:      "complexity",
			expected: []int{5, 10},
		},
		{
			name:     "missing key returns empty",
			content:  "other-key: 42\n",
			key:      "max-args",
			expected: nil,
		},
		{
			name:     "YAML single line mapping",
			content:  "  MethodLength: 25\n",
			key:      "MethodLength",
			expected: []int{25},
		},
		{
			name:     "number after key with colon",
			content:  "max-statements: 80\n",
			key:      "max-statements",
			expected: []int{80},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := findAllConfigVals(tc.content, tc.key, ".yml")
			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("findAllConfigVals(%q, %q, \".yml\") = %v, want %v", tc.content, tc.key, got, tc.expected)
			}
		})
	}
}

func TestFindAllConfigVals_FlatConfig(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		key      string
		ext      string
		expected []int
	}{
		{
			name:     "missing key returns empty",
			content:  `[rules]`,
			key:      "complexity",
			ext:      ".toml",
			expected: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := findAllConfigVals(tc.content, tc.key, tc.ext)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("findAllConfigVals(%q, %q, %q) = %v, want %v", tc.content, tc.key, tc.ext, got, tc.expected)
			}
		})
	}
}

func TestFindAllConfigVals_EdgeCases(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		key      string
		ext      string
		expected []int
	}{
		{
			name:     "ignores single-line comments with //",
			content:  "// max-args: 99\nmax-args: 7\n",
			key:      "max-args",
			ext:      ".yml",
			expected: []int{7},
		},
		{
			name:     "ignores inline comments",
			content:  "# this is a comment\nmax-args: 7",
			key:      "max-args",
			ext:      ".yml",
			expected: []int{7},
		},
		{
			name:     "whole word match required",
			content:  "max-args-extra: 99\n",
			key:      "max-args",
			ext:      ".yml",
			expected: nil,
		},
		{
			name:     "does not match partial key in longer word",
			content:  "somecomplexity: 99\n",
			key:      "complexity",
			ext:      ".yml",
			expected: nil,
		},
		{
			name:     "JSON with float values",
			content:  `{"complexity": 12.5}`,
			key:      "complexity",
			ext:      ".json",
			expected: []int{12},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := findAllConfigVals(tc.content, tc.key, tc.ext)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("findAllConfigVals(%q, %q, %q) = %v, want %v", tc.content, tc.key, tc.ext, got, tc.expected)
			}
		})
	}
}

// ─── getParserForLang ───

func TestGetParserForLang(t *testing.T) {
	cases := []struct {
		lang     string
		wantName string
		wantNil  bool
	}{
		{"typescript", "eslint", false},
		{"javascript", "eslint", false},
		{"python", "pylint", false},
		{"go", "golangci", false},
		{"ruby", "rubocop", false},
		{"rust", "", true},
		{"java", "", true},
		{"csharp", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.lang, func(t *testing.T) {
			got := getParsersForLang(tc.lang)
			if tc.wantNil {
				if len(got) > 0 {
					t.Errorf("getParsersForLang(%q) = %v, want nil", tc.lang, got)
				}
				return
			}
			if len(got) == 0 {
				t.Fatalf("getParsersForLang(%q) = nil, want non-nil", tc.lang)
			}
			found := false
			for _, p := range got {
				if p.Name() == tc.wantName {
					found = true
				}
			}
			if !found {
				t.Errorf("expected to find %s parser", tc.wantName)
			}
		})
	}
}

// ─── DetectRelaxedLimits ───

func TestDetectRelaxedLimits_ESLintJSON(t *testing.T) {
	tempDir := t.TempDir()

	content := `{
		"rules": {
			"complexity": ["error", 12],
			"max-params": ["error", 6],
			"max-lines-per-function": ["error", 100],
			"max-lines": ["error", 500]
		}
	}`

	configPath := filepath.Join(tempDir, ".eslintrc.json")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	got := DetectRelaxedLimits(configPath, ESLintConfigParser{})

	if len(got) != 4 {
		t.Fatalf("expected 4 relaxed limits, got %d: %+v", len(got), got)
	}

	expectedMap := map[string]int{
		"Complexity":     12,
		"ArgumentCount":  6,
		"FunctionLength": 100,
		"FileLength":     500,
	}

	for _, exc := range got {
		wantVal, ok := expectedMap[exc.RuleName]
		if !ok {
			t.Errorf("unexpected rule name: %s", exc.RuleName)
			continue
		}
		if exc.ConfiguredVal != wantVal {
			t.Errorf("%s: ConfiguredVal = %d, want %d", exc.RuleName, exc.ConfiguredVal, wantVal)
		}
		if exc.BaselineVal == 0 {
			t.Errorf("%s: BaselineVal should not be zero", exc.RuleName)
		}
	}
}

func TestDetectRelaxedLimits_PyLintINI(t *testing.T) {
	tempDir := t.TempDir()

	content := `[DESIGN]
max-args=7
max-statements=80
max-complexity=11
max-module-lines=450
`

	configPath := filepath.Join(tempDir, ".pylintrc")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	got := DetectRelaxedLimits(configPath, PyLintConfigParser{})

	if len(got) != 4 {
		t.Fatalf("expected 4 relaxed limits, got %d: %+v", len(got), got)
	}

	expectedMap := map[string]int{
		"Complexity":     11,
		"ArgumentCount":  7,
		"FunctionLength": 80,
		"FileLength":     450,
	}

	for _, exc := range got {
		wantVal, ok := expectedMap[exc.RuleName]
		if !ok {
			t.Errorf("unexpected rule name: %s", exc.RuleName)
			continue
		}
		if exc.ConfiguredVal != wantVal {
			t.Errorf("%s: ConfiguredVal = %d, want %d", exc.RuleName, exc.ConfiguredVal, wantVal)
		}
	}
}

func TestDetectRelaxedLimits_NoRelaxation(t *testing.T) {
	tempDir := t.TempDir()

	// Config with values below baselines — nothing should be flagged as relaxed
	content := `{
		"rules": {
			"complexity": ["error", 5],
			"max-params": ["error", 3]
		}
	}`

	configPath := filepath.Join(tempDir, ".eslintrc.json")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	got := DetectRelaxedLimits(configPath, ESLintConfigParser{})

	if len(got) != 0 {
		t.Errorf("expected 0 relaxed limits when all values are below baseline, got %d: %+v", len(got), got)
	}
}

func TestDetectRelaxedLimits_EmptyPath(t *testing.T) {
	got := DetectRelaxedLimits("", ESLintConfigParser{})
	if len(got) != 0 {
		t.Errorf("expected empty slice for empty config path, got %d: %+v", len(got), got)
	}
}

func TestDetectRelaxedLimits_RuboCopYAML(t *testing.T) {
	tempDir := t.TempDir()

	content := `Metrics/CyclomaticComplexity:
  Max: 15
  Enabled: true

Metrics/MethodLength:
  Max: 60
  CountComments: false
  Enabled: true
`

	configPath := filepath.Join(tempDir, ".rubocop.yml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	got := DetectRelaxedLimits(configPath, RuboCopConfigParser{})

	if len(got) != 2 {
		t.Fatalf("expected 2 relaxed limits, got %d: %+v", len(got), got)
	}

	expectedMap := map[string]int{
		"Complexity":     15,
		"FunctionLength": 60,
	}

	for _, exc := range got {
		wantVal, ok := expectedMap[exc.RuleName]
		if !ok {
			t.Errorf("unexpected rule name: %s", exc.RuleName)
			continue
		}
		if exc.ConfiguredVal != wantVal {
			t.Errorf("%s: ConfiguredVal = %d, want %d", exc.RuleName, exc.ConfiguredVal, wantVal)
		}
	}
}

func TestDetectRelaxedLimits_GoYAML(t *testing.T) {
	tempDir := t.TempDir()

	content := `run:
  timeout: 5m

linters-settings:
  gocognit:
    min-complexity: 12
  funlen:
    lines: 70
    statements: 40
  gocyclo:
    min-complexity: 10
`

	configPath := filepath.Join(tempDir, ".golangci.yml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	got := DetectRelaxedLimits(configPath, GoConfigParser{})

	if len(got) != 2 {
		t.Fatalf("expected 2 relaxed limits, got %d: %+v", len(got), got)
	}

	expectedMap := map[string]int{
		"Complexity":     12, // Because 12 > 10 (maxOf used)
		"FunctionLength": 70,
	}

	for _, exc := range got {
		wantVal, ok := expectedMap[exc.RuleName]
		if !ok {
			t.Errorf("unexpected rule name: %s", exc.RuleName)
			continue
		}
		if exc.ConfiguredVal != wantVal {
			t.Errorf("%s: ConfiguredVal = %d, want %d", exc.RuleName, exc.ConfiguredVal, wantVal)
		}
	}
}
