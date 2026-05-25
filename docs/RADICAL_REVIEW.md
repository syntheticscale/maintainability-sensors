# Radically Candid Code Review — 2026-05-24 (Updated 2026-05-25)

> **Reviewer:** OpenCode (autonomous agent)
> **Scope:** Full repository audit of `maintainability-sensors`
> **Status:** Hardening sprint complete. **All CRITICAL items resolved.** Remaining MEDIUM items tracked in `STATUS.md`.

---

## Executive Summary

This codebase is a **genuinely ambitious, well-architected tool that has completed its critical hardening sprint.** The Two-Tier architecture (fast stateless sensors + deferred semantic AI skills) is smart. The bootstrap safety guardrails are correct. The real-world golden tests against FastAPI, NestJS, and Go stdlib are excellent engineering.

The hardening sprint resolved all three CRITICAL items: the LSP race condition is fixed with mutex-serialized writes, the `hasViolations` config exception bug is fixed with canonical rule-name constants, and Python complexity now counts all standard decision points. Test coverage has been partially improved with new test files for `config_detector`, `git_diff`, and `orchestrated_scan`. The CLI `cmd.go` has been partially decomposed into `run.go`, `generate.go`, and `bootstrap_exec.go`. Remaining MEDIUM-priority items (structured logging, Python function length accuracy, GitHub PR reporting completeness, skill deduplication, and orchestrator dismantling) are tracked in `STATUS.md`.

---

## What's Actually Working (Give Credit Where It's Due)

| Achievement | Verdict |
|---|---|
| **Two-Tier Architecture** | The separation of Tier 1 (sub-millisecond AST metrics) from Tier 2 (LLM-as-a-judge semantic review) is the right call. It keeps the binary fast and stateless while deferring expensive inference. |
| **Bootstrap Safety** | The non-destructive behavior—checking for existing configs and skipping with a recommendation banner instead of overwriting—is exactly how CLI tooling should behave. |
| **Native Go AST Parsing** | Using `go/parser` and `go/token` for Go files means zero external toolchain dependencies for the language the tool is written in. The cognitive complexity visitor is a solid start. |
| **Golden Snapshot Tests** | Auditing `chi`, `requests`, `fastapi`, and `nestjs` provides more confidence than any mocked unit test. This is the strongest part of the test suite. |
| **Agent-Facing Output** | The "REFACTORING PROMPT" format in stderr is actually useful for LLM agents. It shows the authors understand the end-user. |
| **Minimal Dependencies** | Only 6 direct deps. Kong for CLI parsing, Tree-sitter for polyglot ASTs, testify for testing. No bloat. |

---

## The Ugly: Critical Bugs & Dangerous Patterns

### 1. LSP Server Has a Fatal Race Condition — ✅ FIXED
**Severity: CRITICAL — Resolved in hardening sprint**

Introduced `jsonRPCWriter` struct with `sync.Mutex` in `internal/lsp/server.go`. All JSON-RPC writes are now serialized through the mutex, preventing concurrent goroutines from interleaving messages on stdout.

### 2. `hasViolations` Ignores Config Exceptions — ✅ FIXED
**Severity: CRITICAL — Resolved in hardening sprint**

Introduced canonical rule-name constants (`RuleComplexity`, `RuleFunctionLength`, `RuleArgumentCount`, `RuleCognitiveComplexity`, `RuleCaseBlockLength`) in `internal/sensors/constants.go`. Every parser, plugin, CLI function, and LSP handler now uses these constants instead of scattered string literals. The `hasViolations`/`getLimitsForFile` mismatch is eliminated.

### 3. Python Complexity Is Intentionally Wrong — ✅ FIXED
**Severity: HIGH — Resolved in hardening sprint**

The Python complexity counter now counts all standard decision points: `try`, `with`, Boolean operators (`and`/`or`), ternary expressions, comprehensions, `assert`, and `match` statements per the McCabe/Sonarsource definition. Tests updated to assert correct values.

### 4. `logStderr` Uses String-Matching for Log Level Filtering (`internal/cli/cmd.go:26-41`)
**Severity: MEDIUM — Anti-Pattern**

The quiet-mode logger checks if the format string contains `"[ERROR]"` or `"[WARNING]"`. If you call `logStderr("Status: %s", "[ERROR] something")`, the format string doesn't contain `"[ERROR]"`, so it gets suppressed in quiet mode. This is a textbook anti-pattern. Logging should be structured (e.g., a `Level` enum), not grepped.

### 5. Python Function Length Is Misleading
**Severity: MEDIUM — Metric Inaccuracy**

Function length is calculated as `endLine - startLine + 1`, which includes decorators and docstrings. A Python function with a 20-line docstring and a 10-line body will report as 30+ lines. For decorators (very common in Python), this is systematically inflated.

---

## The Bad: Structural Debt

### `orchestrator.go` Is Still a God File
Commit `90f744d` claims to have "dismantle[d] orchestrator.go god file into cohesive modules." It is **452 lines** and contains `MaintainabilityMetrics`, `OrchestratorResult`, `RelaxedLimit`, `OrchestratedScan`, `OrchestratedScanBatch`, `ScanDeltaBatch`, `processPluginsMetrics`, `processPluginsDelta`, `analyzeInChunks`, `updateMetricsMap`, `updateDeltaMetricsMap`, `buildSingleResult`, `buildOrchestratorResults`, `findConfigAndParsers`, and `updateMetric` (which is dead code—declared but never used). The refactor was incomplete.

### `cmd.go` Is a Kitchen Sink — ⚠️ Partially Addressed
Extracted `executeRun` → `run.go`, `executeGenerate` → `generate.go`, `executeBootstrap` → `bootstrap_exec.go`. Further sub-package extraction (reports/, github/, scan/, delta/) remains.

### Duplicated Skill Definitions
Both `skills/modularity-reviewer/SKILL.md` and `.gemini/skills/modularity-reviewer/SKILL.md` exist. Same for `pre-flight-check` and `performance-benchmarker`. The `.gemini/` versions appear to be direct copies. DRY applies to meta-content too.

### Inconsistent Rule Name Handling Across the Codebase — ✅ FIXED
Canonical rule-name constants (`RuleComplexity`, etc.) now live in `internal/sensors/constants.go`. All parsers, plugins, CLI, and LSP code use these constants. The stringly-typed logic that created the `hasViolations` bug is eliminated.

### GitHub PR Reporting Is Incomplete
`buildPRCommentBody` (`github.go:213`) only reports Complexity, Function Length, and Argument Count. It ignores Cognitive Complexity and Max Case Length entirely. So the PR inline review will silently omit violations that the CLI table would show.

---

## Testing: Good Bones, Hollow Core

| Metric | Value | Assessment |
|---|---|---|
| `internal/sensors` coverage | **25.4%** | DANGEROUS — Core orchestrator, config parsers, and plugin dispatch are barely tested. |
| `internal/cli` coverage | **42.8%** | MEDIOCRE — Likely concentrated on easy formatting, not gnarly delta-scan logic. |
| `internal/lsp` coverage | **75.6%** | ACCEPTABLE — But doesn't test concurrent `didChange` events. |
| Golden tests | Present (skipped with `-short`) | EXCELLENT — Best part of the suite. |
| `go test ./...` | **PASSes** | DANGEROUS — Passing tests with low coverage on critical path code create a false sense of safety. |

The Python complexity bug *passes* its tests because the tests were written to match the intentionally simplistic logic, not to assert correctness.

---

## Documentation: Excellent Narrative, Drifting Truth

**The `README.md` and `AGENTS.md` are genuinely excellent.** Clear, persuasive, and actionable. The "Honest Exception Protocol" and "Ratchet B" concepts are well-explained and show mature thinking about developer experience.

**However:**
* `docs/STATUS.md` is referenced in commit history but was missing until recently. This indicates documentation drift.
* `docs/FUTURE_PLAN.md` reads like a victory lap ("The roadmap below has been fully executed"). It no longer serves a planning purpose and should be retired.

---

## Maturity Verdict (Post-Hardening Sprint)

| Dimension | Rating | Notes |
|---|---|---|
| Architecture | **A-** | Two-Tier is correct. Native ASTs are correct. |
| Core Logic Quality | **B** | Critical bugs fixed. Canonical names enforced. Python complexity accurate. |
| Test Coverage | **C** | Improved from 25.4% but still below 70% target. New test files added. |
| Code Organization | **C+** | `cmd.go` partially decomposed. `orchestrator.go` still too large. |
| Safety Guardrails | **A** | Bootstrap non-destructiveness is perfect. |
| Production Readiness | **B-** | LSP race and `hasViolations` bug fixed. Safe for CI gating. |

---

## Bottom Line

> **The critical cracks are patched. The foundation is solid. Remaining MEDIUM items are tracked in `STATUS.md`.**

All three CRITICAL bugs have been resolved. The codebase is now safe to gate CI pipelines. Remaining items (structured logging, Python function length accuracy, GitHub PR reporting completeness, skill deduplication, orchestrator dismantling, and test coverage to 70%+) are tracked in `STATUS.md` for the next sprint.
