# Radically Candid Code Review — 2026-05-24

> **Reviewer:** OpenCode (autonomous agent)
> **Scope:** Full repository audit of `maintainability-sensors`
> **Status:** Foundation is solid. The walls have cracks. **Do not tag 1.0 yet.**

---

## Executive Summary

This codebase is a **genuinely ambitious, well-architected tool trapped in the body of a project that believes it's finished when it's only 70% there.** The Two-Tier architecture (fast stateless sensors + deferred semantic AI skills) is smart. The bootstrap safety guardrails are correct. The real-world golden tests against FastAPI, NestJS, and Go stdlib are excellent engineering.

But the repository suffers from a **false sense of completion.** Critical concurrency bugs, functional bugs in configuration exception handling, dangerously low test coverage in the core `internal/sensors` package (25.4%), and an 872-line CLI kitchen sink mean this tool is **not yet safe to gate production CI pipelines.** It needs a hardening sprint before it can be called "Stable."

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

### 1. LSP Server Has a Fatal Race Condition (`internal/lsp/server.go:200-242`)
**Severity: CRITICAL — LSP Unsafe for Production**

The `textDocument/didChange` handler spawns an anonymous goroutine for every keystroke. Multiple goroutines can call `sendNotification` concurrently, all writing to `os.Stdout` without any mutex or channel serialization. **Two concurrent diagnostics will interleave on stdio, producing corrupted JSON-RPC messages and crashing the LSP client.** This is not a minor bug; it makes the LSP server unsafe for real-world use inside any IDE.

### 2. `hasViolations` Ignores Config Exceptions (`internal/cli/cmd.go:488-496` vs `799-817`)
**Severity: CRITICAL — CI Build-Breaker**

A **functional CI-breaking bug.** The `hasViolations` function checks exception rule names using human-readable strings: `"Cyclomatic Complexity"`, `"Function Length"`, etc. But the `getLimitsForFile` function correctly aliases both `"Cyclomatic Complexity"` and `"Complexity"`. If a parser returns `RuleName: "Complexity"` (which they do), `hasViolations` fails to apply the exception and will **incorrectly fail the build** on a file that is within its relaxed thresholds. The CLI will report violations, exit code 1, and block the PR for no reason. **Two functions in the same file disagree on the canonical rule name.** Unacceptable for a CI gate.

### 3. Python Complexity Is Intentionally Wrong (`internal/sensors/tree_sitter_python.go:101-127`)
**Severity: HIGH — False Confidence for Python Users**

The cyclomatic complexity counter for Python only increments on `if_statement`, `elif_clause`, `for_statement`, `while_statement`, and `except_clause`. It explicitly ignores `try` (a branch), `with`, Boolean operators, ternary expressions, comprehensions, and `assert`. The source comment admits it: *"Our Go test just adds if, for, while, except and elif. Let's stick to the basic nodes."* **A maintainability sensor that intentionally under-reports complexity to make its own tests pass is worse than no sensor at all.** It gives Python developers false confidence.

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

### `cmd.go` Is a Kitchen Sink (872 Lines)
This file defines CLI structs, runs scanning, runs delta analysis, finds files, formats output (JSON, Markdown, HTML, table), prints self-correction guidance, writes reports, posts GitHub reviews, writes step summaries, and parses JSON scorecards. It violates the Single Responsibility Principle aggressively.

### Duplicated Skill Definitions
Both `skills/modularity-reviewer/SKILL.md` and `.gemini/skills/modularity-reviewer/SKILL.md` exist. Same for `pre-flight-check` and `performance-benchmarker`. The `.gemini/` versions appear to be direct copies. DRY applies to meta-content too.

### Inconsistent Rule Name Handling Across the Codebase
Rule names are not centralized. `"Complexity"`, `"Cyclomatic Complexity"`, `"FunctionLength"`, `"Function Length"`, `"CognitiveComplexity"`, `"Cognitive Complexity"` are all scattered. This is exactly the kind of stringly-typed logic that creates the `hasViolations` bug. There should be a `const` block of canonical rule identifiers.

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

## Maturity Verdict

| Dimension | Rating | Notes |
|---|---|---|
| Architecture | **A-** | Two-Tier is correct. Native ASTs are correct. |
| Core Logic Quality | **C+** | Bugs in exception matching and rule names. Python parser is weak. |
| Test Coverage | **D+** | 25.4% in core sensors is not acceptable for a CI gate. |
| Code Organization | **C** | `cmd.go` and `orchestrator.go` are still too large. |
| Safety Guardrails | **A** | Bootstrap non-destructiveness is perfect. |
| Production Readiness | **C-** | The LSP race condition and `hasViolations` bug make it unsafe for critical paths. |

---

## Bottom Line

> **The foundation is solid. The walls have cracks. Patch them before you invite the world inside.**

This tool should not be tagged as 1.0 Stable. It needs a focused hardening sprint. See `STATUS.md` for the remediation plan.
