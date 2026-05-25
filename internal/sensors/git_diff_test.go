package sensors

import (
	"bufio"
	"strings"
	"testing"
)

func TestParseGitDiffOutput_Basic(t *testing.T) {
	diff := `diff --git a/src/main.go b/src/main.go
--- a/src/main.go
+++ b/src/main.go
@@ -10,2 +10,5 @@
 func main() {
 	line1
 	line2
+	line3
+	line4
+	line5
`
	scanner := bufio.NewScanner(strings.NewReader(diff))
	result, err := parseGitDiffOutput(scanner)
	if err != nil {
		t.Fatalf("parseGitDiffOutput failed: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 file, got %d", len(result))
	}

	ranges := result["src/main.go"]
	if len(ranges) != 1 {
		t.Fatalf("expected 1 range, got %d", len(ranges))
	}
	if ranges[0].Start != 10 {
		t.Errorf("expected start 10, got %d", ranges[0].Start)
	}
	if ranges[0].End != 14 {
		t.Errorf("expected end 14, got %d", ranges[0].End)
	}
}

func TestParseGitDiffOutput_MultipleFiles(t *testing.T) {
	diff := `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,3 @@
 package main
+
+
+func A() {}
diff --git a/b.go b/b.go
--- a/b.go
+++ b/b.go
@@ -5,1 +5,2 @@
 func B() {
+	line
 }
`
	scanner := bufio.NewScanner(strings.NewReader(diff))
	result, err := parseGitDiffOutput(scanner)
	if err != nil {
		t.Fatalf("parseGitDiffOutput failed: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 files, got %d", len(result))
	}

	aRanges := result["a.go"]
	if len(aRanges) != 1 {
		t.Errorf("expected 1 range for a.go, got %d", len(aRanges))
	}
	if aRanges[0].Start != 1 || aRanges[0].End != 3 {
		t.Errorf("expected a.go range [1,3], got [%d,%d]", aRanges[0].Start, aRanges[0].End)
	}

	bRanges := result["b.go"]
	if len(bRanges) != 1 {
		t.Errorf("expected 1 range for b.go, got %d", len(bRanges))
	}
	if bRanges[0].Start != 5 || bRanges[0].End != 6 {
		t.Errorf("expected b.go range [5,6], got [%d,%d]", bRanges[0].Start, bRanges[0].End)
	}
}

func TestParseGitDiffOutput_MultipleHunks(t *testing.T) {
	diff := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,1 +1,2 @@
 package main
+
@@ -10,1 +10,2 @@
 func main() {
+
 }
`
	scanner := bufio.NewScanner(strings.NewReader(diff))
	result, err := parseGitDiffOutput(scanner)
	if err != nil {
		t.Fatalf("parseGitDiffOutput failed: %v", err)
	}

	ranges := result["main.go"]
	if len(ranges) != 2 {
		t.Fatalf("expected 2 ranges, got %d", len(ranges))
	}
	if ranges[0].Start != 1 || ranges[0].End != 2 {
		t.Errorf("expected range [1,2], got [%d,%d]", ranges[0].Start, ranges[0].End)
	}
	if ranges[1].Start != 10 || ranges[1].End != 11 {
		t.Errorf("expected range [10,11], got [%d,%d]", ranges[1].Start, ranges[1].End)
	}
}

func TestParseGitDiffOutput_EmptyDiff(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader(""))
	result, err := parseGitDiffOutput(scanner)
	if err != nil {
		t.Fatalf("parseGitDiffOutput failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d files", len(result))
	}
}

func TestParseDiffFileHeader(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"src/main.go", "src/main.go"},
		{"\"path with spaces/file.go\"", "path with spaces/file.go"},
		{"  src/main.go  ", "src/main.go"},
	}

	for _, tc := range cases {
		got := parseDiffFileHeader(tc.input)
		if got != tc.expected {
			t.Errorf("parseDiffFileHeader(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestParseDiffHunk(t *testing.T) {
	cases := []struct {
		matches  []string
		expected LineRange
		ok       bool
	}{
		{[]string{"@@ -10,0 +15,5 @@", "15", "5"}, LineRange{Start: 15, End: 19}, true},
		{[]string{"@@ -10,0 +15 @@", "15", ""}, LineRange{Start: 15, End: 15}, true},
		{[]string{"@@ -abc +def @@", "abc", ""}, LineRange{}, false},
		{[]string{"@@ -10,0 +15,0 @@", "15", "0"}, LineRange{}, false},
	}

	for _, tc := range cases {
		got, ok := parseDiffHunk(tc.matches)
		if ok != tc.ok {
			t.Errorf("parseDiffHunk(%v) ok = %v, want %v", tc.matches, ok, tc.ok)
			continue
		}
		if ok && got != tc.expected {
			t.Errorf("parseDiffHunk(%v) = %v, want %v", tc.matches, got, tc.expected)
		}
	}
}

func TestProcessDiffLine(t *testing.T) {
	result := make(map[string][]LineRange)
	currentFile := ""

	currentFile = processDiffLine("+++ b/main.go", currentFile, result)
	if currentFile != "main.go" {
		t.Errorf("expected currentFile 'main.go', got %q", currentFile)
	}
	if _, ok := result["main.go"]; !ok {
		t.Error("expected 'main.go' to be added to result")
	}

	currentFile = processDiffLine("@@ -1,0 +5,3 @@", currentFile, result)
	ranges := result["main.go"]
	if len(ranges) != 1 {
		t.Fatalf("expected 1 range, got %d", len(ranges))
	}
	if ranges[0].Start != 5 || ranges[0].End != 7 {
		t.Errorf("expected range [5,7], got [%d,%d]", ranges[0].Start, ranges[0].End)
	}

	currentFile = processDiffLine("+++ b/other.go", currentFile, result)
	if currentFile != "other.go" {
		t.Errorf("expected currentFile 'other.go', got %q", currentFile)
	}
}
