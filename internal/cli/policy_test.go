package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPolicy_NoConfigNoFlags(t *testing.T) {
	policy, err := LoadPolicy("", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.DefaultSeverity != SeverityError {
		t.Errorf("expected default severity error, got %q", policy.DefaultSeverity)
	}
	if len(policy.Rules) != 0 {
		t.Errorf("expected no rules, got %d", len(policy.Rules))
	}
}

func TestLoadPolicy_ConfigFileDefaultSeverity(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".maintainability-sensors.yml")

	configContent := `version: "1"
check-diff:
  default-severity: warn
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	policy, err := LoadPolicy(configPath, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.DefaultSeverity != SeverityWarn {
		t.Errorf("expected default severity warn, got %q", policy.DefaultSeverity)
	}
}

func TestLoadPolicy_UIDefaultSeverityOverridesConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".maintainability-sensors.yml")

	configContent := `version: "1"
check-diff:
  default-severity: warn
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	policy, err := LoadPolicy(configPath, "error", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.DefaultSeverity != SeverityError {
		t.Errorf("expected default severity error (CLI overrides config), got %q", policy.DefaultSeverity)
	}
}

func TestLoadPolicy_SeverityFlagOverridesSpecificRule(t *testing.T) {
	policy, err := LoadPolicy("", "error", []string{"Complexity:warn"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.DefaultSeverity != SeverityError {
		t.Errorf("expected default severity error, got %q", policy.DefaultSeverity)
	}
	if rule, ok := policy.Rules["Complexity"]; !ok {
		t.Errorf("expected Complexity rule in policy")
	} else if rule.Severity != SeverityWarn {
		t.Errorf("expected Complexity severity warn, got %q", rule.Severity)
	}
}

func TestLoadPolicy_SeverityFlagOverridesConfigRule(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".maintainability-sensors.yml")

	configContent := `version: "1"
check-diff:
  default-severity: error
  rules:
    - name: Complexity
      severity: error
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	policy, err := LoadPolicy(configPath, "error", []string{"Complexity:warn"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule, ok := policy.Rules["Complexity"]; !ok {
		t.Errorf("expected Complexity rule in policy")
	} else if rule.Severity != SeverityWarn {
		t.Errorf("expected Complexity severity warn (CLI overrides config), got %q", rule.Severity)
	}
}

func TestLoadPolicy_ConfigWithPerRuleThreshold(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".maintainability-sensors.yml")

	configContent := `version: "1"
check-diff:
  default-severity: warn
  rules:
    - name: Complexity
      severity: error
      threshold: 12
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	policy, err := LoadPolicy(configPath, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.DefaultSeverity != SeverityWarn {
		t.Errorf("expected default severity warn, got %q", policy.DefaultSeverity)
	}
	if rule, ok := policy.Rules["Complexity"]; !ok {
		t.Errorf("expected Complexity rule in policy")
	} else if rule.Severity != SeverityError {
		t.Errorf("expected Complexity severity error, got %q", rule.Severity)
	} else if rule.Threshold == nil {
		t.Errorf("expected Complexity threshold to be set")
	} else if *rule.Threshold != 12 {
		t.Errorf("expected Complexity threshold 12, got %d", *rule.Threshold)
	}
}

func TestLoadPolicy_UnknownRuleNameInConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".maintainability-sensors.yml")

	configContent := `version: "1"
check-diff:
  rules:
    - name: InvalidRule
      severity: warn
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	_, err := LoadPolicy(configPath, "", nil)
	if err == nil {
		t.Fatal("expected error for invalid rule name")
	}
	expected := `invalid rule name "InvalidRule" in config file`
	if !containsSubstring(err.Error(), expected) {
		t.Errorf("expected error to contain %q, got: %v", expected, err)
	}
}

func TestLoadPolicy_InvalidSeverityInConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".maintainability-sensors.yml")

	configContent := `version: "1"
check-diff:
  default-severity: invalid
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	_, err := LoadPolicy(configPath, "", nil)
	if err == nil {
		t.Fatal("expected error for invalid severity")
	}
	if !containsSubstring(err.Error(), "invalid default-severity") {
		t.Errorf("expected error about invalid severity, got: %v", err)
	}
}

func TestLoadPolicy_InvalidSeverityInCLI(t *testing.T) {
	_, err := LoadPolicy("", "invalid", nil)
	if err == nil {
		t.Fatal("expected error for invalid CLI default severity")
	}
	if !containsSubstring(err.Error(), "invalid --default-severity") {
		t.Errorf("expected error about invalid --default-severity, got: %v", err)
	}
}

func TestLoadPolicy_InvalidSeverityInSeverityFlag(t *testing.T) {
	_, err := LoadPolicy("", "", []string{"Complexity:invalid"})
	if err == nil {
		t.Fatal("expected error for invalid severity in --severity flag")
	}
	if !containsSubstring(err.Error(), "invalid severity") {
		t.Errorf("expected error about invalid severity, got: %v", err)
	}
}

func TestLoadPolicy_InvalidSeverityFlagFormat(t *testing.T) {
	_, err := LoadPolicy("", "", []string{"Complexity"})
	if err == nil {
		t.Fatal("expected error for invalid --severity format")
	}
	if !containsSubstring(err.Error(), "invalid --severity format") {
		t.Errorf("expected error about invalid --severity format, got: %v", err)
	}
}

func TestLoadPolicy_InvalidRuleInSeverityFlag(t *testing.T) {
	_, err := LoadPolicy("", "", []string{"InvalidRule:warn"})
	if err == nil {
		t.Fatal("expected error for invalid rule name in --severity flag")
	}
	if !containsSubstring(err.Error(), "invalid rule name") {
		t.Errorf("expected error about invalid rule name, got: %v", err)
	}
}

func TestLoadPolicy_InvalidSeverityForRuleInConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".maintainability-sensors.yml")

	configContent := `version: "1"
check-diff:
  rules:
    - name: Complexity
      severity: invalid
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	_, err := LoadPolicy(configPath, "", nil)
	if err == nil {
		t.Fatal("expected error for invalid rule severity in config")
	}
	if !containsSubstring(err.Error(), "invalid severity") {
		t.Errorf("expected error about invalid severity, got: %v", err)
	}
}

func TestLoadPolicy_MultipleSeverityFlags(t *testing.T) {
	policy, err := LoadPolicy("", "error", []string{"Complexity:warn", "FunctionLength:ignore"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.Rules["Complexity"].Severity != SeverityWarn {
		t.Errorf("expected Complexity warn, got %q", policy.Rules["Complexity"].Severity)
	}
	if policy.Rules["FunctionLength"].Severity != SeverityIgnore {
		t.Errorf("expected FunctionLength ignore, got %q", policy.Rules["FunctionLength"].Severity)
	}
	if _, ok := policy.Rules["ArgumentCount"]; ok {
		t.Errorf("expected no ArgumentCount rule override")
	}
}

func TestLoadPolicy_ConfigPreservesThresholdOnCLISeverityOverride(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".maintainability-sensors.yml")

	configContent := `version: "1"
check-diff:
  rules:
    - name: Complexity
      severity: error
      threshold: 12
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// CLI --severity override should change severity but preserve threshold.
	policy, err := LoadPolicy(configPath, "", []string{"Complexity:warn"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rule := policy.Rules["Complexity"]
	if rule.Severity != SeverityWarn {
		t.Errorf("expected severity warn, got %q", rule.Severity)
	}
	if rule.Threshold == nil {
		t.Errorf("expected threshold preserved")
	} else if *rule.Threshold != 12 {
		t.Errorf("expected threshold 12, got %d", *rule.Threshold)
	}
}

func TestLoadPolicy_EmptyConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".maintainability-sensors.yml")

	// Empty YAML
	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	policy, err := LoadPolicy(configPath, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.DefaultSeverity != SeverityError {
		t.Errorf("expected default severity error for empty config, got %q", policy.DefaultSeverity)
	}
}

func TestLoadPolicy_DefaultSeverityExplicitError(t *testing.T) {
	// When CLI explicitly passes "error" as default-severity, it should be applied
	policy, err := LoadPolicy("", "error", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.DefaultSeverity != SeverityError {
		t.Errorf("expected default severity error, got %q", policy.DefaultSeverity)
	}
}

func TestLoadPolicy_DefaultSeverityExplicitWarn(t *testing.T) {
	policy, err := LoadPolicy("", "warn", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.DefaultSeverity != SeverityWarn {
		t.Errorf("expected default severity warn, got %q", policy.DefaultSeverity)
	}
}

func TestLoadPolicy_DefaultSeverityExplicitIgnore(t *testing.T) {
	policy, err := LoadPolicy("", "ignore", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.DefaultSeverity != SeverityIgnore {
		t.Errorf("expected default severity ignore, got %q", policy.DefaultSeverity)
	}
}

func TestLoadPolicy_ConfigRuleWithoutSeverity(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".maintainability-sensors.yml")

	configContent := `version: "1"
check-diff:
  default-severity: warn
  rules:
    - name: Complexity
      threshold: 12
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	policy, err := LoadPolicy(configPath, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rule := policy.Rules["Complexity"]
	if rule.Severity != SeverityWarn {
		// Should inherit default severity
		t.Errorf("expected severity warn (inherited from default), got %q", rule.Severity)
	}
	if rule.Threshold == nil {
		t.Errorf("expected threshold set")
	} else if *rule.Threshold != 12 {
		t.Errorf("expected threshold 12, got %d", *rule.Threshold)
	}
}

// --- Test findConfigFile ---

func TestFindConfigFile_FindsInDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".maintainability-sensors.yml")
	if err := os.WriteFile(configPath, []byte("version: \"1\""), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	found := findConfigFile(tmpDir)
	if found != configPath {
		t.Errorf("expected %q, got %q", configPath, found)
	}
}

func TestFindConfigFile_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	found := findConfigFile(tmpDir)
	if found != "" {
		t.Errorf("expected empty string, got %q", found)
	}
}

func TestFindConfigFile_FindsForFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".maintainability-sensors.yml")
	if err := os.WriteFile(configPath, []byte("version: \"1\""), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	someFile := filepath.Join(tmpDir, "main.go")
	// create the file so that os.Stat can correctly identify it as a file
	if err := os.WriteFile(someFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	found := findConfigFile(someFile)
	if found != configPath {
		t.Errorf("expected %q, got %q", configPath, found)
	}
}

// --- Test helper functions ---

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsStr(s, substr)))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
