package sensors

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	eslintTemplate = `{
  "parser": "@typescript-eslint/parser",
  "plugins": ["@typescript-eslint"],
  "rules": {
    "complexity": ["error", %d],
    "max-params": ["error", %d],
    "max-lines-per-function": ["error", { "max": %d, "skipBlankLines": true, "skipComments": true }],
    "max-lines": ["error", { "max": %d, "skipBlankLines": true, "skipComments": true }],
    "@typescript-eslint/no-explicit-any": "warn"
  }
}
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

linters:
  enable:
    - gocognit
    - funlen
    - cyclop
    - lll
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
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		absPath = repoPath
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("target path does not exist: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("target path is not a directory: %s", absPath)
	}

	// 1. Detect codebase languages by counting file extensions
	langs := detectLanguages(absPath)
	if len(langs) == 0 {
		return fmt.Errorf("no supported codebase language detected (TS/JS, Python, Go, Java) in directory: %s", absPath)
	}

	for _, lang := range langs {
		fmt.Printf("=========================================\n")
		fmt.Printf(" Orchestrating Bootstrap for %s...\n", getFriendlyLangName(lang))
		fmt.Printf("=========================================\n\n")

		if err := bootstrapLanguage(lang, absPath); err != nil {
			return err
		}
	}

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

func bootstrapTSJS(absPath string) error {
	eslintPath := filepath.Join(absPath, ".eslintrc.json")
	if _, err := os.Stat(eslintPath); err == nil {
		printExistingConfigBanner(".eslintrc.json", fmt.Sprintf(`
- "complexity": ["error", %d]
- "max-params": ["error", %d]
- "max-lines-per-function": ["error", { "max": %d }]
- "max-lines": ["error", { "max": %d }]`, BaselineComplexity, BaselineArgumentCount, BaselineFunctionLength, BaselineFileLength))
	} else {
		if err := os.WriteFile(eslintPath, []byte(fmt.Sprintf(eslintTemplate, BaselineComplexity, BaselineArgumentCount, BaselineFunctionLength, BaselineFileLength)), 0644); err != nil {
			return fmt.Errorf("failed to write .eslintrc.json: %w", err)
		}
		fmt.Printf("- [CREATED] .eslintrc.json (Pristine Maintainability Rule Suite)\n\n")
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
		fmt.Printf("- [CREATED] .pylintrc (Pristine McCabe / PyLint Complexity Rules)\n\n")
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
- gocyclo: { min-complexity: %d }`, BaselineComplexity, BaselineFunctionLength, BaselineComplexity))
	} else {
		if err := os.WriteFile(gociPath, []byte(fmt.Sprintf(golangciTemplate, BaselineComplexity, BaselineFunctionLength, BaselineComplexity)), 0644); err != nil {
			return fmt.Errorf("failed to write .golangci.yml: %w", err)
		}
		fmt.Printf("- [CREATED] .golangci.yml (Pristine Go Vet / Gocognit Complexity Rules)\n\n")
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
		fmt.Printf("- [CREATED] checkstyle.xml (Pristine Java Checkstyle Complexity Rules)\n\n")
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
		fmt.Printf("- [CREATED] .rubocop.yml (Pristine Ruby RuboCop Complexity Rules)\n\n")
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
		fmt.Printf("- [CREATED] .editorconfig (Pristine Microsoft C# EditorConfig Analyzers)\n\n")
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

	_ = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
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
	fmt.Printf("- [SKIP] '%s' already exists in repository root. Protecting existing setup.\n", fileName)
	fmt.Printf("  >>> RECOMMENDATION: Manually integrate the following parameters into your custom configuration:\n%s\n\n", recommendations)
}

func printInstallerInstructions(lang string) {
	fmt.Printf("-----------------------------------------\n")
	fmt.Printf(" Next Steps: Install Required Local Tools\n")
	fmt.Printf("-----------------------------------------\n")

	switch lang {
	case "tsjs":
		fmt.Printf("Execute this command to install the required development engines:\n")
		fmt.Printf("  npm install --save-dev eslint @typescript-eslint/parser @typescript-eslint/eslint-plugin\n\n")
		fmt.Printf("Or for Yarn / PNPM:\n")
		fmt.Printf("  pnpm add -D eslint @typescript-eslint/parser @typescript-eslint/eslint-plugin\n")
	case "python":
		fmt.Printf("Execute this command to install the required PyLint engine:\n")
		fmt.Printf("  pip install pylint\n\n")
		fmt.Printf("To run McCabe cyclomatic checks with pylint:\n")
		fmt.Printf("  pylint --load-plugins=pylint.extensions.mccabe your_code_directory/\n")
	case "go":
		fmt.Printf("Execute this command to install the golangci-lint meta-linter:\n")
		fmt.Printf("  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.60.0\n\n")
		fmt.Printf("Run checks with:\n")
		fmt.Printf("  golangci-lint run ./...\n")
	case "java":
		fmt.Printf("To run Java Checkstyle checks, add the checkstyle-plugin to your Maven pom.xml or Gradle build script:\n\n")
		fmt.Printf("Maven pom.xml Configuration:\n")
		fmt.Printf("  <plugin>\n")
		fmt.Printf("    <groupId>org.apache.maven.plugins</groupId>\n")
		fmt.Printf("    <artifactId>maven-checkstyle-plugin</artifactId>\n")
		fmt.Printf("    <version>3.3.1</version>\n")
		fmt.Printf("    <configuration>\n")
		fmt.Printf("      <configLocation>checkstyle.xml</configLocation>\n")
		fmt.Printf("    </configuration>\n")
		fmt.Printf("  </plugin>\n")
	case "ruby":
		fmt.Printf("Execute this command to install the RuboCop engine:\n")
		fmt.Printf("  gem install rubocop\n\n")
		fmt.Printf("To run checks natively:\n")
		fmt.Printf("  rubocop --format json your_code_directory/\n")
	case "csharp":
		fmt.Printf("Microsoft C# Analyzers are built natively into the .NET SDK.\n")
		fmt.Printf("To verify code formatting and analyzer rules, run standard .NET commands:\n\n")
		fmt.Printf("Run static code analysis:\n")
		fmt.Printf("  dotnet build /p:TreatWarningsAsErrors=true\n\n")
		fmt.Printf("Or run automatic formatting verification:\n")
		fmt.Printf("  dotnet format --verify-no-changes\n")
	}
	fmt.Printf("\nOnce installed, run maintainability-sensors again to activate precise Level 1+ analysis!\n")
}

