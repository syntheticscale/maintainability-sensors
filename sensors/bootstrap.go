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
    "complexity": ["error", 8],
    "max-params": ["error", 4],
    "max-lines-per-function": ["error", { "max": 50, "skipBlankLines": true, "skipComments": true }],
    "max-lines": ["error", { "max": 300, "skipBlankLines": true, "skipComments": true }],
    "@typescript-eslint/no-explicit-any": "warn"
  }
}
`

	pylintTemplate = `[MASTER]
load-plugins=pylint.extensions.mccabe

[DESIGN]
max-args=4
max-statements=50
max-complexity=8
max-module-lines=300
`

	golangciTemplate = `run:
  timeout: 5m

linters-settings:
  gocognit:
    min-complexity: 8
  funlen:
    lines: 50
    statements: 40
  gocyclo:
    min-complexity: 8
  lll:
    line-length: 120

linters:
  enable:
    - gocognit
    - funlen
    - gocyclo
    - lll
`

	checkstyleTemplate = `<?xml version="1.0"?>
<!DOCTYPE module PUBLIC
          "-//Checkstyle//DTD Checkstyle Configuration 1.3//EN"
          "https://checkstyle.org/dtds/configuration_1_3.dtd">
<module name="Checker">
  <property name="severity" value="warning"/>
  <module name="TreeWalker">
    <!-- Cyclomatic Complexity Limit: max 8 -->
    <module name="CyclomaticComplexity">
      <property name="max" value="8"/>
    </module>
    <!-- Method Parameter (Argument) Count Limit: max 4 -->
    <module name="ParameterNumber">
      <property name="max" value="4"/>
    </module>
    <!-- Method Length (Function Length) Limit: max 50 lines -->
    <module name="MethodLength">
      <property name="max" value="50"/>
      <property name="countEmpty" value="false"/>
    </module>
    <!-- File Length Limit: max 300 lines -->
    <module name="FileLength">
      <property name="max" value="300"/>
    </module>
  </module>
</module>
`
)

// BootstrapRepo detects the primary language/framework of a repository
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

	// 1. Detect primary codebase language by counting file extensions
	lang := detectPrimaryLanguage(absPath)
	if lang == "" {
		return fmt.Errorf("no supported codebase language detected (TS/JS, Python, Go, Java) in directory: %s", absPath)
	}

	fmt.Printf("=========================================\n")
	fmt.Printf(" Orchestrating Bootstrap for %s...\n", getFriendlyLangName(lang))
	fmt.Printf("=========================================\n\n")

	// 2. Based on detected language, check config, write template, or output instructions
	switch lang {
	case "tsjs":
		eslintPath := filepath.Join(absPath, ".eslintrc.json")
		if _, err := os.Stat(eslintPath); err == nil {
			printExistingConfigBanner(".eslintrc.json", `
- "complexity": ["error", 8]
- "max-params": ["error", 4]
- "max-lines-per-function": ["error", { "max": 50 }]
- "max-lines": ["error", { "max": 300 }]`)
		} else {
			if err := os.WriteFile(eslintPath, []byte(eslintTemplate), 0644); err != nil {
				return fmt.Errorf("failed to write .eslintrc.json: %w", err)
			}
			fmt.Printf("- [CREATED] .eslintrc.json (Pristine Maintainability Rule Suite)\n\n")
		}
		printInstallerInstructions("tsjs")

	case "python":
		pylintPath := filepath.Join(absPath, ".pylintrc")
		if _, err := os.Stat(pylintPath); err == nil {
			printExistingConfigBanner(".pylintrc", `
- [DESIGN]
  max-args=4
  max-statements=50
  max-complexity=8`)
		} else {
			if err := os.WriteFile(pylintPath, []byte(pylintTemplate), 0644); err != nil {
				return fmt.Errorf("failed to write .pylintrc: %w", err)
			}
			fmt.Printf("- [CREATED] .pylintrc (Pristine McCabe / PyLint Complexity Rules)\n\n")
		}
		printInstallerInstructions("python")

	case "go":
		gociPath := filepath.Join(absPath, ".golangci.yml")
		if _, err := os.Stat(gociPath); err == nil {
			printExistingConfigBanner(".golangci.yml", `
- gocognit: { min-complexity: 8 }
- funlen: { lines: 50 }
- gocyclo: { min-complexity: 8 }`)
		} else {
			if err := os.WriteFile(gociPath, []byte(golangciTemplate), 0644); err != nil {
				return fmt.Errorf("failed to write .golangci.yml: %w", err)
			}
			fmt.Printf("- [CREATED] .golangci.yml (Pristine Go Vet / Gocognit Complexity Rules)\n\n")
		}
		printInstallerInstructions("go")

	case "java":
		checkPath := filepath.Join(absPath, "checkstyle.xml")
		if _, err := os.Stat(checkPath); err == nil {
			printExistingConfigBanner("checkstyle.xml", `
- <module name="CyclomaticComplexity"> <property name="max" value="8"/> </module>
- <module name="ParameterNumber"> <property name="max" value="4"/> </module>
- <module name="MethodLength"> <property name="max" value="50"/> </module>`)
		} else {
			if err := os.WriteFile(checkPath, []byte(checkstyleTemplate), 0644); err != nil {
				return fmt.Errorf("failed to write checkstyle.xml: %w", err)
			}
			fmt.Printf("- [CREATED] checkstyle.xml (Pristine Java Checkstyle Complexity Rules)\n\n")
		}
		printInstallerInstructions("java")
	}

	return nil
}

func detectPrimaryLanguage(dirPath string) string {
	counts := map[string]int{
		"tsjs":   0,
		"python": 0,
		"go":     0,
		"java":   0,
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
		}
		return nil
	})

	maxCount := 0
	lang := ""
	for k, v := range counts {
		if v > maxCount {
			maxCount = v
			lang = k
		}
	}

	if maxCount > 0 {
		return lang
	}
	return ""
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
	}
	fmt.Printf("\nOnce installed, run maintainability-sensors again to activate precise Level 1+ analysis!\n")
}
