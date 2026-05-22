package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/syntheticscale/maintainability-sensors/sensors"
	"gopkg.in/yaml.v3"
)

// Severity represents the severity level for a rule.
type Severity string

const (
	SeverityError  Severity = "error"
	SeverityWarn   Severity = "warn"
	SeverityIgnore Severity = "ignore"
)

// Valid severities for validation.
var validSeverities = map[Severity]bool{
	SeverityError:  true,
	SeverityWarn:   true,
	SeverityIgnore: true,
}

// Valid rule names for validation.
var validRuleNames = map[string]bool{
	"Complexity":     true,
	"FunctionLength": true,
	"ArgumentCount":  true,
}

// RulePolicy holds the configuration for a single rule.
type RulePolicy struct {
	Name      string
	Severity  Severity
	Threshold *int // nil means "use baseline constant"
}

// CheckDiffPolicy holds the resolved policy for check-diff.
type CheckDiffPolicy struct {
	DefaultSeverity Severity
	Rules           map[string]RulePolicy
}

// CheckDiffConfigFile represents the YAML structure for .maintainability-sensors.yml.
type CheckDiffConfigFile struct {
	Version   string `yaml:"version"`
	CheckDiff struct {
		DefaultSeverity string `yaml:"default-severity"`
		Rules           []struct {
			Name      string `yaml:"name"`
			Severity  string `yaml:"severity"`
			Threshold *int   `yaml:"threshold"`
		} `yaml:"rules"`
	} `yaml:"check-diff"`
}

// findConfigFile searches for .maintainability-sensors.yml in the target directory.
func findConfigFile(targetPath string) string {
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return ""
	}

	var checkPaths []string
	info, err := os.Stat(absPath)
	if err == nil && !info.IsDir() {
		checkPaths = append(checkPaths, filepath.Dir(absPath))
	} else {
		checkPaths = append(checkPaths, absPath)
	}

	for _, dir := range checkPaths {
		candidate := filepath.Join(dir, ".maintainability-sensors.yml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

// LoadPolicy loads the check-diff policy from config file and CLI flags.
// CLI flags take precedence over config file values.
//
// priority (highest to lowest):
// 1. --severity flag overrides
// 2. --default-severity flag
// 3. Config file values
// 4. Built-in default (error)
func LoadPolicy(configPath string, defaultSeverity string, severityOverrides []string) (*CheckDiffPolicy, error) {
	// Start with built-in defaults (backwards compatible)
	policy := &CheckDiffPolicy{
		DefaultSeverity: SeverityError,
		Rules:           make(map[string]RulePolicy),
	}

	if err := applyConfigFile(policy, configPath); err != nil {
		return nil, err
	}

	if err := applyCLIDefaultSeverity(policy, defaultSeverity); err != nil {
		return nil, err
	}

	if err := applyCLISeverityOverrides(policy, severityOverrides); err != nil {
		return nil, err
	}

	return policy, nil
}

func applyConfigFile(policy *CheckDiffPolicy, configPath string) error {
	if configPath == "" {
		return nil
	}
	if _, err := os.Stat(configPath); err != nil {
		return nil
	}

	config, err := loadConfigFile(configPath)
	if err != nil {
		return err
	}

	if err := applyConfigDefaultSeverity(policy, config); err != nil {
		return err
	}

	return applyConfigRules(policy, config)
}

func applyConfigDefaultSeverity(policy *CheckDiffPolicy, config *CheckDiffConfigFile) error {
	if config.CheckDiff.DefaultSeverity != "" {
		sev := Severity(config.CheckDiff.DefaultSeverity)
		if !isValidSeverity(sev) {
			return fmt.Errorf("invalid default-severity %q in config file, expected error, warn, or ignore", config.CheckDiff.DefaultSeverity)
		}
		policy.DefaultSeverity = sev
	}
	return nil
}

func applyConfigRules(policy *CheckDiffPolicy, config *CheckDiffConfigFile) error {
	for _, rule := range config.CheckDiff.Rules {
		if !isValidRuleName(rule.Name) {
			return fmt.Errorf("invalid rule name %q in config file, expected Complexity, FunctionLength, or ArgumentCount", rule.Name)
		}
		sev := policy.DefaultSeverity
		if rule.Severity != "" {
			sev = Severity(rule.Severity)
			if !isValidSeverity(sev) {
				return fmt.Errorf("invalid severity %q for rule %s in config file, expected error, warn, or ignore", rule.Severity, rule.Name)
			}
		}
		policy.Rules[rule.Name] = RulePolicy{
			Name:      rule.Name,
			Severity:  sev,
			Threshold: rule.Threshold,
		}
	}
	return nil
}

func applyCLIDefaultSeverity(policy *CheckDiffPolicy, defaultSeverity string) error {
	if defaultSeverity != "" {
		sev := Severity(defaultSeverity)
		if !isValidSeverity(sev) {
			return fmt.Errorf("invalid --default-severity %q, expected error, warn, or ignore", defaultSeverity)
		}
		policy.DefaultSeverity = sev
	}
	return nil
}

func applyCLISeverityOverrides(policy *CheckDiffPolicy, severityOverrides []string) error {
	for _, override := range severityOverrides {
		parts := strings.SplitN(override, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid --severity format %q, expected Rule:level", override)
		}
		name, sevStr := parts[0], parts[1]
		if !isValidRuleName(name) {
			return fmt.Errorf("invalid rule name %q in --severity, expected Complexity, FunctionLength, or ArgumentCount", name)
		}
		sev := Severity(sevStr)
		if !isValidSeverity(sev) {
			return fmt.Errorf("invalid severity %q for rule %s in --severity, expected error, warn, or ignore", sevStr, name)
		}

		// Preserve existing threshold if rule already exists in policy.
		existing := policy.Rules[name]
		policy.Rules[name] = RulePolicy{
			Name:      name,
			Severity:  sev,
			Threshold: existing.Threshold,
		}
	}
	return nil
}

// loadConfigFile reads and parses the YAML config file.
func loadConfigFile(path string) (*CheckDiffConfigFile, error) {
	if info, err := os.Stat(path); err == nil && (!info.Mode().IsRegular() || info.Size() > 2*1024*1024) {
		return nil, fmt.Errorf("config file %s is too large or not a regular file", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var config CheckDiffConfigFile
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	return &config, nil
}

// isValidSeverity checks if a severity string is valid.
func isValidSeverity(s Severity) bool {
	return validSeverities[s]
}

// isValidRuleName checks if a rule name is valid.
func isValidRuleName(name string) bool {
	return validRuleNames[name]
}

// getBaselineForRule returns the baseline threshold for a given rule name.
func getBaselineForRule(ruleName string) int {
	switch ruleName {
	case "Complexity":
		return sensors.BaselineComplexity
	case "FunctionLength":
		return sensors.BaselineFunctionLength
	case "ArgumentCount":
		return sensors.BaselineArgumentCount
	default:
		return 0
	}
}

// getSeverityForRule returns the effective severity for a violation.
func getSeverityForRule(policy *CheckDiffPolicy, ruleName string) Severity {
	if policy == nil {
		return SeverityError
	}
	if rulePolicy, ok := policy.Rules[ruleName]; ok {
		return rulePolicy.Severity
	}
	return policy.DefaultSeverity
}

// getThresholdForRule returns the effective threshold for a rule, or baseline if not overridden.
func getThresholdForRule(policy *CheckDiffPolicy, ruleName string) int {
	if policy != nil {
		if rulePolicy, ok := policy.Rules[ruleName]; ok && rulePolicy.Threshold != nil {
			return *rulePolicy.Threshold
		}
	}
	return getBaselineForRule(ruleName)
}
