package cli

//nolint // maintainability: highly cohesive test

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/syntheticscale/maintainability-sensors/internal/sensors"
)

// cliBinary holds the path to the compiled CLI binary (set in TestMain).
var cliBinary string

func TestMain(m *testing.M) {
	// Build the CLI binary once for subprocess tests
	tmpDir, err := os.MkdirTemp("", "cli-test-*")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}
	cliBinary = filepath.Join(tmpDir, "maintainability-sensors")

	_, thisFile, _, _ := runtime.Caller(0)
	moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", cliBinary, "./cmd/maintainability-sensors") // -buildvcs=false avoids build failures in shallow worktrees or Docker without git history
	cmd.Dir = moduleRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		panic("failed to build CLI binary: " + err.Error() + "\n" + string(output))
	}

	code := m.Run()

	os.RemoveAll(tmpDir)
	os.Exit(code)
}

// ─── Test helpers ───

func orchestratedResult(path string, complexity, funcLength, argCount int, exceptions []sensors.RelaxedLimit) sensors.OrchestratorResult {
	return sensors.OrchestratorResult{
		FilePath:        path,
		Language:        "go",
		ToolingDetected: true,
		Metrics: sensors.MaintainabilityMetrics{
			Complexity:     complexity,
			FunctionLength: funcLength,
			ArgumentCount:  argCount,
		},
		Exceptions: exceptions,
	}
}

func blindResult(path string, lang string) sensors.OrchestratorResult {
	return sensors.OrchestratorResult{
		FilePath:        path,
		Language:        lang,
		ToolingDetected: false,
		Message:         "no config found",
	}
}

// ─── GenerateMarkdownScorecard ───

func TestGenerateMarkdownScorecard_EmptyResults(t *testing.T) {
	md := GenerateMarkdownScorecard([]sensors.OrchestratorResult{})

	if !strings.Contains(md, "Maintainability Sensors Scorecard") {
		t.Error("expected scorecard header")
	}
	if !strings.Contains(md, "Scan Summary") {
		t.Error("expected scan summary section")
	}
	if strings.Contains(md, "Actionable Refactoring Prompts") {
		t.Error("should not have self-correction section for empty results")
	}
	if strings.Contains(md, "Configured Exceptions") {
		t.Error("should not have exceptions section for empty results")
	}
}

func TestGenerateMarkdownScorecard_AllOrchestrated(t *testing.T) {
	results := []sensors.OrchestratorResult{
		orchestratedResult("/repo/clean.go", 5, 30, 3, nil),
		orchestratedResult("/repo/another.go", 8, 50, 4, nil),
	}

	md := GenerateMarkdownScorecard(results)

	if !strings.Contains(md, "ORCHESTRATED") {
		t.Error("expected ORCHESTRATED status")
	}
	if strings.Contains(md, "BLIND") {
		t.Error("should not have BLIND status")
	}
	if strings.Contains(md, "Actionable Refactoring Prompts") {
		t.Error("should not have self-correction section when no violations")
	}
}

func TestGenerateMarkdownScorecard_AllBlind(t *testing.T) {
	results := []sensors.OrchestratorResult{
		blindResult("/repo/legacy.cs", "csharp"),
		blindResult("/repo/old.java", "java"),
	}

	md := GenerateMarkdownScorecard(results)

	if !strings.Contains(md, "BLIND") {
		t.Error("expected BLIND status")
	}
	if strings.Contains(md, "ORCHESTRATED") {
		t.Error("should not have ORCHESTRATED status")
	}
	if strings.Contains(md, "Actionable Refactoring Prompts") {
		t.Error("should not have self-correction section for blind files")
	}
}

func TestGenerateMarkdownScorecard_Mixed(t *testing.T) {
	results := []sensors.OrchestratorResult{
		orchestratedResult("/repo/clean.go", 5, 30, 3, nil),
		blindResult("/repo/legacy.cs", "csharp"),
		orchestratedResult("/repo/another.go", 8, 50, 4, nil),
	}

	md := GenerateMarkdownScorecard(results)

	if !strings.Contains(md, "ORCHESTRATED") {
		t.Error("expected ORCHESTRATED status")
	}
	if !strings.Contains(md, "BLIND") {
		t.Error("expected BLIND status")
	}
}

func TestGenerateMarkdownScorecard_WithViolations(t *testing.T) {
	results := []sensors.OrchestratorResult{
		orchestratedResult("/repo/complex.go", 15, 80, 7, nil),
	}

	md := GenerateMarkdownScorecard(results)

	if !strings.Contains(md, "Actionable Refactoring Prompts") {
		t.Error("expected self-correction prompts section")
	}
	if !strings.Contains(md, "complex.go") {
		t.Error("expected filename in prompts section")
	}
	if !strings.Contains(md, "Complexity is 15") {
		t.Error("expected complexity violation message")
	}
	if !strings.Contains(md, "Function lines is 80") {
		t.Error("expected function length violation message")
	}
	if !strings.Contains(md, "Parameter count is 7") {
		t.Error("expected parameter count violation message")
	}
}

func TestGenerateMarkdownScorecard_WithExceptions(t *testing.T) {
	exceptions := []sensors.RelaxedLimit{
		{RuleName: "Complexity", ConfiguredVal: 15, BaselineVal: 8},
	}
	results := []sensors.OrchestratorResult{
		orchestratedResult("/repo/relaxed.go", 5, 30, 3, exceptions),
	}

	md := GenerateMarkdownScorecard(results)

	if !strings.Contains(md, "Configured Exceptions") {
		t.Error("expected exceptions section")
	}
	if !strings.Contains(md, "relaxed.go") {
		t.Error("expected filename in exceptions section")
	}
	if !strings.Contains(md, "Complexity") {
		t.Error("expected rule name in exceptions")
	}
	if !strings.Contains(md, "Configured Limit is 15") {
		t.Error("expected configured value")
	}
	if !strings.Contains(md, "Standard Baseline is 8") {
		t.Error("expected baseline value")
	}
}

func TestGenerateMarkdownScorecard_ViolationsAndExceptions(t *testing.T) {
	exceptions := []sensors.RelaxedLimit{
		{RuleName: "FunctionLength", ConfiguredVal: 100, BaselineVal: 50},
	}
	results := []sensors.OrchestratorResult{
		orchestratedResult("/repo/mixed.go", 12, 60, 3, exceptions),
	}

	md := GenerateMarkdownScorecard(results)

	if !strings.Contains(md, "Actionable Refactoring Prompts") {
		t.Error("expected self-correction prompts section")
	}
	if !strings.Contains(md, "Configured Exceptions") {
		t.Error("expected exceptions section")
	}
}

func TestGenerateMarkdownScorecard_SingleViolation(t *testing.T) {
	results := []sensors.OrchestratorResult{
		orchestratedResult("/repo/only_complex.go", 20, 10, 2, nil),
	}

	md := GenerateMarkdownScorecard(results)

	if !strings.Contains(md, "Actionable Refactoring Prompts") {
		t.Error("expected self-correction prompts section")
	}
	if !strings.Contains(md, "Complexity is 20") {
		t.Error("expected complexity violation")
	}
	if strings.Contains(md, "Function lines is") {
		t.Error("should not have function length violation")
	}
	if strings.Contains(md, "Parameter count is") {
		t.Error("should not have parameter count violation")
	}
}

// ─── GenerateHTMLScorecard ───

func TestGenerateHTMLScorecard_EmptyResults(t *testing.T) {
	html := GenerateHTMLScorecard([]sensors.OrchestratorResult{})

	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("expected valid HTML doctype")
	}
	if !strings.Contains(html, "Maintainability Sensors Scorecard") {
		t.Error("expected scorecard title")
	}
	if !strings.Contains(html, "Zero maintainability violations detected") {
		t.Error("expected empty violations state")
	}
	if !strings.Contains(html, "No relaxed limits detected") {
		t.Error("expected empty exceptions state")
	}
}

func TestGenerateHTMLScorecard_AllOrchestrated(t *testing.T) {
	results := []sensors.OrchestratorResult{
		orchestratedResult("/repo/clean.go", 5, 30, 3, nil),
	}

	html := GenerateHTMLScorecard(results)

	if !strings.Contains(html, "clean.go") {
		t.Error("expected filename in HTML")
	}
	if !strings.Contains(html, "ORCHESTRATED") {
		t.Error("expected ORCHESTRATED badge")
	}
	if strings.Contains(html, "BLIND") {
		t.Error("should not have BLIND badge")
	}
}

func TestGenerateHTMLScorecard_AllBlind(t *testing.T) {
	results := []sensors.OrchestratorResult{
		blindResult("/repo/legacy.cs", "csharp"),
	}

	html := GenerateHTMLScorecard(results)

	if !strings.Contains(html, "legacy.cs") {
		t.Error("expected filename in HTML")
	}
	if !strings.Contains(html, "BLIND") {
		t.Error("expected BLIND badge")
	}
	if strings.Contains(html, "ORCHESTRATED") {
		t.Error("should not have ORCHESTRATED badge")
	}
}

func TestGenerateHTMLScorecard_WithViolations(t *testing.T) {
	results := []sensors.OrchestratorResult{
		orchestratedResult("/repo/bad.go", 15, 80, 7, nil),
	}

	html := GenerateHTMLScorecard(results)

	if !strings.Contains(html, "bad.go") {
		t.Error("expected filename")
	}
	if !strings.Contains(html, "text-error") {
		t.Error("expected error CSS class for violations")
	}
	if !strings.Contains(html, "Actionable Refactoring Prompts") {
		t.Error("expected self-correction prompts section")
	}
	if !strings.Contains(html, "Complexity is 15") {
		t.Error("expected complexity violation in prompts")
	}
}

func TestGenerateHTMLScorecard_WithExceptions(t *testing.T) {
	exceptions := []sensors.RelaxedLimit{
		{RuleName: "Complexity", ConfiguredVal: 15, BaselineVal: 8},
	}
	results := []sensors.OrchestratorResult{
		orchestratedResult("/repo/relaxed.go", 5, 30, 3, exceptions),
	}

	html := GenerateHTMLScorecard(results)

	if !strings.Contains(html, "Configured Exceptions") {
		t.Error("expected exceptions section")
	}
	if !strings.Contains(html, "Complexity") {
		t.Error("expected rule name in exceptions")
	}
	if !strings.Contains(html, "Configured Limit 15") {
		t.Error("expected configured limit value")
	}
}

func TestGenerateHTMLScorecard_Mixed(t *testing.T) {
	results := []sensors.OrchestratorResult{
		orchestratedResult("/repo/clean.go", 5, 30, 3, nil),
		blindResult("/repo/legacy.cs", "csharp"),
	}

	html := GenerateHTMLScorecard(results)

	if !strings.Contains(html, "clean.go") {
		t.Error("expected orchestrated filename")
	}
	if !strings.Contains(html, "legacy.cs") {
		t.Error("expected blind filename")
	}
	if !strings.Contains(html, "Zero maintainability violations detected") {
		t.Error("expected empty violations state when orchestrated files have no violations")
	}
}

// ─── WriteGitHubStepSummary ───

func TestWriteGitHubStepSummary_NoEnvVar(t *testing.T) {
	unsetEnv(t, "GITHUB_STEP_SUMMARY")

	err := WriteGitHubStepSummary("test content")
	if err != nil {
		t.Errorf("expected nil error when GITHUB_STEP_SUMMARY not set, got: %v", err)
	}
}

func TestWriteGitHubStepSummary_WritesToFile(t *testing.T) {
	tmpDir := t.TempDir()
	summaryFile := filepath.Join(tmpDir, "step-summary.md")

	setEnv(t, "GITHUB_STEP_SUMMARY", summaryFile)

	content := "# Test Scorecard\n\nSome content here."
	err := WriteGitHubStepSummary(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(summaryFile)
	if err != nil {
		t.Fatalf("failed to read summary file: %v", err)
	}
	if string(data) != content {
		t.Errorf("expected %q, got %q", content, string(data))
	}
}

func TestWriteGitHubStepSummary_AppendsToExisting(t *testing.T) {
	tmpDir := t.TempDir()
	summaryFile := filepath.Join(tmpDir, "step-summary.md")

	setEnv(t, "GITHUB_STEP_SUMMARY", summaryFile)

	if err := os.WriteFile(summaryFile, []byte("existing content\n"), 0644); err != nil {
		t.Fatalf("failed to write initial content: %v", err)
	}

	err := WriteGitHubStepSummary("new content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(summaryFile)
	if err != nil {
		t.Fatalf("failed to read summary file: %v", err)
	}
	expected := "existing content\nnew content"
	if string(data) != expected {
		t.Errorf("expected %q, got %q", expected, string(data))
	}
}

// ─── getPRNumber ───

func TestGetPRNumber_FromEventPath(t *testing.T) {
	tmpDir := t.TempDir()
	eventFile := filepath.Join(tmpDir, "event.json")

	eventJSON := `{"pull_request": {"number": 42}}`
	if err := os.WriteFile(eventFile, []byte(eventJSON), 0644); err != nil {
		t.Fatalf("failed to write event file: %v", err)
	}

	setEnv(t, "GITHUB_EVENT_PATH", eventFile)
	unsetEnv(t, "GITHUB_REF")

	prNum, err := getPRNumber()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prNum != "42" {
		t.Errorf("expected PR number 42, got %q", prNum)
	}
}

func TestGetPRNumber_FromGitHubRef(t *testing.T) {
	unsetEnv(t, "GITHUB_EVENT_PATH")
	setEnv(t, "GITHUB_REF", "refs/pull/123/merge")

	prNum, err := getPRNumber()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prNum != "123" {
		t.Errorf("expected PR number 123, got %q", prNum)
	}
}

func TestGetPRNumber_EventPathTakesPrecedence(t *testing.T) {
	tmpDir := t.TempDir()
	eventFile := filepath.Join(tmpDir, "event.json")

	eventJSON := `{"pull_request": {"number": 99}}`
	if err := os.WriteFile(eventFile, []byte(eventJSON), 0644); err != nil {
		t.Fatalf("failed to write event file: %v", err)
	}

	setEnv(t, "GITHUB_EVENT_PATH", eventFile)
	setEnv(t, "GITHUB_REF", "refs/pull/555/merge")

	prNum, err := getPRNumber()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prNum != "99" {
		t.Errorf("expected PR number 99 from event path, got %q", prNum)
	}
}

func TestGetPRNumber_MissingEnvReturnsError(t *testing.T) {
	unsetEnv(t, "GITHUB_EVENT_PATH")
	unsetEnv(t, "GITHUB_REF")

	_, err := getPRNumber()
	if err == nil {
		t.Fatal("expected error when no PR number source available")
	}
	if !strings.Contains(err.Error(), "could not determine PR number") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGetPRNumber_InvalidEventFile(t *testing.T) {
	setEnv(t, "GITHUB_EVENT_PATH", "/nonexistent/path/event.json")
	setEnv(t, "GITHUB_REF", "refs/pull/77/merge")

	prNum, err := getPRNumber()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prNum != "77" {
		t.Errorf("expected PR number 77 from GITHUB_REF fallback, got %q", prNum)
	}
}

func TestGetPRNumber_EventFileWithNoPR(t *testing.T) {
	tmpDir := t.TempDir()
	eventFile := filepath.Join(tmpDir, "event.json")

	eventJSON := `{"push": {"ref": "refs/heads/main"}}`
	if err := os.WriteFile(eventFile, []byte(eventJSON), 0644); err != nil {
		t.Fatalf("failed to write event file: %v", err)
	}

	setEnv(t, "GITHUB_EVENT_PATH", eventFile)
	unsetEnv(t, "GITHUB_REF")

	_, err := getPRNumber()
	if err == nil {
		t.Fatal("expected error when event file has no PR number")
	}
}

// ─── PostGitHubPRComment error paths ───

func TestPostGitHubPRComment_NoToken(t *testing.T) {
	unsetEnv(t, "GITHUB_TOKEN")
	unsetEnv(t, "GITHUB_REPOSITORY")
	unsetEnv(t, "GITHUB_REF")

	err := PostGitHubPRComment(nil)
	if err == nil {
		t.Fatal("expected error when GITHUB_TOKEN is not set")
	}
	if !strings.Contains(err.Error(), "GITHUB_TOKEN") {
		t.Errorf("expected error about GITHUB_TOKEN, got: %v", err)
	}
}

func TestPostGitHubPRComment_NoRepo(t *testing.T) {
	setEnv(t, "GITHUB_TOKEN", "fake-token")
	unsetEnv(t, "GITHUB_REPOSITORY")

	err := PostGitHubPRComment(nil)
	if err == nil {
		t.Fatal("expected error when GITHUB_REPOSITORY is not set")
	}
	if !strings.Contains(err.Error(), "GITHUB_REPOSITORY") {
		t.Errorf("expected error about GITHUB_REPOSITORY, got: %v", err)
	}
}

func TestPostGitHubPRComment_NoPRNumber(t *testing.T) {
	setEnv(t, "GITHUB_TOKEN", "fake-token")
	setEnv(t, "GITHUB_REPOSITORY", "owner/repo")
	unsetEnv(t, "GITHUB_EVENT_PATH")
	unsetEnv(t, "GITHUB_REF")

	err := PostGitHubPRComment(nil)
	if err == nil {
		t.Fatal("expected error when PR number cannot be determined")
	}
	if !strings.Contains(err.Error(), "PR number") {
		t.Errorf("expected error about PR number, got: %v", err)
	}
}

// ─── writeReports ───

func TestWriteReports_NoOutputPaths(t *testing.T) {
	results := []sensors.OrchestratorResult{
		orchestratedResult("/repo/test.go", 5, 30, 3, nil),
	}

	err := writeReports(results, ReportOptions{ActionVerb: "Tested"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteReports_WritesMarkdown(t *testing.T) {
	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "report.md")

	results := []sensors.OrchestratorResult{
		orchestratedResult("/repo/test.go", 5, 30, 3, nil),
	}

	err := writeReports(results, ReportOptions{MarkdownOut: mdPath, ActionVerb: "Tested"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("failed to read markdown file: %v", err)
	}
	if !strings.Contains(string(data), "Maintainability Sensors Scorecard") {
		t.Error("expected scorecard content in markdown file")
	}
}

func TestWriteReports_WritesJSON(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "report.json")

	results := []sensors.OrchestratorResult{
		orchestratedResult("/repo/test.go", 5, 30, 3, nil),
	}

	err := writeReports(results, ReportOptions{JSONOut: jsonPath, ActionVerb: "Tested"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read JSON file: %v", err)
	}

	var parsed []sensors.OrchestratorResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if len(parsed) != 1 {
		t.Errorf("expected 1 result, got %d", len(parsed))
	}
}

func TestWriteReports_WritesHTML(t *testing.T) {
	tmpDir := t.TempDir()
	htmlPath := filepath.Join(tmpDir, "report.html")

	results := []sensors.OrchestratorResult{
		orchestratedResult("/repo/test.go", 5, 30, 3, nil),
	}

	err := writeReports(results, ReportOptions{HTMLOut: htmlPath, ActionVerb: "Tested"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("failed to read HTML file: %v", err)
	}
	if !strings.Contains(string(data), "<!DOCTYPE html>") {
		t.Error("expected valid HTML in output file")
	}
}

func TestWriteReports_WritesAllFormats(t *testing.T) {
	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "report.md")
	jsonPath := filepath.Join(tmpDir, "report.json")
	htmlPath := filepath.Join(tmpDir, "report.html")

	results := []sensors.OrchestratorResult{
		orchestratedResult("/repo/test.go", 12, 60, 6, nil),
	}

	err := writeReports(results, ReportOptions{MarkdownOut: mdPath, JSONOut: jsonPath, HTMLOut: htmlPath, ActionVerb: "Tested"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(mdPath); os.IsNotExist(err) {
		t.Error("markdown file was not created")
	}
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Error("JSON file was not created")
	}
	if _, err := os.Stat(htmlPath); os.IsNotExist(err) {
		t.Error("HTML file was not created")
	}
}

func TestWriteReports_InvalidPath(t *testing.T) {
	results := []sensors.OrchestratorResult{
		orchestratedResult("/repo/test.go", 5, 30, 3, nil),
	}

	err := writeReports(results, ReportOptions{MarkdownOut: "/nonexistent/dir/report.md", ActionVerb: "Tested"})
	if err == nil {
		t.Fatal("expected error when writing to nonexistent directory")
	}
}

// ─── executeGenerate ───

func TestExecuteGenerate_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "bad.json")

	if err := os.WriteFile(jsonPath, []byte(`{not valid json`), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cmd := exec.Command("go", "run", "../../cmd/maintainability-sensors", "generate", jsonPath)
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit code for invalid JSON")
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 1 {
			t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
		}
	}
}

func TestExecuteGenerate_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("go", "run", "../../cmd/maintainability-sensors", "generate", filepath.Join(tmpDir, "nonexistent.json"))
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit code for missing file")
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 1 {
			t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
		}
	}
}

func TestExecuteGenerate_ValidJSONNoOutput(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "input.json")

	results := []sensors.OrchestratorResult{
		orchestratedResult("/repo/test.go", 5, 30, 3, nil),
	}
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal test data: %v", err)
	}
	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	captured := captureOutput(func() {
		// discard error; we check output below
		_ = executeGenerate(jsonPath, "", "")
	})

	if strings.Contains(captured, "ERROR") {
		t.Errorf("unexpected error output: %s", captured)
	}
}

func TestExecuteGenerate_ValidJSONNoOutput_ReturnsNil(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "input.json")

	results := []sensors.OrchestratorResult{
		orchestratedResult("/repo/test.go", 5, 30, 3, nil),
	}
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal test data: %v", err)
	}
	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	if err := executeGenerate(jsonPath, "", ""); err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
}

func TestExecuteGenerate_MissingFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "missing_filepath.json")

	// JSON missing file_path
	resultsJSON := `[{"language": "go"}]`
	if err := os.WriteFile(jsonPath, []byte(resultsJSON), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cmd := exec.Command("go", "run", "../../cmd/maintainability-sensors", "generate", jsonPath)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit code for missing file_path")
	}
	if !strings.Contains(string(output), "Missing 'file_path' in result at index 0") {
		t.Errorf("expected error message about missing file_path, got: %s", string(output))
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 1 {
			t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
		}
	}
}

func TestExecuteGenerate_MissingLanguage(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "missing_language.json")

	// JSON missing language
	resultsJSON := `[{"file_path": "/path/to/file.go"}]`
	if err := os.WriteFile(jsonPath, []byte(resultsJSON), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cmd := exec.Command("go", "run", "../../cmd/maintainability-sensors", "generate", jsonPath)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit code for missing language")
	}
	if !strings.Contains(string(output), "Missing 'language' in result at index 0") {
		t.Errorf("expected error message about missing language, got: %s", string(output))
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 1 {
			t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
		}
	}
}

func TestExecuteGenerate_ValidJSONWithOutputs(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "input.json")
	mdPath := filepath.Join(tmpDir, "output.md")
	htmlPath := filepath.Join(tmpDir, "output.html")

	results := []sensors.OrchestratorResult{
		orchestratedResult("/repo/test.go", 5, 30, 3, nil),
	}
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal test data: %v", err)
	}
	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	captured := captureOutput(func() {
		_ = executeGenerate(jsonPath, mdPath, htmlPath)
	})

	if strings.Contains(captured, "ERROR") {
		t.Errorf("unexpected error output: %s", captured)
	}

	if _, err := os.Stat(mdPath); os.IsNotExist(err) {
		t.Error("markdown output file was not created")
	}
	if _, err := os.Stat(htmlPath); os.IsNotExist(err) {
		t.Error("HTML output file was not created")
	}
}

// ─── printScanResult ───

func TestPrintScanResult_JSONOutput(t *testing.T) {
	res := orchestratedResult("/repo/test.go", 5, 30, 3, nil)

	captured := captureOutput(func() {
		printScanResult(res, true)
	})

	var parsed sensors.OrchestratorResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(captured)), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed.FilePath != "/repo/test.go" {
		t.Errorf("expected file path /repo/test.go, got %q", parsed.FilePath)
	}
}

func TestPrintScanResult_OrchestratedText(t *testing.T) {
	res := orchestratedResult("/repo/test.go", 5, 30, 3, nil)

	captured := captureOutput(func() {
		printScanResult(res, false)
	})

	if !strings.Contains(captured, "ORCHESTRATED") {
		t.Error("expected ORCHESTRATED status")
	}
	if !strings.Contains(captured, "test.go") {
		t.Error("expected filename in output")
	}
	if !strings.Contains(captured, "Complexity") {
		t.Error("expected complexity in telemetry")
	}
}

func TestPrintScanResult_BlindText(t *testing.T) {
	res := blindResult("/repo/legacy.cs", "csharp")

	captured := captureOutput(func() {
		printScanResult(res, false)
	})

	if !strings.Contains(captured, "BLIND") {
		t.Error("expected BLIND status")
	}
	if !strings.Contains(captured, "legacy.cs") {
		t.Error("expected filename in output")
	}
}

func TestPrintScanResult_WithExceptions(t *testing.T) {
	exceptions := []sensors.RelaxedLimit{
		{RuleName: "Complexity", ConfiguredVal: 15, BaselineVal: 8},
	}
	res := orchestratedResult("/repo/relaxed.go", 5, 30, 3, exceptions)

	captured := captureOutput(func() {
		printScanResult(res, false)
	})

	if !strings.Contains(captured, "Configured Exceptions") {
		t.Error("expected exceptions section")
	}
	if !strings.Contains(captured, "Complexity") {
		t.Error("expected rule name in exceptions")
	}
}

func TestPrintScanResult_WithViolations(t *testing.T) {
	res := orchestratedResult("/repo/bad.go", 15, 80, 7, nil)

	captured := captureOutput(func() {
		printScanResult(res, false)
	})

	if !strings.Contains(captured, "Actionable Refactoring Prompts") {
		t.Error("expected self-correction prompts section")
	}
	if !strings.Contains(captured, "Complexity is 15") {
		t.Error("expected complexity violation")
	}
	if !strings.Contains(captured, "Function lines is 80") {
		t.Error("expected function length violation")
	}
	if !strings.Contains(captured, "Parameter count is 7") {
		t.Error("expected parameter count violation")
	}
}

// ─── printSelfCorrectionGuidance ───

func TestPrintSelfCorrectionGuidance_NoViolations(t *testing.T) {
	res := orchestratedResult("/repo/clean.go", 5, 30, 3, nil)

	captured := captureOutput(func() {
		printSelfCorrectionGuidance(sensors.Evaluate(res))
	})

	if captured != "" {
		t.Errorf("expected no output for clean file, got: %s", captured)
	}
}

func TestPrintSelfCorrectionGuidance_ComplexityViolation(t *testing.T) {
	res := orchestratedResult("/repo/complex.go", 20, 10, 2, nil)

	captured := captureOutput(func() {
		printSelfCorrectionGuidance(sensors.Evaluate(res))
	})

	if !strings.Contains(captured, "Complexity is 20") {
		t.Error("expected complexity violation guidance")
	}
	if strings.Contains(captured, "Function lines is") {
		t.Error("should not have function length guidance")
	}
}

func TestPrintSelfCorrectionGuidance_AllViolations(t *testing.T) {
	res := orchestratedResult("/repo/bad.go", 20, 100, 10, nil)

	captured := captureOutput(func() {
		printSelfCorrectionGuidance(sensors.Evaluate(res))
	})

	if !strings.Contains(captured, "Complexity is 20") {
		t.Error("expected complexity violation")
	}
	if !strings.Contains(captured, "Function lines is 100") {
		t.Error("expected function length violation")
	}
	if !strings.Contains(captured, "Parameter count is 10") {
		t.Error("expected parameter count violation")
	}
}

func TestGetFilePromptsIncludesAllRules(t *testing.T) {
	res := sensors.OrchestratorResult{
		ToolingDetected: true,
		Metrics: sensors.MaintainabilityMetrics{
			Complexity:          15,
			CognitiveComplexity: 15,
			FunctionLength:      60,
			ArgumentCount:       6,
			MaxCaseLength:       15,
		},
	}
	prompts := getFilePrompts(res)
	if len(prompts) != 5 {
		t.Errorf("expected 5 prompts, got %d: %v", len(prompts), prompts)
	}
	foundCogCmplx := false
	foundCase := false
	for _, p := range prompts {
		if strings.Contains(p, "Cognitive") {
			foundCogCmplx = true
		}
		if strings.Contains(p, "Case block") {
			foundCase = true
		}
	}
	if !foundCogCmplx {
		t.Error("CognitiveComplexity prompt missing from getFilePrompts")
	}
	if !foundCase {
		t.Error("MaxCaseLength prompt missing from getFilePrompts")
	}
}

func TestBuildPRCommentBodyIncludesAllRules(t *testing.T) {
	res := sensors.OrchestratorResult{
		ToolingDetected: true,
		Metrics: sensors.MaintainabilityMetrics{
			CognitiveComplexity: 15,
			MaxCaseLength:       15,
		},
	}
	summary := sensors.Evaluate(res)
	body := buildPRCommentBody(summary)
	if body == "" {
		t.Error("buildPRCommentBody returned empty string for CognitiveComplexity + MaxCaseLength violations")
	}
	if !strings.Contains(body, "Cognitive") {
		t.Error("CognitiveComplexity missing from PR comment body")
	}
	if !strings.Contains(body, "Case block") {
		t.Error("MaxCaseLength missing from PR comment body")
	}
}

func TestGetEffectiveLimitsWithExceptions(t *testing.T) {
	res := sensors.OrchestratorResult{
		Exceptions: []sensors.RelaxedLimit{
			{RuleName: sensors.RuleCognitiveComplexity, ConfiguredVal: 20, BaselineVal: sensors.BaselineCognitiveComplexity},
			{RuleName: sensors.RuleCaseBlockLength, ConfiguredVal: 30, BaselineVal: sensors.BaselineCaseLength},
		},
	}
	limits := sensors.GetEffectiveLimits(res)
	if limits.CognitiveComplexity != 20 {
		t.Errorf("expected CognitiveComplexity limit 20, got %d", limits.CognitiveComplexity)
	}
	if limits.MaxCaseLength != 30 {
		t.Errorf("expected MaxCaseLength limit 30, got %d", limits.MaxCaseLength)
	}
	if limits.Complexity != sensors.BaselineComplexity {
		t.Errorf("expected default Complexity limit %d, got %d", sensors.BaselineComplexity, limits.Complexity)
	}
}

func TestGetHTMLFilePromptsCountsAllViolations(t *testing.T) {
	res := sensors.OrchestratorResult{
		ToolingDetected: true,
		Metrics: sensors.MaintainabilityMetrics{
			Complexity:          15,
			CognitiveComplexity: 15,
			FunctionLength:      60,
			ArgumentCount:       6,
			MaxCaseLength:       15,
		},
	}
	data := &ReportData{}
	summary := sensors.Evaluate(res)
	prompts := getHTMLFilePrompts(data, summary)
	if len(prompts) != 5 {
		t.Errorf("expected 5 HTML prompts, got %d", len(prompts))
	}
	if data.TotalViolations != 5 {
		t.Errorf("expected TotalViolations=5, got %d", data.TotalViolations)
	}
}

func TestGetHTMLFilePromptsOnlyCogAndCase(t *testing.T) {
	res := sensors.OrchestratorResult{
		ToolingDetected: true,
		Metrics: sensors.MaintainabilityMetrics{
			CognitiveComplexity: 15,
			MaxCaseLength:       15,
		},
	}
	data := &ReportData{}
	summary := sensors.Evaluate(res)
	prompts := getHTMLFilePrompts(data, summary)
	if len(prompts) != 2 {
		t.Errorf("expected 2 HTML prompts for Cog+Case, got %d", len(prompts))
	}
	if data.TotalViolations != 2 {
		t.Errorf("expected TotalViolations=2, got %d", data.TotalViolations)
	}
}

func TestEvaluateDetectsCogAndCase(t *testing.T) {
	cogRes := sensors.OrchestratorResult{
		ToolingDetected: true,
		Metrics: sensors.MaintainabilityMetrics{
			CognitiveComplexity: 15,
		},
	}
	if !sensors.Evaluate(cogRes).HasViolations {
		t.Error("Evaluate should detect CognitiveComplexity violation")
	}

	caseRes := sensors.OrchestratorResult{
		ToolingDetected: true,
		Metrics: sensors.MaintainabilityMetrics{
			MaxCaseLength: 15,
		},
	}
	if !sensors.Evaluate(caseRes).HasViolations {
		t.Error("Evaluate should detect MaxCaseLength violation")
	}
}

// ─── isValidExtension ───

func TestIsValidExtension(t *testing.T) {
	tests := []struct {
		ext      string
		expected bool
	}{
		{".ts", true},
		{".tsx", true},
		{".js", true},
		{".jsx", true},
		{".py", true},
		{".go", true},
		{".rb", true},
		{".cs", true},
		{".java", true},
		{".txt", false},
		{".md", false},
	}
	for _, tt := range tests {
		if got := isValidExtension(tt.ext); got != tt.expected {
			t.Errorf("isValidExtension(%q) = %v, want %v", tt.ext, got, tt.expected)
		}
	}
}

// ─── checkWalkDirPath ───

func TestCheckWalkDirPathPreventsSiblingEscape(t *testing.T) {
	tmpDir := t.TempDir()
	siblingDir := tmpDir + "-sibling"
	os.MkdirAll(siblingDir, 0755)
	defer os.RemoveAll(siblingDir)
	absTargetDir, _ := filepath.Abs(tmpDir)
	result := checkWalkDirPath(siblingDir, absTargetDir)
	if result != "" {
		t.Errorf("checkWalkDirPath should reject sibling directory %q against target %q", siblingDir, absTargetDir)
	}
}

// ─── Helper: capture stdout/stderr ───

func captureOutput(fn func()) string {
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		panic("os.Pipe failed: " + err.Error())
	}

	os.Stdout = w
	os.Stderr = w
	defer func() {
		w.Close()
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	fn()

	// Close write end before reading
	w.Close()
	// Prevent defer from double-closing
	w = nil

	data, _ := io.ReadAll(r)
	return string(data)
}

// ─── Helper: set and restore env vars ───

func setEnv(t *testing.T, key, value string) {
	t.Helper()
	orig := os.Getenv(key)
	os.Setenv(key, value)
	t.Cleanup(func() {
		if orig == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, orig)
		}
	})
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	orig := os.Getenv(key)
	os.Unsetenv(key)
	t.Cleanup(func() {
		if orig != "" {
			os.Setenv(key, orig)
		}
	})
}
