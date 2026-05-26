# Maintainability Sensors — Agent Operational Protocol 📡

This document defines the high-assurance operational standards and workflows for the `maintainability-sensors` CLI repository. 

Every AI assistant or developer working on this codebase must respect these rules.

---

## 🏛️ Architecture Rules

This tool is a lightweight, ultra-fast Go CLI that orchestrates local static analysis and parses ASTs natively. The key boundaries:

- **`cmd/maintainability-sensors/`** — Core CLI. **`cmd/legacy-plugin/`** — Polyglot IPC subprocess for non-Go languages. Two binaries, both must be built.
- **`internal/plugin/protocol/`** — Single source of truth for domain constants (baselines, rule names, `ParserRule`), IPC message types, and `ProtocolVersion`. Both `internal/sensors/` and `internal/legacy/` delegate here. Do not duplicate.
- **`internal/sensors/evaluator.go`** — Canonical `EffectiveLimits` struct and `Evaluate()` function. All violation checking flows through here. CLI formatters must not reimplement evaluation logic.
- **`internal/sensors/bootstrap_*.go`** — Per-language bootstrap files. Adding a new language = one new file + a case in `bootstrapLanguage`/`detectLanguages`.
- **`internal/legacy/*_plugin.go`** — Per-linter subprocess parsers. Each shells out to a linter (ESLint, Pylint, RuboCop, etc.) and parses its JSON output.
- **`internal/lsp/`** — Experimental stub. Not production-ready.

---

## 🧩 Architectural Constraints (ADR Rules)

1. **Stateless Execution:** The CLI must remain completely stateless. It reads local files and writes to stdout or stderr. No database dependencies, no filesystem caches, and no remote telemetry.
2. **Minimal External Dependencies:** The binary must have minimal external Go dependencies, strictly limited to standard config unmarshallers (like `yaml.v3` and `go-toml/v2`).
3. **Safety Guardrails:** The `bootstrap` command must **never** destructive-overwrite existing custom configuration files. If an existing config is found, skip writing, alert the user, and output recommended addition snippets.
4. **Agent-Facing Output:** All warnings and errors must output clear, actionable, and structured **Refactoring Prompts** specifically optimized for LLM coding agents to ingest and act on.
5. **Single Source of Truth:** Domain constants (baselines, rule names, `ParserRule`) live in `internal/plugin/protocol/`. Both `internal/sensors/` and `internal/legacy/` delegate to this package via type aliases and constant assignments. Do not duplicate.
6. **Protocol Versioning:** The IPC protocol between the core and `legacy-plugin` carries a `ProtocolVersion` field. Both sides validate it on every message. Bump the version in `internal/plugin/protocol/schema.go` when changing the schema.

---

## 🚀 Standard Operational Commands

```bash
# Run all tests
make test

# Build both binaries (required for full functionality)
make build

# Scan the current folder
./bin/maintainability-sensors run .

# Bootstrap a directory
./bin/maintainability-sensors bootstrap /path/to/repo
```

Repositories can add a `.maintainability-sensors.yml` at the root to configure `check-diff` severity levels (e.g., `default-severity: warn`) for gradual adoption without modifying CI pipelines.

---

## 📋 Standard PR Checklists for AI Agents

When modifying existing sensors or adding a new language bootstrap:
1. **Spec First:** Define the language limits and expected linter patterns.
2. **Test First (TDD):** Implement table-driven tests inside the `tests/` directory and ensure they fail. **Testing Policy:** Prefer component/integration tests over testing implementation details with unit tests. Only add unit tests for highly complex, isolated logic (e.g., metric extraction from ASTs).
3. **Implement Cleanly:** Write the minimum code inside the `internal/sensors/` package to pass the tests.
4. **Compile & Verify:** Confirm that `go test ./...` passes beautifully in milliseconds and the compiled binary functions as expected.
5. **Subagent Protocol (Stop & Report):** If you encounter any blocking issues, ambiguous requirements, or areas that warrant architectural questions during execution, you MUST stop and report back to the orchestrating agent immediately. Do not guess or force a fragile solution.
6. **Semantic Modularity Review (Tier 2 Skill):** For any significant architectural changes or large features, you MUST invoke the `modularity-reviewer` AI skill to evaluate your code for semantic duplication and misplaced responsibilities. Do not simply accept the "happy path" completion.
7. **Pre-Flight Check (Tier 2 Skill):** Before reporting task completion to the user, you MUST invoke the `pre-flight-check` AI skill to autonomously run `check-diff` and the test suite. Fix any refactoring prompts generated.
8. **NFR & Performance Review (Tier 2 Skill):** Whenever refactoring for performance, evaluating Non-Functional Requirements (NFRs), or adding high-frequency features (e.g. LSP handling), you MUST invoke the `performance-benchmarker` AI skill to establish an empirical baseline. Never optimize without microbenchmarks.
9. **Documentation First:** Before finalizing any task and making the final commit, you MUST review the project's documentation (`README.md`, `STATUS.md`, `PLAN.md`, etc.) and ensure it accurately reflects the new architectural state or completed features.
10. **Commit Often:** Always commit changes after each significant step, rather than waiting until the end of a long feature or refactoring session. Ensure changes are checkpoints safely along the way.

## 🔄 Iterative Subagent Development Loop

When executing new features, extensions, or major refactors, you **MUST** use the following strict iterative subagent loop. Do not attempt to implement large changes in a single monolithic step.

1.  **Break Down the Task:** Identify the first discrete step suitable for a subagent (e.g., the `generalist` agent).
2.  **Delegate via R-S-T-I:** Invoke the subagent and instruct it to follow the strict **Requirements -> Spec -> Tests -> Implementation** workflow.
3.  **Verify & Commit:** Upon the subagent's return, review the work, run the test suite, and commit the checkpoint.
4.  **Repeat:** Find the next logical step and repeat the cycle until the overarching goal is complete or user input is required.
