package sensors

//nolint // maintainability: highly cohesive test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	cases := []struct {
		path     string
		expected string
	}{
		{"/project/src/index.ts", "typescript"},
		{"/project/src/index.tsx", "typescript"},
		{"/project/src/app.js", "javascript"},
		{"/project/src/app.jsx", "javascript"},
		{"/project/src/main.py", "python"},
		{"/project/src/main.go", "go"},
		{"/project/src/app.rb", "ruby"},
		{"/project/src/App.java", "java"},
		{"/project/src/Program.cs", "csharp"},
		{"/project/README.md", ""},
		{"/project/Cargo.toml", ""},
		{"/project/src/main", ""},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			got := DetectLanguage(tc.path)
			if got != tc.expected {
				t.Errorf("DetectLanguage(%q) = %q, want %q", tc.path, got, tc.expected)
			}
		})
	}
}

func TestDetectConfigAndParser_FindsConfigInSameDir(t *testing.T) {
	tempDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tempDir, ".eslintrc.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	filePath := filepath.Join(tempDir, "index.ts")
	if err := os.WriteFile(filePath, []byte("const x = 1;\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	anchor, parser := DetectConfigAndParser(filePath, "typescript")
	if anchor == "" {
		t.Fatal("expected to find config anchor, got empty string")
	}
	if parser == nil {
		t.Fatal("expected non-nil parser")
	}
	if parser.Name() != "eslint" && parser.Name() != "biome" {
		t.Errorf("expected eslint or biome parser, got %q", parser.Name())
	}
}

func TestDetectConfigAndParser_WalksUpParentDirs(t *testing.T) {
	tempDir := t.TempDir()
	parentDir := filepath.Join(tempDir, "parent")
	subDir := filepath.Join(parentDir, "child")

	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	configPath := filepath.Join(parentDir, ".pylintrc")
	if err := os.WriteFile(configPath, []byte("[DESIGN]\n"), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	filePath := filepath.Join(subDir, "app.py")
	if err := os.WriteFile(filePath, []byte("print(1)\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	anchor, parser := DetectConfigAndParser(filePath, "python")
	if anchor != configPath {
		t.Errorf("DetectConfigAndParser(%q) = %q, want %q", filePath, anchor, configPath)
	}
	if parser == nil {
		t.Fatal("expected non-nil parser")
	}
	if parser.Name() != "ruff" && parser.Name() != "pylint" {
		t.Errorf("expected ruff or pylint parser, got %q", parser.Name())
	}
}

func TestDetectConfigAndParser_NoConfig(t *testing.T) {
	tempDir := t.TempDir()

	filePath := filepath.Join(tempDir, "index.ts")
	if err := os.WriteFile(filePath, []byte("const x = 1;\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	anchor, parser := DetectConfigAndParser(filePath, "typescript")
	if anchor != "" {
		t.Errorf("expected empty anchor, got %q", anchor)
	}
	if parser != nil {
		t.Error("expected nil parser when no config found")
	}
}

func TestDetectConfigAndParser_UnsupportedLang(t *testing.T) {
	tempDir := t.TempDir()

	filePath := filepath.Join(tempDir, "main.rs")
	if err := os.WriteFile(filePath, []byte("fn main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	anchor, parser := DetectConfigAndParser(filePath, "rust")
	if anchor != "" {
		t.Errorf("expected empty anchor for unsupported lang, got %q", anchor)
	}
	if parser != nil {
		t.Error("expected nil parser for unsupported lang")
	}
}

func TestDetectConfigAndParser_GoConfig(t *testing.T) {
	tempDir := t.TempDir()

	configPath := filepath.Join(tempDir, ".golangci.yml")
	if err := os.WriteFile(configPath, []byte("linters-settings:\n"), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	filePath := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	anchor, parser := DetectConfigAndParser(filePath, "go")
	if anchor != configPath {
		t.Errorf("DetectConfigAndParser(%q) = %q, want %q", filePath, anchor, configPath)
	}
	if parser == nil {
		t.Fatal("expected non-nil parser")
	}
	if parser.Name() != "golangci" {
		t.Errorf("expected golangci parser, got %q", parser.Name())
	}
}

func TestDetectConfigAndParser_RubyConfig(t *testing.T) {
	tempDir := t.TempDir()

	configPath := filepath.Join(tempDir, ".rubocop.yml")
	if err := os.WriteFile(configPath, []byte("Metrics/MethodLength:\n  Max: 30\n"), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	filePath := filepath.Join(tempDir, "app.rb")
	if err := os.WriteFile(filePath, []byte("def hello\nend\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	anchor, parser := DetectConfigAndParser(filePath, "ruby")
	if anchor != configPath {
		t.Errorf("DetectConfigAndParser(%q) = %q, want %q", filePath, anchor, configPath)
	}
	if parser == nil {
		t.Fatal("expected non-nil parser")
	}
	if parser.Name() != "standardrb" && parser.Name() != "rubocop" {
		t.Errorf("expected standardrb or rubocop parser, got %q", parser.Name())
	}
}
