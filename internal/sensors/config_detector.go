package sensors

import (
	"os"
	"path/filepath"
)

func DetectLanguage(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".py":
		return "python"
	case ".go":
		return "go"
	case ".rb":
		return "ruby"
	case ".java":
		return "java"
	case ".cs":
		return "csharp"
	}
	return ""
}

func getParsersForLang(lang string) []ConfigParser {
	switch lang {
	case "typescript", "javascript":
		return []ConfigParser{BiomeConfigParser{}, ESLintConfigParser{}}
	case "python":
		return []ConfigParser{RuffConfigParser{}, PyLintConfigParser{}}
	case "go":
		return []ConfigParser{GoConfigParser{}}
	case "ruby":
		return []ConfigParser{StandardRBConfigParser{}, RuboCopConfigParser{}}
	}
	return nil
}

func findConfigInDir(absDir string, parsers []ConfigParser) (string, ConfigParser) {
	for _, parser := range parsers {
		anchors := parser.Anchors()
		for _, anchor := range anchors {
			p := filepath.Join(absDir, anchor)
			if _, err := os.Stat(p); err == nil {
				return p, parser
			}
		}
	}
	return "", nil
}

func DetectConfigAndParser(filePath string, lang string) (string, ConfigParser) {
	parsers := getParsersForLang(lang)
	if len(parsers) == 0 {
		return "", nil
	}

	dir := filepath.Dir(filePath)
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}

	for {
		if p, parser := findConfigInDir(absDir, parsers); p != "" {
			return p, parser
		}
		parent := filepath.Dir(absDir)
		if parent == absDir {
			break
		}
		absDir = parent
	}
	return "", nil
}

func findMaxConfigVal(content string, ext string, keys []string) (int, bool) {
	for _, key := range keys {
		vals := findAllConfigVals(content, key, ext)
		if len(vals) > 0 {
			return maxOf(vals), true
		}
	}
	return 0, false
}

func isValidConfigFile(configPath string) bool {
	info, err := os.Stat(configPath)
	return err == nil && info.Mode().IsRegular() && info.Size() <= 2*1024*1024
}

func DetectRelaxedLimits(configPath string, parser ConfigParser) []RelaxedLimit {
	var exceptions []RelaxedLimit
	if configPath == "" || parser == nil || !isValidConfigFile(configPath) {
		return exceptions
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return exceptions
	}
	content := string(data)
	ext := filepath.Ext(configPath)

	for _, rule := range parser.Rules() {
		if foundVal, found := findMaxConfigVal(content, ext, rule.Keys); found && foundVal > rule.Baseline {
			exceptions = append(exceptions, RelaxedLimit{
				RuleName:      rule.RuleName,
				ConfiguredVal: foundVal,
				BaselineVal:   rule.Baseline,
			})
		}
	}

	return exceptions
}
