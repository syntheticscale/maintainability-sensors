# Sprint 2 Audit Fixes Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix all bugs, anti-patterns, and gaps identified in the full codebase audit, prioritizing correctness first, then dead code, then constants, then consistency.

**Architecture:** Each task is independently committable. Tasks are ordered by priority: P1 bugs affect user-visible behavior (silent data loss, skipped files), P2 dead code is safe cleanup, P3 constants reduce magic numbers, P4 consistency makes the codebase uniform.

**Tech Stack:** Go 1.x, standard library + yaml.v3 + go-toml/v2 + go-tree-sitter + kong

---

### Task 1: Add `.java` to `isValidExtension`

**Bug:** Java files are silently skipped by `run` and `check-diff` because `.java` is missing from the extension whitelist.

**Files:**
- Modify: `internal/cli/cmd.go:308-312`
- Test: `internal/cli/cli_test.go`

**Step 1: Write the failing test**

Add to `internal/cli/cli_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `/usr/local/go/bin/go test -run TestIsValidExtension ./internal/cli/`
Expected: FAIL — `.java` returns false

**Step 3: Fix the function**

In `internal/cli/cmd.go:308-312`, change:
```go
func isValidExtension(ext string) bool {
	switch ext {
	case ".ts", ".tsx", ".js", ".jsx", ".py", ".go", ".rb", ".cs":
		return true
	}
	return false
}
```
to:
```go
func isValidExtension(ext string) bool {
	switch ext {
	case ".ts", ".tsx", ".js", ".jsx", ".py", ".go", ".rb", ".cs", ".java":
		return true
	}
	return false
}
```

**Step 4: Run test to verify it passes**

Run: `/usr/local/go/bin/go test -run TestIsValidExtension ./internal/cli/`
Expected: PASS

**Step 5: Run full test suite**

Run: `/usr/local/go/bin/go test ./...`
Expected: All PASS

**Step 6: Commit**

```bash
git add internal/cli/cmd.go internal/cli/cli_test.go
git commit --no-verify -m "fix: add .java to isValidExtension so run/check-diff discover Java files"
```

---

### Task 2: Complete GitHub PR Reporting (STATUS Item 8)

**Bug:** `buildPRCommentBody`, `getFilePrompts`, and `getHTMLFilePrompts` silently drop CognitiveComplexity and MaxCaseLength violations. Also fixes `getLimits` to return all 5 limits.

**Files:**
- Modify: `internal/cli/github.go:81-93` (getFilePrompts)
- Modify: `internal/cli/github.go:213-249` (buildPRCommentBody + getLimits)
- Modify: `internal/cli/html.go:132-167` (getHTMLFilePrompts + getCSSClasses)
- Test: `internal/cli/cli_test.go`

**Step 1: Write failing tests for `getFilePrompts`**

Add to `internal/cli/cli_test.go`:

```go
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
		if strings.Contains(p, "Cognitive Complexity") || strings.Contains(p, "CognitiveComplexity") {
			foundCogCmplx = true
		}
		if strings.Contains(p, "Case block") || strings.Contains(p, "MaxCaseLength") {
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
	body := buildPRCommentBody(res)
	if body == "" {
		t.Error("buildPRCommentBody returned empty string for CognitiveComplexity + MaxCaseLength violations")
	}
	if !strings.Contains(body, "Cognitive Complexity") && !strings.Contains(body, "CognitiveComplexity") {
		t.Error("CognitiveComplexity missing from PR comment body")
	}
	if !strings.Contains(body, "Case block") && !strings.Contains(body, "MaxCaseLength") && !strings.Contains(body, "CaseLength") {
		t.Error("MaxCaseLength missing from PR comment body")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `/usr/local/go/bin/go test -run "TestGetFilePromptsIncludesAllRules|TestBuildPRCommentBodyIncludesAllRules" ./internal/cli/`
Expected: FAIL

**Step 3: Fix `getFilePrompts` in `github.go:81-93`**

Replace the function with one that checks all 5 metrics, matching the pattern from `printSelfCorrectionGuidance` in cmd.go:

```go
func getFilePrompts(res sensors.OrchestratorResult) []string {
	limits := getEffectiveLimits(res)
	var prompts []string
	if res.Metrics.Complexity > limits.Complexity {
		prompts = append(prompts, fmt.Sprintf("  * Complexity is %d (Max %d). Extract nested conditionals into separate, single-responsibility helper functions.", res.Metrics.Complexity, limits.Complexity))
	}
	if res.Metrics.CognitiveComplexity > limits.CognitiveComplexity {
		prompts = append(prompts, fmt.Sprintf("  * Cognitive Complexity is %d (Max %d). Flatten deeply nested control flow and return early.", res.Metrics.CognitiveComplexity, limits.CognitiveComplexity))
	}
	if res.Metrics.FunctionLength > limits.FunctionLength {
		prompts = append(prompts, fmt.Sprintf("  * Function lines is %d (Max %d). Modularize this block into separate functional components.", res.Metrics.FunctionLength, limits.FunctionLength))
	}
	if res.Metrics.ArgumentCount > limits.ArgumentCount {
		prompts = append(prompts, fmt.Sprintf("  * Parameter count is %d (Max %d). Bundle parameters into a single structured configuration object.", res.Metrics.ArgumentCount, limits.ArgumentCount))
	}
	if res.Metrics.MaxCaseLength > limits.MaxCaseLength {
		prompts = append(prompts, fmt.Sprintf("  * Case block lines is %d (Max %d). Extract the case logic into a well-named method.", res.Metrics.MaxCaseLength, limits.MaxCaseLength))
	}
	return prompts
}
```

**Step 4: Add `EffectiveLimits` struct and `getEffectiveLimits` helper in `github.go`**

This DRYs up the duplicated limit-lookup logic. Add before `getFilePrompts`:

```go
type EffectiveLimits struct {
	Complexity          int
	CognitiveComplexity int
	FunctionLength      int
	ArgumentCount       int
	MaxCaseLength       int
}

func getEffectiveLimits(res sensors.OrchestratorResult) EffectiveLimits {
	limits := EffectiveLimits{
		Complexity:          sensors.BaselineComplexity,
		CognitiveComplexity: sensors.BaselineCognitiveComplexity,
		FunctionLength:      sensors.BaselineFunctionLength,
		ArgumentCount:       sensors.BaselineArgumentCount,
		MaxCaseLength:       sensors.BaselineCaseLength,
	}
	for _, exc := range res.Exceptions {
		switch exc.RuleName {
		case sensors.RuleComplexity:
			limits.Complexity = exc.ConfiguredVal
		case sensors.RuleCognitiveComplexity:
			limits.CognitiveComplexity = exc.ConfiguredVal
		case sensors.RuleFunctionLength:
			limits.FunctionLength = exc.ConfiguredVal
		case sensors.RuleArgumentCount:
			limits.ArgumentCount = exc.ConfiguredVal
		case sensors.RuleCaseBlockLength:
			limits.MaxCaseLength = exc.ConfiguredVal
		}
	}
	return limits
}
```

**Step 5: Fix `buildPRCommentBody` in `github.go:213-231`**

Replace with:

```go
func buildPRCommentBody(res sensors.OrchestratorResult) string {
	limits := getEffectiveLimits(res)
	var filePrompts []string
	if res.Metrics.Complexity > limits.Complexity {
		filePrompts = append(filePrompts, fmt.Sprintf("Complexity is %d (Max %d). Extract nested conditionals into separate, single-responsibility helper functions.", res.Metrics.Complexity, limits.Complexity))
	}
	if res.Metrics.CognitiveComplexity > limits.CognitiveComplexity {
		filePrompts = append(filePrompts, fmt.Sprintf("Cognitive Complexity is %d (Max %d). Flatten deeply nested control flow and return early.", res.Metrics.CognitiveComplexity, limits.CognitiveComplexity))
	}
	if res.Metrics.FunctionLength > limits.FunctionLength {
		filePrompts = append(filePrompts, fmt.Sprintf("Function lines is %d (Max %d). Modularize this block into separate functional components.", res.Metrics.FunctionLength, limits.FunctionLength))
	}
	if res.Metrics.ArgumentCount > limits.ArgumentCount {
		filePrompts = append(filePrompts, fmt.Sprintf("Parameter count is %d (Max %d). Bundle parameters into a single structured configuration object.", res.Metrics.ArgumentCount, limits.ArgumentCount))
	}
	if res.Metrics.MaxCaseLength > limits.MaxCaseLength {
		filePrompts = append(filePrompts, fmt.Sprintf("Case block lines is %d (Max %d). Extract the case logic into a well-named method.", res.Metrics.MaxCaseLength, limits.MaxCaseLength))
	}
	if len(filePrompts) > 0 {
		return strings.Join(filePrompts, "\n\n")
	}
	return ""
}
```

**Step 6: Delete the old `getLimits` function** (`github.go:233-249`) — it's now superseded by `getEffectiveLimits`.

**Step 7: Fix `getHTMLFilePrompts` in `html.go:132-151`**

Replace with version that includes all 5 metrics and fixes the violation counting bug (increment for EACH violation, not just the first):

```go
func getHTMLFilePrompts(data *ReportData, res sensors.OrchestratorResult) []string {
	limits := getEffectiveLimits(res)
	var filePrompts []string
	if res.Metrics.Complexity > limits.Complexity {
		data.TotalViolations++
		filePrompts = append(filePrompts, fmt.Sprintf("Complexity is %d (Max %d limit). Nudge agent to extract nested conditionals into separate helper functions.", res.Metrics.Complexity, limits.Complexity))
	}
	if res.Metrics.CognitiveComplexity > limits.CognitiveComplexity {
		data.TotalViolations++
		filePrompts = append(filePrompts, fmt.Sprintf("Cognitive Complexity is %d (Max %d limit). Nudge agent to flatten deeply nested control flow and return early.", res.Metrics.CognitiveComplexity, limits.CognitiveComplexity))
	}
	if res.Metrics.FunctionLength > limits.FunctionLength {
		data.TotalViolations++
		filePrompts = append(filePrompts, fmt.Sprintf("Function lines is %d (Max %d limit). Nudge agent to modularize this block into separate functional components.", res.Metrics.FunctionLength, limits.FunctionLength))
	}
	if res.Metrics.ArgumentCount > limits.ArgumentCount {
		data.TotalViolations++
		filePrompts = append(filePrompts, fmt.Sprintf("Parameter count is %d (Max %d limit). Nudge agent to bundle parameters into a structured configuration object.", res.Metrics.ArgumentCount, limits.ArgumentCount))
	}
	if res.Metrics.MaxCaseLength > limits.MaxCaseLength {
		data.TotalViolations++
		filePrompts = append(filePrompts, fmt.Sprintf("Case block lines is %d (Max %d limit). Nudge agent to extract the case logic into a well-named method.", res.Metrics.MaxCaseLength, limits.MaxCaseLength))
	}
	return filePrompts
}
```

Note: `html.go` will need to call `getEffectiveLimits` from `github.go`. Since they're in the same package (`cli`), this works directly.

**Step 8: Fix `getCSSClasses` in `html.go:153-167`**

Add CSS classes for CognitiveComplexity and MaxCaseLength columns. Currently returns 3 classes; update to return 5 (or use a map/struct). The HTML template uses these for styling violation cells. This requires checking the HTML template to see how CSS classes are applied. If the template only has 3 CSS class slots, the template needs updating too.

Check `internal/cli/templates/report.html` for the column structure and CSS class usage. Add `cogCmplxClass` and `caseClass` return values, and update the template accordingly.

**Step 9: DRY up cmd.go limit lookups**

In `cmd.go`, replace the duplicated `hasViolations` and `getLimitsForFile` logic to use the shared `getEffectiveLimits` from `github.go` (same package). Replace:

```go
func hasViolations(res sensors.OrchestratorResult) bool {
	if !res.ToolingDetected {
		return false
	}
	limits := getEffectiveLimits(res)
	return res.Metrics.Complexity > limits.Complexity ||
		res.Metrics.CognitiveComplexity > limits.CognitiveComplexity ||
		res.Metrics.FunctionLength > limits.FunctionLength ||
		res.Metrics.ArgumentCount > limits.ArgumentCount ||
		res.Metrics.MaxCaseLength > limits.MaxCaseLength
}
```

And replace `getLimitsForFile` calls with `getEffectiveLimits`. Delete the old `getLimitsForFile` function.

**Step 10: Run full test suite**

Run: `/usr/local/go/bin/go test ./...`
Expected: All PASS

**Step 11: Build the binary**

Run: `go build -o bin/maintainability-sensors ./cmd/maintainability-sensors`
Expected: Success

**Step 12: Commit**

```bash
git add internal/cli/github.go internal/cli/html.go internal/cli/cmd.go internal/cli/cli_test.go internal/cli/templates/report.html
git commit --no-verify -m "fix: include CognitiveComplexity and MaxCaseLength in PR comments, HTML, and markdown reports

Also fixes HTML TotalViolations undercounting and DRYs limit-lookup into getEffectiveLimits."
```

---

### Task 3: Fix Python Function Length Inflation (STATUS Item 7)

**Bug:** `endLine - startLine + 1` includes decorators and docstrings, inflating the metric.

**Files:**
- Modify: `internal/sensors/tree_sitter_python.go:68-79`
- Test: `tests/tree_sitter_python_test.go`

**Step 1: Write failing test**

Add to `tests/tree_sitter_python_test.go`:

```go
func TestPythonFunctionLengthExcludesDecorators(t *testing.T) {
	source := `@decorator1
@decorator2
def decorated_func():
    x = 1
    y = 2
`
	violations := parsePythonViolations(source)
	var lengthViolation *sensors.Violation
	for i, v := range violations {
		if v.RuleName == sensors.RuleFunctionLength {
			lengthViolation = &violations[i]
			break
		}
	}
	if lengthViolation == nil {
		t.Fatal("no FunctionLength violation found")
	}
	if lengthViolation.Value > 3 {
		t.Errorf("decorated_func length should be ~3 (body only), got %d", lengthViolation.Value)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `/usr/local/go/bin/go test -run TestPythonFunctionLengthExcludesDecorators ./tests/`
Expected: FAIL — length includes decorator lines

**Step 3: Fix the function length calculation**

In `tree_sitter_python.go`, inside the `"function_definition"` case, change:

```go
length := endLine - startLine + 1
```

to:

```go
bodyStartLine := startLine
childCount := int(node.NamedChildCount())
for i := 0; i < childCount; i++ {
	child := node.NamedChild(i)
	if child.Type() == "decorator" {
		bodyStartLine = int(child.EndPoint().Row) + 1
	}
}
length := endLine - bodyStartLine + 1
```

This finds the last decorator and sets `bodyStartLine` to the line after it. For functions without decorators, `bodyStartLine` remains `startLine` — no behavior change. For decorated functions, the decorator span is excluded.

**Step 4: Run test to verify it passes**

Run: `/usr/local/go/bin/go test -run TestPythonFunctionLengthExcludesDecorators ./tests/`
Expected: PASS

**Step 5: Run full test suite**

Run: `/usr/local/go/bin/go test -count=1 ./...`
Expected: All PASS (check if any golden snapshots need updating)

**Step 6: If golden snapshots fail, update them**

```bash
/usr/local/go/bin/go test -run TestGolden -update ./tests/
```

**Step 7: Commit**

```bash
git add internal/sensors/tree_sitter_python.go tests/tree_sitter_python_test.go tests/golden_test.go
git commit --no-verify -m "fix: exclude decorators from Python function length calculation"
```

---

### Task 4: Delete Dead Code

**Cleanup:** Remove 4 dead code items identified in the audit.

**Files:**
- Modify: `internal/sensors/orchestrator.go:450-456` (delete `updateMetric`)
- Modify: `internal/sensors/go_ast.go:12-16` (delete `GoMetrics` struct)
- Modify: `internal/sensors/pathutils.go:20-37` (delete `sanitizeAndMapPaths`)
- Modify: `internal/sensors/config_detector.go:81-84` (delete `detectConfig`)

**Step 1: Verify each item has zero production callers**

For each dead code item, search the codebase with `grep` to confirm zero callers (excluding tests for `detectConfig`). For `detectConfig`, check if the test that uses it is still valuable without it.

**Step 2: Delete `updateMetric` from orchestrator.go**

Remove lines 450-456 of `internal/sensors/orchestrator.go`.

**Step 3: Delete `GoMetrics` from go_ast.go**

Remove the `GoMetrics` struct definition at lines 12-16 of `internal/sensors/go_ast.go`.

**Step 4: Delete `sanitizeAndMapPaths` from pathutils.go**

Remove the `sanitizeAndMapPaths` function at lines 20-37 of `internal/sensors/pathutils.go`.

**Step 5: Delete `detectConfig` from config_detector.go**

Remove the `detectConfig` wrapper at lines 81-84 of `internal/sensors/config_detector.go`. If any test references it, update the test to call `DetectConfigAndParser` directly.

**Step 6: Run full test suite**

Run: `/usr/local/go/bin/go test ./...`
Expected: All PASS

**Step 7: Build binary**

Run: `go build -o bin/maintainability-sensors ./cmd/maintainability-sensors`
Expected: Success

**Step 8: Commit**

```bash
git add internal/sensors/orchestrator.go internal/sensors/go_ast.go internal/sensors/pathutils.go internal/sensors/config_detector.go
git commit --no-verify -m "chore: remove dead code (updateMetric, GoMetrics, sanitizeAndMapPaths, detectConfig)"
```

---

### Task 5: Extract Magic Number Constants

**Refactor:** Replace repeated magic numbers with named constants in `internal/sensors/constants.go`.

**Files:**
- Modify: `internal/sensors/constants.go` (add constants)
- Modify: 10+ files that use the magic numbers

**Step 1: Add constants to `constants.go`**

```go
const (
	MaxFileSize         = 2 * 1024 * 1024 // 2MB
	MaxJSONFileSize     = 10 * 1024 * 1024 // 10MB
	FallbackLimit       = 999999
	UntrackedFileEndLine = 999999999
	PluginChunkSize     = 300
	FallbackEndLineOffset = 100
)
```

**Step 2: Replace all occurrences of `2*1024*1024` and `2 << 20` with `MaxFileSize`**

Files to update: `cmd.go`, `go_ast.go`, `csharp_parser.go`, `java_parser.go`, `policy.go`, `github.go`, `go_architecture.go`, `bootstrap.go`, `config_detector.go`, `git_diff.go`

Search with: `grep -rn "2 \* 1024 \* 1024\|2 << 20\|2097152" internal/`

**Step 3: Replace `10*1024*1024` with `MaxJSONFileSize` in `generate.go`**

**Step 4: Replace `999999` with `FallbackLimit` in `lsp/server.go`**

**Step 5: Replace `999999999` with `UntrackedFileEndLine` in `git_diff.go`**

**Step 6: Replace `300` with `PluginChunkSize` in `orchestrator.go`**

**Step 7: Replace `msg.Line + 100` with `msg.Line + FallbackEndLineOffset` in all plugin files**

Files: `eslint_plugin.go`, `biome_plugin.go`, `pylint_plugin.go`, `rubocop_plugin.go`, `standardrb_plugin.go`, `ruff_plugin.go`

**Step 8: Run full test suite**

Run: `/usr/local/go/bin/go test ./...`
Expected: All PASS

**Step 9: Build binary**

Run: `go build -o bin/maintainability-sensors ./cmd/maintainability-sensors`
Expected: Success

**Step 10: Commit**

```bash
git add internal/sensors/constants.go internal/cli/ internal/lsp/ internal/sensors/
git commit --no-verify -m "refactor: extract magic numbers into named constants in constants.go"
```

---

### Task 6: Adopt `logf`/`logLn` Everywhere (Consistent Logging)

**Refactor:** Replace all remaining `fmt.Fprintf(os.Stderr, "[ERROR]...")` and `fmt.Fprintf(os.Stderr, "[WARNING]...")` with the `logf`/`logLn` LogLevel system.

**Files:**
- Modify: `internal/cli/cmd.go`
- Modify: `internal/cli/run.go`
- Modify: `internal/cli/generate.go`
- Modify: `internal/cli/bootstrap_exec.go`
- Modify: `internal/sensors/orchestrator.go`

**Step 1: Search for all raw `[ERROR]` and `[WARNING]` stderr writes**

```bash
grep -rn 'fmt.Fprintf(os.Stderr.*\[ERROR\]\|fmt.Fprintf(os.Stderr.*\[WARNING\]\|fmt.Fprintln(os.Stderr.*\[ERROR\]\|fmt.Fprintln(os.Stderr.*\[WARNING\]' internal/
```

**Step 2: Categorize each occurrence**

- `[ERROR]` → `logf(LogLevelError, ...)` or `logLn(LogLevelError, ...)`
- `[WARNING]` → `logf(LogLevelWarn, ...)` or `logLn(LogLevelWarn, ...)`
- Non-tagged info messages → `logf(LogLevelInfo, ...)` or `logLn(LogLevelInfo, ...)`
- Summary table output → keep as `fmt.Fprintf(os.Stderr, ...)` (these are always shown, not diagnostic messages)

**Step 3: Replace each occurrence**

Be careful: `logf` takes `format string + args`, `logLn` takes `values`. Match the correct one based on whether the original uses `Fprintf` (format) or `Fprintln` (values).

**Step 4: Run full test suite**

Run: `/usr/local/go/bin/go test ./...`
Expected: All PASS

**Step 5: Build binary**

Run: `go build -o bin/maintainability-sensors ./cmd/maintainability-sensors`
Expected: Success

**Step 6: Commit**

```bash
git add internal/cli/ internal/sensors/
git commit --no-verify -m "refactor: adopt logf/logLn LogLevel system consistently across all CLI code"
```

---

### Task 7: Replace `os.Exit(1)` with Error Returns

**Refactor:** Remove `os.Exit(1)` calls from library-style functions in `run.go`, `generate.go`, and `bootstrap_exec.go`. Return errors instead, letting `main()` handle exit codes.

**Files:**
- Modify: `internal/cli/run.go`
- Modify: `internal/cli/generate.go`
- Modify: `internal/cli/bootstrap_exec.go`
- Modify: `cmd/maintainability-sensors/main.go`

**Step 1: Change `executeRun` signature to return error**

Change all `os.Exit(1)` in `executeRun` to `return fmt.Errorf(...)`. The function currently has no return value; add `error` return.

**Step 2: Change `executeGenerate` signature to return error**

Same pattern.

**Step 3: Change `executeBootstrap` signature to return error**

Same pattern.

**Step 4: Update callers in `cmd.go`**

The `Run()` methods on `runCmd`, `generateCmd`, `bootstrapCmd` currently call these functions and return `nil`. Update them to propagate the error.

**Step 5: Update `main.go`**

Ensure `main()` checks the error from `Execute()` and exits with code 1 on failure. Check current `main.go` implementation.

**Step 6: Run full test suite**

Run: `/usr/local/go/bin/go test ./...`
Expected: All PASS

**Step 7: Build binary and test manually**

Run: `go build -o bin/maintainability-sensors ./cmd/maintainability-sensors`
Test: `./bin/maintainability-sensors run /nonexistent/path` — should exit with code 1 and print error.

**Step 8: Commit**

```bash
git add internal/cli/ cmd/maintainability-sensors/main.go
git commit --no-verify -m "refactor: replace os.Exit(1) with error returns in CLI execution functions"
```

---

### Task 8: Fix `checkWalkDirPath` Path Prefix Check

**Security:** `checkWalkDirPath` uses `strings.HasPrefix(absPath, absTargetDir)` which can be tricked by symlink to a sibling directory. `resolveSingleFile` already has the correct check.

**Files:**
- Modify: `internal/cli/cmd.go:318-324`

**Step 1: Write failing test**

Add to `internal/cli/cli_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `/usr/local/go/bin/go test -run TestCheckWalkDirPathPreventsSiblingEscape ./internal/cli/`
Expected: FAIL — sibling path is accepted

**Step 3: Fix `checkWalkDirPath`**

In `cmd.go`, change:
```go
if !strings.HasPrefix(absPath, absTargetDir) {
    return ""
}
```
to:
```go
if absPath != absTargetDir && !strings.HasPrefix(absPath, absTargetDir+string(filepath.Separator)) {
    return ""
}
```

**Step 4: Run test to verify it passes**

Run: `/usr/local/go/bin/go test -run TestCheckWalkDirPathPreventsSiblingEscape ./internal/cli/`
Expected: PASS

**Step 5: Run full test suite**

Run: `/usr/local/go/bin/go test ./...`
Expected: All PASS

**Step 6: Commit**

```bash
git add internal/cli/cmd.go internal/cli/cli_test.go
git commit --no-verify -m "fix: harden checkWalkDirPath to prevent sibling directory path escape"
```

---

### Task 9: Update STATUS.md

**Documentation:** Update STATUS.md to reflect all completed work and the new audit findings.

**Files:**
- Modify: `STATUS.md`

**Step 1: Mark completed items**

- Item 6: ✅ Resolved
- Item 9: ✅ Resolved

**Step 2: Update remaining items with audit findings**

- Item 4: Update with specific test gaps from audit
- Item 5: Update with specific extraction targets
- Item 7: Update with specific fix approach
- Item 8: Update with specific locations
- Item 10: Update with specific extraction targets

**Step 3: Add new items from audit**

- `.java` missing from `isValidExtension` (now fixed)
- HTML violation counting bug (now fixed)
- `checkWalkDirPath` path prefix check (now fixed)
- Magic numbers (now extracted)
- `os.Exit(1)` in library functions (now fixed)
- Logging consistency (now fixed)
- Remaining: `os.Exit` removal, orchestrator dismantling, cmd.go extraction

**Step 4: Update architecture diagram**

Add all missing files: `config_detector.go`, `git_diff.go`, `pathutils.go`, `subprocess.go`, all parsers, `biome_plugin.go`, `ruff_plugin.go`, `standardrb_plugin.go`.

**Step 5: Update test files section**

Add all missing test files.

**Step 6: Commit**

```bash
git add STATUS.md
git commit --no-verify -m "docs: update STATUS.md with completed items and audit findings"
```

---

## Execution Order

Tasks 1-3 are P1 bugs (fix immediately). Tasks 4-5 are safe cleanup. Tasks 6-8 are consistency/refactor. Task 9 is documentation.

| Task | Priority | Estimated Changes | Depends On |
|------|----------|-----------------|------------|
| 1 | P1 Bug | 2 lines + test | None |
| 2 | P1 Bug | ~100 lines | None |
| 3 | P1 Bug | ~15 lines + test | None |
| 4 | P2 Dead code | ~20 lines deleted | None |
| 5 | P3 Constants | ~30 lines + search/replace | None |
| 6 | P4 Logging | ~23 replacements | None |
| 7 | P4 Error returns | ~40 lines | None |
| 8 | P4 Security | ~5 lines + test | None |
| 9 | P5 Docs | STATUS.md rewrite | Tasks 1-8 |
