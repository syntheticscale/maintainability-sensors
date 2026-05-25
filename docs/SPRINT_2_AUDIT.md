# Full Codebase Audit Report (Sprint 2 Baseline)

**Date:** 2026-05-25
**Branch:** main
**Note:** This audit was performed at the start of Sprint 2. Items marked ✅ have been resolved during Sprint 2.

---

## 1. Dead Code — ✅ All Resolved

| Item | Location | Status |
|------|----------|--------|
| `updateMetric` function | `internal/sensors/orchestrator.go:450-456` | ✅ Deleted |
| `GoMetrics` struct | `internal/sensors/go_ast.go:12-16` | ✅ Deleted |
| `sanitizeAndMapPaths` function | `internal/sensors/pathutils.go:20-37` | ✅ Deleted |
| `detectConfig` function | `internal/sensors/config_detector.go:81-84` | ✅ Deleted (tests updated) |
| `maxOf` function | `internal/sensors/config_parsers.go:192-202` | Used internally — retained |

---

## 2. Missing Test Coverage

| Package/File | Test File | Status |
|--------------|-----------|--------|
| `internal/cli/cmd.go` | `internal/cli/cli_test.go` | **Partial** — covers markdown/HTML generation, `printScanResult`, `writeReports`, GitHub PR helpers. Does NOT test: `FindFiles`, `hasViolations`, `FormatResultsCLI`, `isTrueViolation`, `hasOverlap`, `groupFilesByLanguage`, `processViolationsMap`, `processDeltaGroups`, `isSkippedDir`, `isValidExtension`, `checkWalkDirPath`, `resolveSingleFile`, `loadCheckDiffPolicy`. |
| `internal/cli/run.go` | (covered by `cli_test.go`) | **No direct tests** — `executeRun` calls `os.Exit(1)` making it hard to test. `ScanFiles` is untested. `postGitHubResults` is untested. `saveReportsAndExit` is untested. |
| `internal/cli/generate.go` | (covered by `cli_test.go`) | **No direct unit tests** — only tested via subprocess integration tests. `validateScorecardResults` is untested. `parseJSONScorecard` edge cases untested. |
| `internal/cli/bootstrap_exec.go` | None | **No test** — but trivially delegates to `sensors.BootstrapRepoWithPolicy`. Low risk. |
| `internal/cli/html.go` | (covered by `cli_test.go`) | **Partial** — tests generation but does NOT test `getHTMLFilePrompts` violation counting bug (see Section 5), `getCSSClasses`, `processBlindResult`, `processOrchestratedResult` individually. |
| `internal/cli/github.go` | (covered by `cli_test.go`) | **Partial** — tests `getPRNumber` and `PostGitHubReview` error paths, but NOT: `GenerateMarkdownScorecard` edge cases, `getFilePrompts`, `buildPRCommentBody` missing rules (Item 8), `sendGitHubReviewRequest`, `getRelativePath`. |
| `internal/cli/policy.go` | `internal/cli/policy_test.go` | **Good** — comprehensive coverage of `LoadPolicy`, `findConfigFile`, validation. Missing: `getBaselineForRule` (only used via `getThresholdForRule`), `getThresholdForRule` direct tests. |
| `internal/lsp/server.go` | `internal/lsp/server_test.go`, `server_bench_test.go` | **Partial** — tests `initialize` and `textDocument/didChange`. Does NOT test: `exit` handling, malformed JSON, `getLimitForRule` unit, concurrent `didChange` (the race that was fixed), `sendResponse` error paths. |
| `internal/sensors/orchestrator.go` | `tests/orchestrator_test.go`, `internal/sensors/orchestrated_scan_test.go` | **Partial** — tests scan results but NOT: `filterPathsForPlugin` edge cases, `analyzeInChunks` chunk boundaries (300 limit), `updateMetricsMap`/`updateDeltaMetricsMap` directly, `buildSingleResult` message logic, `findConfigAndParsers` with multiple configs. |
| `internal/sensors/go_ast.go` | `tests/orchestrator_test.go` | **Partial** — tests via `ParseGoAST` but NOT: `//nolint` suppression, cognitive complexity calculations directly, case block length detection, nested `FuncLit` exclusion. |
| `internal/sensors/tree_sitter_python.go` | `tests/tree_sitter_python_test.go`, `internal/sensors/orchestrated_scan_test.go` | **Good** — tests complexity, function length, argument count. Missing: `assert` statement complexity, `match` statement complexity, decorator inflation (Item 7). |
| `internal/sensors/tree_sitter_typescript.go` | `tests/tree_sitter_typescript_test.go` | **Good** — uses `testify`. Missing: `require()` import detection for architecture, arrow function single-param edge cases. |
| `internal/sensors/config_parsers.go` | `internal/sensors/parsers_test.go` | **Good** — comprehensive tests for `findAllConfigVals` across JSON, YAML, JS, TOML. Missing: edge cases in `extractVal` (e.g., nested maps with both `max` and `Max`). |
| `internal/sensors/config_detector.go` | `internal/sensors/config_detector_test.go` | **Good** — tests `DetectLanguage`, `DetectConfigAndParser`. Missing: `isValidConfigFile` edge cases. |
| `internal/sensors/git_diff.go` | `internal/sensors/git_diff_test.go` | **Good** — tests parsing. Missing: `GetModifiedLines` integration (requires git repo), `addUntrackedFiles` integration, `processUntrackedFile` edge cases. |
| `internal/sensors/pathutils.go` | `internal/sensors/sanitize_test.go` | **Good** — tests `sanitizePath` and `OrchestratedScan` integration. |
| `internal/sensors/subprocess.go` | `internal/sensors/subprocess_test.go` | **Good** — uses mock linter scripts. Tests ESLint, PyLint, RuboCop. Missing: Ruff, StandardRB subprocess tests. |
| `internal/sensors/architecture_parser.go` | `tests/architecture_test.go` | **Good** — tests Go, TS, Python architecture. |
| `internal/sensors/go_architecture.go` | None directly | No dedicated test — tested via `tests/architecture_test.go`. |
| `internal/sensors/csharp_parser.go` | `tests/orchestrator_test.go` | **Partial** — only tested via `ParseCSharp` in orchestrator_test. |
| `internal/sensors/java_parser.go` | `tests/orchestrator_test.go` | **Partial** — only tested via `ParseJava` in orchestrator_test. |
| `internal/sensors/eslint_plugin.go` | `internal/sensors/subprocess_test.go` | **Good** — tested via mock. |
| `internal/sensors/biome_plugin.go` | None | **No test** — `extractBiomeRuleAndVal` uses fragile `strings.Contains` heuristics. |
| `internal/sensors/ruff_plugin.go` | None | **No subprocess test** — only tested indirectly if ruff is installed. |
| `internal/sensors/standardrb_plugin.go` | None | **No subprocess test** — similar to ruff. |
| `internal/sensors/rubocop_plugin.go` | `internal/sensors/subprocess_test.go` | **Good** — tested via mock. |
| `internal/sensors/pylint_plugin.go` | `internal/sensors/subprocess_test.go` | **Good** — tested via mock. |
| **Files with NO tests at all:** | | `eslint_parser.go`, `biome_parser.go`, `ruff_parser.go`, `rubocop_parser.go`, `standardrb_parser.go`, `golangci_parser.go` — parser structs are trivial (just `Name()`, `Anchors()`, `Rules()`) and partially covered by `parsers_test.go`. |

---

## 3. Inconsistencies — ✅ All Critical Items Resolved

| Issue | Status | Details |
|-------|--------|---------|
| **`isValidExtension` missing `.java`** | ✅ Fixed | Added `.java` to the switch case |
| **Cognitive Complexity missing from HTML report** | ✅ Fixed | `getHTMLFilePrompts` + `getCSSClasses` now include all 5 metrics |
| **Cognitive Complexity missing from Markdown report** | ✅ Fixed | `getFilePrompts` now includes all 5 metrics |
| **Violation counting bug in HTML report** | ✅ Fixed | `TotalViolations++` now runs for each violation independently |
| **Inconsistent limit-lookup patterns** | ✅ Fixed | DRYed into `getEffectiveLimits` + `EffectiveLimits` struct; deleted `getLimits` and `getLimitsForFile` |
| **Mixed logging approaches** | ✅ Fixed | All `internal/cli/` stderr writes now use `logf`/`logLn` LogLevel system |
| **`detectLanguages` uses `strings.Contains` for skip dirs** | Open | `bootstrap.go:358` still uses fragile pattern |
| **Test helper duplication** | Open | `getMax` defined identically in 3 test files |
| **Mixed test assertion styles** | Open | `tree_sitter_typescript_test.go` uses testify; others use stdlib |
| **`getParsersForLang` missing java and csharp** | Intentional | Native-only, no linter configs for these languages |

---

## 4. STATUS.md Item Status

### Item 4: Test coverage — Partial
**Current state:** Tests have been added for `config_detector.go`, `git_diff.go`, and `orchestrated_scan.go`. The `tests/` directory has golden, architecture, component, and severity tests.  
**Remaining gaps:**
- No test for `biome_plugin.go` `extractBiomeRuleAndVal` (fragile `strings.Contains` heuristics)
- No subprocess tests for `ruff_plugin.go` or `standardrb_plugin.go`
- `internal/cli/cmd.go` has many untested functions: `FindFiles`, `hasViolations`, `FormatResultsCLI`, `processViolationsMap`, `processDeltaGroups`
- `internal/cli/run.go` has no direct tests (`executeRun` uses `os.Exit`)
- Coverage is well below the 70% target for `internal/sensors`

### Item 5: cmd.go refactoring — Partial
**Current state:** `executeRun` → `run.go`, `executeGenerate` → `generate.go`, `executeBootstrap` → `bootstrap_exec.go`. `policy.go` extracted.  
**Remaining gaps:**
- `cmd.go` is still 684 lines. It contains: CLI struct definitions, `CheckDiffCmd` and all its logic (`isTrueViolation`, `hasOverlap`, `mapModifiedLinesToAbsPaths`, `groupFilesByLanguage`, `ViolationCtx`, `processSingleViolationFile`, `processViolationsMap`, `formatViolationMessage`, `processDeltaGroups`), `FindFiles` and all its helpers, `FormatResultsCLI` and all print functions, `writeReports`, `hasViolations`, `getLimitsForFile`, `printSelfCorrectionGuidance`, `getSuppressionExample`.
- Further sub-package extraction (`reports/`, `github/`, `scan/`, `delta/`) not done.

### Item 7: Python function length calculation — ✅ Resolved
Tree-sitter already excludes decorators. The actual bug was docstring inflation — now subtracts docstring line count from function length.

### Item 8: GitHub PR reporting — ✅ Resolved
All 3 functions now include CognitiveComplexity and MaxCaseLength. HTML TotalViolations bug fixed. Limit-lookup DRYed into `getEffectiveLimits`.

### Item 10: orchestrator.go dismantling — Not addressed
**Current state:** `internal/sensors/orchestrator.go` is still 456 lines containing: `OrchestratorResult`, `RelaxedLimit`, `MaintainabilityMetrics`, `OrchestratedScan`, `processPluginsMetrics`, `OrchestratedScanBatch`, `ProcessDeltaCtx`, `processPluginsDelta`, `ScanDeltaBatch`, `filterPathsForPlugin`, `analyzeInChunks`, `UpdateMetricsCtx`, `isPathForPlugin`, `updateSingleMetric`, `updateMetricsForPath`, `findAndUpdateMetrics`, `updateMetricsMap`, `UpdateDeltaCtx`, `findAndUpdateDeltaMetrics`, `updateDeltaMetricsMap`, `BatchContext`, `hasNativePlugin`, `populateResultMessage`, `buildSingleResult`, `buildOrchestratorResults`, `findConfigAndParsers`, `updateMetric` (dead code).  
**Status:** None of the planned extractions (`result.go`, `delta.go`, `metric_updater.go`) have been done. Dead code `updateMetric` is still present.

---

## 5. Remaining Anti-Patterns — ✅ Magic Numbers Resolved

| Pattern | Status | Details |
|---------|--------|---------|
| **Magic number `2*1024*1024`** | ✅ Fixed | Replaced with `MaxFileSize` constant |
| **Magic number `10*1024*1024`** | ✅ Fixed | Replaced with `MaxJSONFileSize` constant |
| **Magic number `999999`** | ✅ Fixed | Replaced with `FallbackLimit` constant |
| **Magic number `999999999`** | ✅ Fixed | Replaced with `UntrackedFileEndLine` constant |
| **Magic number `300`** | ✅ Fixed | Replaced with `PluginChunkSize` constant |
| **Magic number `100`** | ✅ Fixed | Replaced with `FallbackEndLineOffset` constant |
| **Magic string `"-ast"` suffix** | Open | `strings.HasSuffix(pluginName, "-ast")` — should be a constant or interface method |
| **Fragile `strings.Contains` in biome_plugin.go** | Open | Biome diagnostic matching is fragile |
| **Fragile regex for JS config parsing** | Open | `findAllConfigValsJS` uses massive regex |
| **Inconsistent `[ERROR]`/`[WARNING]` prefix** | ✅ Fixed | All `internal/cli/` uses `logf`/`logLn` LogLevel system |

---

## 6. Import Hygiene

| Issue | Location | Details |
|-------|----------|---------|
| **`_ "embed"` import** | `internal/cli/html.go:4` | Correctly used for `//go:embed templates/report.html`. Fine. |
| **`regexp` in `github.go`** | `internal/cli/github.go:11` | Used for `refs/pull/(\d+)/` regex in `getPRNumber`. Could use `strings.TrimPrefix` + `strings.Split` instead, but regex is clearer here. Acceptable. |
| **`go-toml/v2`** | `go.mod` | Only used in `config_parsers.go` for TOML config parsing. Necessary for `pyproject.toml` / `ruff.toml` support. |
| **`stretchr/testify`** | `go.mod` | Only used in `tests/tree_sitter_typescript_test.go`. Inconsistent with the rest of the test suite. |
| **`context` import** | `tree_sitter_python.go:5`, `tree_sitter_typescript.go:5`, `java_parser.go:5`, `csharp_parser.go:5` | All use `context.Background()` for `ParseCtx`. This is fine for CLI usage but could be more flexible if callers provided a context. |

---

## 7. Documentation Drift

| STATUS.md says | Actual state |
|----------------|-------------|
| `internal/cli/cmd.go` — "Subcommands & Workspace Jailing" | cmd.go still has FindFiles, FormatResultsCLI, hasViolations, print functions, writeReports, CheckDiff logic — far more than just "subcommands" |
| Missing from STATUS.md architecture: `internal/cli/policy.go` | Actually exists and is listed |
| Missing from STATUS.md architecture: `internal/sensors/config_detector.go`, `git_diff.go`, `pathutils.go`, `subprocess.go` | These files exist but are not listed in the STATUS.md architecture diagram |
| Missing from STATUS.md architecture: All parser files (`eslint_parser.go`, `biome_parser.go`, `pylint_parser.go`, `ruff_parser.go`, `rubocop_parser.go`, `standardrb_parser.go`, `golangci_parser.go`) | These exist but are not in the diagram |
| Missing from STATUS.md architecture: `architecture_parser.go` | Exists but not in the diagram |
| STATUS.md says `internal/lsp/server.go` — "Real-time IDE feedback (mutex-protected writes)" | Accurate — the mutex fix is in place |
| STATUS.md says `tests/golden_test.go` — "Validates formatted LLM prompts" | Oversimplified — it actually validates full scan results against golden JSON snapshots from real repos |
| Missing tests from STATUS.md: `checkdiff_severity_test.go`, `close_leak_test.go`, `component_test.go`, `tree_sitter_python_test.go`, `tree_sitter_typescript_test.go` | These exist in `tests/` but aren't listed |
| STATUS.md Item 9: "Deduplicate Skill Definitions" | `.gemini/` directory no longer exists in the repo, so this may be resolved or the `.gemini/` was cleaned up |

---

## 8. Error Handling Gaps — ✅ `os.Exit` Resolved

| Issue | Status | Details |
|-------|--------|---------|
| **`os.Exit(1)` in library-style functions** | ✅ Fixed | `executeRun`, `executeGenerate`, `executeBootstrap` now return `error`. Errors propagate to `main()` via kong. |
| **Ignored error from `filepath.Abs`** | Open | `absTargetDir, _ := filepath.Abs(targetPath)` — LOW severity |
| **Ignored error from `os.ReadFile`** | Open | `bootstrap.go` silently skips on read failure — MEDIUM |
| **Ignored error from `yaml.Unmarshal`** | Open | Silently produces zero values on parse failure — MEDIUM |
| **Ignored error from `toml.Unmarshal`** | Open | Same as yaml — MEDIUM |
| **Ignored tree-sitter parse errors** | Open | Returns empty violations on nil tree — LOW |
| **`captureOutput` test helper race condition** | Open | Modifies process-level state — LOW |
| **Missing error on empty violation body** | ✅ Fixed | `buildPRCommentBody` now includes all 5 rules, so body won't be empty for Cog/Case violations |

---

## 9. Concurrency Issues

| Issue | Location | Status |
|-------|----------|--------|
| **LSP JSON-RPC write race** | `internal/lsp/server.go` | **FIXED** — `jsonRPCWriter` with `sync.Mutex` serializes all writes. |
| **Orchestrator `analyzeInChunks` concurrent map write** | `internal/sensors/orchestrator.go:228-260` | **SAFE** — Uses `sync.Mutex` around map writes and `errgroup` for coordination. |
| **`ScanFiles` concurrent result collection** | `internal/cli/run.go:51-101` | **SAFE** — Uses `sync.Mutex` for `allResults` append. |
| **Architecture config cache** | `internal/sensors/go_architecture.go:12-15` | **SAFE** — Uses `sync.RWMutex` for the global cache. |
| **`captureOutput` test helper** | `internal/cli/cli_test.go:953-978` | **POTENTIAL** — Modifies `os.Stdout`/`os.Stderr` without synchronization. Safe only because tests run sequentially within a package by default. |
| **`GlobalRegistry` init-time registration** | `internal/sensors/plugin.go:54-76` | **SAFE** — All registrations happen in `init()` before any goroutines start. Read-only after init. |
| **Potential concern: LSP `didChange` goroutine** | `internal/lsp/server.go:236-278` | The anonymous goroutine captures `file`, `uri`, `lang` by value (good). The `writer.sendNotification` call is mutex-protected (good). However, if the main loop reads the next request while a goroutine is still processing, there's no backpressure — many `didChange` events could queue up. This is a performance concern, not a correctness bug. |

---

## 10. Security Concerns — ✅ Path Prefix Hardened

| Issue | Status | Details |
|-------|--------|---------|
| **Path traversal protection exists** | ✅ Handled | `sanitizePath` rejects null bytes and `..` prefixes |
| **Workspace jailing in `FindFiles`** | ✅ Handled | `checkWalkDirPath` resolves symlinks and checks prefix |
| **`strings.HasPrefix` path prefix check** | ✅ Fixed | Now uses `absPath != absTargetDir && !strings.HasPrefix(absPath, absTargetDir+string(filepath.Separator))` |
| **`sanitizePath` only checks `..` prefix** | Open | LOW — callers need to verify resolved path stays within workspace |
| **GitHub token in HTTP request** | ✅ Handled | Uses `Authorization: Bearer` from env var |
| **No SSRF protection in GitHub API URL** | Open | LOW — CI environment is trusted |
| **Untracked file end-line sentinel** | ✅ Fixed | Now uses `UntrackedFileEndLine` constant |
| **No input validation on plugin Analyze args** | Open | LOW — `exec.Command` doesn't use shell |
| **Regex DoS in config_parsers.go** | Open | MEDIUM — `MaxFileSize` limit provides practical bound |

---

## 11. Recommended Fix Plan — Sprint 2 Progress

**Priority 1 — Bugs:** ✅ All 4 resolved (`.java`, PR reporting, HTML counting, Python length)

**Priority 2 — Dead code:** ✅ All 4 resolved

**Priority 3 — Constants:** ✅ All 6 resolved

**Priority 4 — Consistent logging:** ✅ Both items resolved

**Priority 5 — DRY limit-lookup:** ✅ Resolved (`getEffectiveLimits` + `EffectiveLimits`)

**Priority 6 — Test coverage:** Open (biome_plugin, ruff/standardrb subprocess tests, FindFiles, processViolationsMap)

**Priority 7 — Architecture:** Open (orchestrator.go dismantling, cmd.go extraction)

**Priority 8 — Documentation:** ✅ Resolved (STATUS.md architecture diagram + test files updated)
