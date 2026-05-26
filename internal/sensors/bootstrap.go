package sensors

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func BootstrapRepo(repoPath string) error {
	return BootstrapRepoWithPolicy(repoPath, false)
}

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
