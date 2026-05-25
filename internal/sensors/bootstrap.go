package sensors

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	eslintFlatTemplate = `import typescriptEslint from "@typescript-eslint/eslint-plugin";
import tsParser from "@typescript-eslint/parser";

export default [
  {
    files: ["**/*.ts", "**/*.tsx", "**/*.js", "**/*.jsx"],
    languageOptions: {
      parser: tsParser,
    },
    plugins: {
      "@typescript-eslint": typescriptEslint,
    },
    rules: {
      "complexity": ["error", %d],
      "max-params": ["error", %d],
      "max-lines-per-function": ["error", { "max": %d, "skipBlankLines": true, "skipComments": true }],
      "max-lines": ["error", { "max": %d, "skipBlankLines": true, "skipComments": true }],
      "@typescript-eslint/no-explicit-any": "warn"
    }
  }
];
`

	pylintTemplate = `[MASTER]
load-plugins=pylint.extensions.mccabe

[DESIGN]
max-args=%d
max-statements=%d
max-complexity=%d
max-module-lines=%d
`

	golangciTemplate = `run:
  timeout: 5m

linters-settings:
  gocognit:
    min-complexity: %d
  funlen:
    lines: %d
    statements: 40
  cyclop:
    max-complexity: %d
  lll:
    line-length: 120
  revive:
    rules:
      - name: argument-limit
        arguments: [%d]

linters:
  enable:
    - gocognit
    - funlen
    - cyclop
    - lll
    - revive
`

	checkstyleTemplate = `<?xml version="1.0"?>
<!DOCTYPE module PUBLIC
          "-//Checkstyle//DTD Checkstyle Configuration 1.3//EN"
          "https://checkstyle.org/dtds/configuration_1_3.dtd">
<module name="Checker">
  <property name="severity" value="warning"/>
  <module name="TreeWalker">
    <!-- Cyclomatic Complexity Limit: max %d -->
    <module name="CyclomaticComplexity">
      <property name="max" value="%d"/>
    </module>
    <!-- Method Parameter (Argument) Count Limit: max %d -->
    <module name="ParameterNumber">
      <property name="max" value="%d"/>
    </module>
    <!-- Method Length (Function Length) Limit: max %d lines -->
    <module name="MethodLength">
      <property name="max" value="%d"/>
      <property name="countEmpty" value="false"/>
    </module>
    <!-- File Length Limit: max %d lines -->
    <module name="FileLength">
      <property name="max" value="%d"/>
    </module>
  </module>
</module>
`

	rubocopTemplate = `# Pristine RuboCop Maintainability Rules
Metrics/CyclomaticComplexity:
  Max: %d
  Enabled: true

Metrics/MethodLength:
  Max: %d
  CountComments: false
  Enabled: true

Metrics/ParameterLists:
  Max: %d
  Enabled: true

Metrics/ModuleLength:
  Max: %d
  Enabled: true
`

	editorconfigTemplate = `# Pristine Microsoft .editorconfig Maintainability Rules
root = true

[*.cs]
# Roslyn CA1502: Avoid excessive cyclomatic complexity (Limit: %d)
dotnet_diagnostic.CA1502.severity = warning
dotnet_code_quality.CA1502.maximum_cyclomatic_complexity = %d

# Roslyn CA1506: Avoid excessive class coupling
dotnet_diagnostic.CA1506.severity = warning

# Enforce standard method length / formatting rules
dotnet_sort_system_directives_first = true
`
)

// BootstrapRepo detects the languages/frameworks of a repository
// and boots up pristine, non-overwriting configs with maintainability thresholds.
func BootstrapRepo(repoPath string) error {
	return BootstrapRepoWithPolicy(repoPath, false)
}

// BootstrapRepoWithPolicy detects the languages/frameworks of a repository
// and boots up pristine, non-overwriting configs. If warnPolicy is true,
// it also generates a .maintainability-sensors.yml with default-severity: warn.
func BootstrapRepoWithPolicy(repoPath string, warnPolicy bool) error {
	absPath, info, err := resolveAndStatPath(repoPath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("target path is not a directory: %s", absPath)
	}

	langs := detectLanguages(absPath)
	if len(langs) == 0 {
		return fmt.Errorf("no supported codebase language detected (TS/JS, Python, Go, Java) in directory: %s", absPath)
	}

	if err := orchestrateBootstrapping(langs, absPath); err != nil {
		return err
	}

	if warnPolicy {
		return bootstrapMaintainabilitySensors(absPath)
	}
	return nil
}

func resolveAndStatPath(repoPath string) (string, os.FileInfo, error) {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		absPath = repoPath
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", nil, fmt.Errorf("target path does not exist: %w", err)
	}
	return absPath, info, nil
}

func orchestrateBootstrapping(langs []string, absPath string) error {
	for _, lang := range langs {
		fmt.Fprintf(os.Stderr, "=========================================\n")
		fmt.Fprintf(os.Stderr, " Orchestrating Bootstrap for %s...\n", getFriendlyLangName(lang))
		fmt.Fprintf(os.Stderr, "=========================================\n\n")

		if err := bootstrapLanguage(lang, absPath); err != nil {
			return err
		}
	}
	return nil
}

func bootstrapMaintainabilitySensors(absPath string) error {
	configPath := filepath.Join(absPath, ".maintainability-sensors.yml")
	if _, err := os.Stat(configPath); err == nil {
		fmt.Fprintf(os.Stderr, "- [SKIP] '.maintainability-sensors.yml' already exists in repository root. Protecting existing setup.\n")
		return nil
	}
	content := `version: "1"
check-diff:
  default-severity: warn
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write .maintainability-sensors.yml: %w", err)
	}
	fmt.Fprintf(os.Stderr, "- [CREATED] .maintainability-sensors.yml (Gradual Adoption Policy)\n")
	fmt.Fprintf(os.Stderr, "  default-severity: warn\n\n")
	return nil
}

func bootstrapLanguage(lang, absPath string) error {
	switch lang {
	case "tsjs":
		return bootstrapTSJS(absPath)
	case "python":
		return bootstrapPython(absPath)
	case "go":
		return bootstrapGo(absPath)
	case "java":
		return bootstrapJava(absPath)
	case "ruby":
		return bootstrapRuby(absPath)
	case "csharp":
		return bootstrapCSharp(absPath)
	}
	return nil
}

func findExistingESLintConfig(absPath string) string {
	anchors := ESLintConfigParser{}.Anchors()
	for _, anchor := range anchors {
		p := filepath.Join(absPath, anchor)
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if anchor == "package.json" {
			if !hasESLintConfigInPackageJson(p, info) {
				continue
			}
		}
		return anchor
	}
	return ""
}

func hasESLintConfigInPackageJson(p string, info os.FileInfo) bool {
	if !info.Mode().IsRegular() || info.Size() > MaxFileSize {
		return false
	}
	content, err := os.ReadFile(p)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), `"eslintConfig"`)
}

func bootstrapTSJS(absPath string) error {
	existingConfig := findExistingESLintConfig(absPath)

	if existingConfig != "" {
		printExistingConfigBanner(existingConfig, fmt.Sprintf(`
- "complexity": ["error", %d]
- "max-params": ["error", %d]
- "max-lines-per-function": ["error", { "max": %d }]
- "max-lines": ["error", { "max": %d }]`, BaselineComplexity, BaselineArgumentCount, BaselineFunctionLength, BaselineFileLength))
	} else {
		eslintPath := filepath.Join(absPath, "eslint.config.mjs")
		content := fmt.Sprintf(eslintFlatTemplate, BaselineComplexity, BaselineArgumentCount, BaselineFunctionLength, BaselineFileLength)
		if err := os.WriteFile(eslintPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write eslint.config.mjs: %w", err)
		}
		fmt.Fprintf(os.Stderr, "- [CREATED] eslint.config.mjs (Pristine Maintainability Rule Suite)\n\n")
	}
	printInstallerInstructions("tsjs")
	return nil
}

func bootstrapPython(absPath string) error {
	pylintPath := filepath.Join(absPath, ".pylintrc")
	if _, err := os.Stat(pylintPath); err == nil {
		printExistingConfigBanner(".pylintrc", fmt.Sprintf(`
- [DESIGN]
  max-args=%d
  max-statements=%d
  max-complexity=%d`, BaselineArgumentCount, BaselineFunctionLength, BaselineComplexity))
	} else {
		if err := os.WriteFile(pylintPath, []byte(fmt.Sprintf(pylintTemplate, BaselineArgumentCount, BaselineFunctionLength, BaselineComplexity, BaselineFileLength)), 0644); err != nil {
			return fmt.Errorf("failed to write .pylintrc: %w", err)
		}
		fmt.Fprintf(os.Stderr, "- [CREATED] .pylintrc (Pristine McCabe / PyLint Complexity Rules)\n\n")
	}
	printInstallerInstructions("python")
	return nil
}

func bootstrapGo(absPath string) error {
	gociPath := filepath.Join(absPath, ".golangci.yml")
	if _, err := os.Stat(gociPath); err == nil {
		printExistingConfigBanner(".golangci.yml", fmt.Sprintf(`
- gocognit: { min-complexity: %d }
- funlen: { lines: %d }
- gocyclo: { min-complexity: %d }
- revive: { argument-limit: %d }`, BaselineComplexity, BaselineFunctionLength, BaselineComplexity, BaselineArgumentCount))
	} else {
		if err := os.WriteFile(gociPath, []byte(fmt.Sprintf(golangciTemplate, BaselineComplexity, BaselineFunctionLength, BaselineComplexity, BaselineArgumentCount)), 0644); err != nil {
			return fmt.Errorf("failed to write .golangci.yml: %w", err)
		}
		fmt.Fprintf(os.Stderr, "- [CREATED] .golangci.yml (Pristine Go Vet / Gocognit Complexity Rules)\n\n")
	}
	printInstallerInstructions("go")
	return nil
}

func bootstrapJava(absPath string) error {
	checkPath := filepath.Join(absPath, "checkstyle.xml")
	if _, err := os.Stat(checkPath); err == nil {
		printExistingConfigBanner("checkstyle.xml", fmt.Sprintf(`
- <module name="CyclomaticComplexity"> <property name="max" value="%d"/> </module>
- <module name="ParameterNumber"> <property name="max" value="%d"/> </module>
- <module name="MethodLength"> <property name="max" value="%d"/> </module>`, BaselineComplexity, BaselineArgumentCount, BaselineFunctionLength))
	} else {
		if err := os.WriteFile(checkPath, []byte(fmt.Sprintf(checkstyleTemplate, BaselineComplexity, BaselineComplexity, BaselineArgumentCount, BaselineArgumentCount, BaselineFunctionLength, BaselineFunctionLength, BaselineFileLength, BaselineFileLength)), 0644); err != nil {
			return fmt.Errorf("failed to write checkstyle.xml: %w", err)
		}
		fmt.Fprintf(os.Stderr, "- [CREATED] checkstyle.xml (Pristine Java Checkstyle Complexity Rules)\n\n")
	}
	printInstallerInstructions("java")
	return nil
}

func bootstrapRuby(absPath string) error {
	ruboPath := filepath.Join(absPath, ".rubocop.yml")
	if _, err := os.Stat(ruboPath); err == nil {
		printExistingConfigBanner(".rubocop.yml", fmt.Sprintf(`
- Metrics/CyclomaticComplexity: { Max: %d }
- Metrics/MethodLength: { Max: %d }
- Metrics/ParameterLists: { Max: %d }`, BaselineComplexity, BaselineFunctionLength, BaselineArgumentCount))
	} else {
		if err := os.WriteFile(ruboPath, []byte(fmt.Sprintf(rubocopTemplate, BaselineComplexity, BaselineFunctionLength, BaselineArgumentCount, BaselineFileLength)), 0644); err != nil {
			return fmt.Errorf("failed to write .rubocop.yml: %w", err)
		}
		fmt.Fprintf(os.Stderr, "- [CREATED] .rubocop.yml (Pristine Ruby RuboCop Complexity Rules)\n\n")
	}
	printInstallerInstructions("ruby")
	return nil
}

func bootstrapCSharp(absPath string) error {
	editorPath := filepath.Join(absPath, ".editorconfig")
	if _, err := os.Stat(editorPath); err == nil {
		printExistingConfigBanner(".editorconfig", fmt.Sprintf(`
- dotnet_code_quality.CA1502.maximum_cyclomatic_complexity = %d
- dotnet_diagnostic.CA1502.severity = warning`, BaselineComplexity))
	} else {
		if err := os.WriteFile(editorPath, []byte(fmt.Sprintf(editorconfigTemplate, BaselineComplexity, BaselineComplexity)), 0644); err != nil {
			return fmt.Errorf("failed to write .editorconfig: %w", err)
		}
		fmt.Fprintf(os.Stderr, "- [CREATED] .editorconfig (Pristine Microsoft C# EditorConfig Analyzers)\n\n")
	}
	printInstallerInstructions("csharp")
	return nil
}

func detectLanguages(dirPath string) []string {
	counts := map[string]int{
		"tsjs":   0,
		"python": 0,
		"go":     0,
		"java":   0,
		"ruby":   0,
		"csharp": 0,
	}

	err := filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		// Skip standard node_modules / git directories
		if strings.Contains(path, "node_modules") || strings.Contains(path, ".git") || strings.Contains(path, "vendor") {
			return nil
		}
		ext := filepath.Ext(path)
		switch ext {
		case ".ts", ".tsx", ".js", ".jsx":
			counts["tsjs"]++
		case ".py":
			counts["python"]++
		case ".go":
			counts["go"]++
		case ".java":
			counts["java"]++
		case ".rb":
			counts["ruby"]++
		case ".cs":
			counts["csharp"]++
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to walk directory during language detection: %v\n", err)
	}

	var langs []string
	for k, v := range counts {
		if v >= 1 {
			langs = append(langs, k)
		}
	}

	return langs
}

func getFriendlyLangName(lang string) string {
	switch lang {
	case "tsjs":
		return "TypeScript / JavaScript (NextJS, React, NodeJS)"
	case "python":
		return "Python (FastAPI, Django, Flask)"
	case "go":
		return "Go (Standard modules)"
	case "java":
		return "Java (Spring Boot, Spring framework)"
	case "ruby":
		return "Ruby (Ruby on Rails, Sinatra)"
	case "csharp":
		return "C# (.NET Core, ASP.NET)"
	}
	return "Unknown"
}

func printExistingConfigBanner(fileName string, recommendations string) {
	fmt.Fprintf(os.Stderr, "- [SKIP] '%s' already exists in repository root. Protecting existing setup.\n", fileName)
	fmt.Fprintf(os.Stderr, "  >>> RECOMMENDATION: Manually integrate the following parameters into your custom configuration:\n%s\n\n", recommendations)
}

func printInstallerInstructions(lang string) {
	fmt.Fprintf(os.Stderr, "-----------------------------------------\n")
	fmt.Fprintf(os.Stderr, " Next Steps: Install Required Local Tools\n")
	fmt.Fprintf(os.Stderr, "-----------------------------------------\n")

	switch lang {
	case "tsjs":
		printTSJSInstaller()
	case "python":
		printPythonInstaller()
	case "go":
		printGoInstaller()
	case "java":
		printJavaInstaller()
	case "ruby":
		printRubyInstaller()
	case "csharp":
		printCSharpInstaller()
	}
	fmt.Fprintf(os.Stderr, "\nOnce installed, run maintainability-sensors again to activate precise Level 1+ analysis!\n")
}

func printTSJSInstaller() {
	fmt.Fprintf(os.Stderr, "Execute this command to install the required development engines:\n")
	fmt.Fprintf(os.Stderr, "  npm install --save-dev eslint @typescript-eslint/parser @typescript-eslint/eslint-plugin\n\n")
	fmt.Fprintf(os.Stderr, "Or for Yarn / PNPM:\n")
	fmt.Fprintf(os.Stderr, "  pnpm add -D eslint @typescript-eslint/parser @typescript-eslint/eslint-plugin\n")
}

func printPythonInstaller() {
	fmt.Fprintf(os.Stderr, "Execute this command to install the required PyLint engine:\n")
	fmt.Fprintf(os.Stderr, "  pip install pylint\n\n")
	fmt.Fprintf(os.Stderr, "To run McCabe cyclomatic checks with pylint:\n")
	fmt.Fprintf(os.Stderr, "  pylint --load-plugins=pylint.extensions.mccabe your_code_directory/\n")
}

func printGoInstaller() {
	fmt.Fprintf(os.Stderr, "Execute this command to install the golangci-lint meta-linter:\n")
	fmt.Fprintf(os.Stderr, "  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.60.0\n\n")
	fmt.Fprintf(os.Stderr, "Run checks with:\n")
	fmt.Fprintf(os.Stderr, "  golangci-lint run ./...\n")
}

func printJavaInstaller() {
	fmt.Fprintf(os.Stderr, "To run Java Checkstyle checks, add the checkstyle-plugin to your Maven pom.xml or Gradle build script:\n\n")
	fmt.Fprintf(os.Stderr, "Maven pom.xml Configuration:\n")
	fmt.Fprintf(os.Stderr, "  <plugin>\n")
	fmt.Fprintf(os.Stderr, "    <groupId>org.apache.maven.plugins</groupId>\n")
	fmt.Fprintf(os.Stderr, "    <artifactId>maven-checkstyle-plugin</artifactId>\n")
	fmt.Fprintf(os.Stderr, "    <version>3.3.1</version>\n")
	fmt.Fprintf(os.Stderr, "    <configuration>\n")
	fmt.Fprintf(os.Stderr, "      <configLocation>checkstyle.xml</configLocation>\n")
	fmt.Fprintf(os.Stderr, "    </configuration>\n")
	fmt.Fprintf(os.Stderr, "  </plugin>\n")
}

func printRubyInstaller() {
	fmt.Fprintf(os.Stderr, "Execute this command to install the RuboCop engine:\n")
	fmt.Fprintf(os.Stderr, "  gem install rubocop\n\n")
	fmt.Fprintf(os.Stderr, "To run checks natively:\n")
	fmt.Fprintf(os.Stderr, "  rubocop --format json your_code_directory/\n")
}

func printCSharpInstaller() {
	fmt.Fprintf(os.Stderr, "Microsoft C# Analyzers are built natively into the .NET SDK.\n")
	fmt.Fprintf(os.Stderr, "To verify code formatting and analyzer rules, run standard .NET commands:\n\n")
	fmt.Fprintf(os.Stderr, "Run static code analysis:\n")
	fmt.Fprintf(os.Stderr, "  dotnet build /p:TreatWarningsAsErrors=true\n\n")
	fmt.Fprintf(os.Stderr, "Or run automatic formatting verification:\n")
	fmt.Fprintf(os.Stderr, "  dotnet format --verify-no-changes\n")
}
