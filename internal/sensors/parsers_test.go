package sensors

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
			name:     "ESLintConfigParser",
			parser:   ESLintConfigParser{},
			expected: []string{"package.json", ".eslintrc.json", ".eslintrc.js", ".eslintrc.yaml", ".eslintrc.yml", "eslint.config.js", "eslint.config.mjs"},
		},
		{
			name:     "PyLintConfigParser",
			parser:   PyLintConfigParser{},
			expected: []string{"pyproject.toml", ".pylintrc", "setup.cfg", "tox.ini"},
		},
		{
			name:     "GoConfigParser",
			parser:   GoConfigParser{},
			expected: []string{".golangci.yml", "golangci.yml"},
		},
		{
			name:     "RuboCopConfigParser",
			parser:   RuboCopConfigParser{},
			expected: []string{".rubocop.yml", "Gemfile"},
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

func TestConfigParsers_Rules(t *testing.T) {
	cases := []struct {
		name     string
		parser   ConfigParser
		expected []ParserRule
	}{
		{
			name:   "ESLintConfigParser",
			parser: ESLintConfigParser{},
			expected: []ParserRule{
				{RuleName: "Cyclomatic Complexity", Keys: []string{"complexity"}, Baseline: BaselineComplexity},
				{RuleName: "Function Length", Keys: []string{"max-lines-per-function"}, Baseline: BaselineFunctionLength},
				{RuleName: "Argument Count", Keys: []string{"max-params"}, Baseline: BaselineArgumentCount},
				{RuleName: "File Length", Keys: []string{"max-lines"}, Baseline: BaselineFileLength},
			},
		},
		{
			name:   "PyLintConfigParser",
			parser: PyLintConfigParser{},
			expected: []ParserRule{
				{RuleName: "Cyclomatic Complexity", Keys: []string{"max-complexity"}, Baseline: BaselineComplexity},
				{RuleName: "Function Length", Keys: []string{"max-statements"}, Baseline: BaselineFunctionLength},
				{RuleName: "Argument Count", Keys: []string{"max-args"}, Baseline: BaselineArgumentCount},
				{RuleName: "File Length", Keys: []string{"max-module-lines"}, Baseline: BaselineFileLength},
			},
		},
		{
			name:   "GoConfigParser",
			parser: GoConfigParser{},
			expected: []ParserRule{
				{RuleName: "Cyclomatic Complexity", Keys: []string{"min-complexity"}, Baseline: BaselineComplexity},
				{RuleName: "Function Length", Keys: []string{"lines"}, Baseline: BaselineFunctionLength},
				{RuleName: "Argument Count", Keys: []string{"argument-limit"}, Baseline: BaselineArgumentCount},
			},
		},
		{
			name:   "RuboCopConfigParser",
			parser: RuboCopConfigParser{},
			expected: []ParserRule{
				{RuleName: "Cyclomatic Complexity", Keys: []string{"CyclomaticComplexity"}, Baseline: BaselineComplexity},
				{RuleName: "Function Length", Keys: []string{"MethodLength"}, Baseline: BaselineFunctionLength},
				{RuleName: "Argument Count", Keys: []string{"ParameterLists"}, Baseline: BaselineArgumentCount},
				{RuleName: "File Length", Keys: []string{"ModuleLength"}, Baseline: BaselineFileLength},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.parser.Rules()
			if len(got) != len(tc.expected) {
				t.Fatalf("%s.Rules() returned %d rules, want %d", tc.name, len(got), len(tc.expected))
			}
			for i := range got {
				if got[i].RuleName != tc.expected[i].RuleName {
					t.Errorf("rule %d RuleName = %q, want %q", i, got[i].RuleName, tc.expected[i].RuleName)
				}
				if !reflect.DeepEqual(got[i].Keys, tc.expected[i].Keys) {
					t.Errorf("rule %d Keys = %v, want %v", i, got[i].Keys, tc.expected[i].Keys)
				}
				if got[i].Baseline != tc.expected[i].Baseline {
					t.Errorf("rule %d Baseline = %d, want %d", i, got[i].Baseline, tc.expected[i].Baseline)
				}
			}
		})
	}
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

	got := detectConfig(filePath, "typescript")
	if got != configPath {
		t.Errorf("detectConfig(%q, \"typescript\") = %q, want %q", filePath, got, configPath)
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

	got := detectConfig(filePath, "typescript")
	if got != configPath {
		t.Errorf("detectConfig(%q, \"typescript\") = %q, want %q", filePath, got, configPath)
	}
}

func TestDetectConfig_ReturnsEmptyWhenNoConfig(t *testing.T) {
	tempDir := t.TempDir()

	filePath := filepath.Join(tempDir, "index.ts")
	if err := os.WriteFile(filePath, []byte("const x = 1;\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	got := detectConfig(filePath, "typescript")
	if got != "" {
		t.Errorf("detectConfig(%q, \"typescript\") = %q, want \"\"", filePath, got)
	}
}

func TestDetectConfig_UnsupportedLanguageReturnsEmpty(t *testing.T) {
	tempDir := t.TempDir()

	filePath := filepath.Join(tempDir, "index.rs")
	if err := os.WriteFile(filePath, []byte("fn main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	got := detectConfig(filePath, "rust")
	if got != "" {
		t.Errorf("detectConfig(%q, \"rust\") = %q, want \"\"", filePath, got)
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
			name:     "JS format with array",
			content:  `export default [ { rules: { "complexity": ["error", 20] } } ];`,
			key:      "complexity",
			ext:      ".js",
			expected: []int{20},
		},
		{
			name:     "JS format with unquoted key and primitive value",
			content:  `export default [ { rules: { complexity: 15 } } ];`,
			key:      "complexity",
			ext:      ".mjs",
			expected: []int{15},
		},
		{
			name:     "JS format with single quotes",
			content:  `export default [ { rules: { 'max-params': ['warn', 5] } } ];`,
			key:      "max-params",
			ext:      ".js",
			expected: []int{5},
		},
		{
			name:     "JS format multiple occurrences",
			content:  `"complexity": ["error", 10] and later complexity: 30`,
			key:      "complexity",
			ext:      ".js",
			expected: []int{10, 30},
		},
		{
			name:     "JS format with array and object with max",
			content:  `export default [ { rules: { "max-lines-per-function": [ "error", { "max": 50, "skipBlankLines": true } ] } } ];`,
			key:      "max-lines-per-function",
			ext:      ".js",
			expected: []int{50},
		},
		{
			name:     "JS format with single object and max",
			content:  `export default [ { rules: { "complexity": { "max": 10 } } } ];`,
			key:      "complexity",
			ext:      ".js",
			expected: []int{10},
		},
		{
			name: "JS format with multiline array and object",
			content: `
			"max-lines-per-function": [
				"error", 
				{ 
					"max": 60,
					"skipBlankLines": true,
					"skipComments": true
				}
			]`,
			key:      "max-lines-per-function",
			ext:      ".js",
			expected: []int{60},
		},
		{
			name:     "missing key returns empty",
			content:  `export default [];`,
			key:      "complexity",
			ext:      ".js",
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
		"Cyclomatic Complexity": 12,
		"Argument Count":        6,
		"Function Length":       100,
		"File Length":           500,
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
		"Cyclomatic Complexity": 11,
		"Argument Count":        7,
		"Function Length":       80,
		"File Length":           450,
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
		"Cyclomatic Complexity": 15,
		"Function Length":       60,
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
		"Cyclomatic Complexity": 12, // Because 12 > 10 (maxOf used)
		"Function Length":       70,
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
