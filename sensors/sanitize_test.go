package sensors

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizePath_ValidAbsolute(t *testing.T) {
	clean, err := sanitizePath("/home/user/project/main.go")
	if err != nil {
		t.Fatalf("expected no error for absolute path, got %v", err)
	}
	if clean != "/home/user/project/main.go" {
		t.Errorf("expected /home/user/project/main.go, got %s", clean)
	}
}

func TestSanitizePath_ValidRelative(t *testing.T) {
	clean, err := sanitizePath("./main.go")
	if err != nil {
		t.Fatalf("expected no error for relative path, got %v", err)
	}
	if !strings.HasSuffix(clean, "main.go") {
		t.Errorf("expected path ending with main.go, got %s", clean)
	}
}

func TestSanitizePath_ResolvesDotDots(t *testing.T) {
	clean, err := sanitizePath("foo/bar/../baz.go")
	if err != nil {
		t.Fatalf("expected no error for resolvable path, got %v", err)
	}
	if strings.Contains(clean, "..") {
		t.Errorf("expected .. to be resolved, got %s", clean)
	}
}

func TestSanitizePath_RejectsNullByte(t *testing.T) {
	_, err := sanitizePath("/tmp/\x00/etc/passwd")
	if err == nil {
		t.Fatal("expected error for path containing null byte, got nil")
	}
}

func TestSanitizePath_RejectsParentTraversalStart(t *testing.T) {
	cases := []string{
		"../main.go",
		"../../etc/passwd",
		"foo/../../main.go",
	}
	for _, p := range cases {
		_, err := sanitizePath(p)
		if err == nil {
			t.Errorf("expected error for traversal path %q, got nil", p)
		}
	}
}

func TestSanitizePath_RejectEmpty(t *testing.T) {
	clean, err := sanitizePath("")
	if err != nil {
		t.Fatalf("expected no error for empty path, got %v", err)
	}
	if clean != "." {
		t.Errorf("expected '.' for empty path, got %s", clean)
	}
}

func TestOrchestratedScan_RejectNullBytePath(t *testing.T) {
	_, err := OrchestratedScan("/tmp/\x00/passwd")
	if err == nil {
		t.Fatal("expected OrchestratedScan to reject path with null byte")
	}
}

func TestOrchestratedScan_RejectTraversalPath(t *testing.T) {
	tempDir := t.TempDir()
	_, err := OrchestratedScan(filepath.Join(tempDir, "..", "..", "etc", "passwd"))
	if err == nil {
		t.Fatal("expected OrchestratedScan to reject traversal path")
	}
}

func TestRunESLint_RejectBadPath(t *testing.T) {
	_, err := runESLint("../etc/passwd")
	if err == nil {
		t.Fatal("expected runESLint to reject traversal path")
	}
}

func TestRunPyLint_RejectBadPath(t *testing.T) {
	_, err := runPyLint("../etc/passwd")
	if err == nil {
		t.Fatal("expected runPyLint to reject traversal path")
	}
}

func TestRunRuboCop_RejectBadPath(t *testing.T) {
	_, err := runRuboCop("../etc/passwd")
	if err == nil {
		t.Fatal("expected runRuboCop to reject traversal path")
	}
}
