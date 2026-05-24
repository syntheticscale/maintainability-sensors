# Maintainability Sensors — Agent Operational Protocol 📡

This document defines the high-assurance operational standards and workflows for the `maintainability-sensors` CLI repository. 

Every AI assistant or developer working on this codebase must respect these rules.

---

## 🏛️ Repository Architecture

This tool is designed to be a lightweight, ultra-fast Go CLI utility that orchestrates local static analysis and parses ASTs natively.

```
/
├── cmd/
│   └── maintainability-sensors/
│       └── main.go               # CLI entrypoint
├── internal/
│   ├── cli/
│   │   ├── cmd.go            # Subcommands (run, generate, bootstrap) & flag parsing
│   │   ├── html.go           # HTML scorecard generator
│   │   ├── github.go         # GitHub Actions step summary & PR comment poster
│   │   ├── cli_test.go       # Unit tests for CLI commands
│   │   └── templates/
│   │       └── report.html   # Dark-themed HTML scorecard template
│   ├── lsp/
│   │   ├── server.go         # Language Server Protocol foundation
│   │   └── server_test.go    # LSP JSON-RPC parsing tests
│   └── sensors/
│       ├── orchestrator.go   # Subprocess executor and linter JSON parser
│       ├── config_parsers.go # ConfigParser interface + shared utilities
│       ├── tree_sitter_python.go  # Native Tree-sitter Python AST metrics
│       ├── tree_sitter_typescript.go # Native Tree-sitter TS/JS AST metrics
│       ├── go_ast.go         # Native Go AST metric collector
│       ├── go_architecture.go # Native architecture dependency boundary rules
│       ├── architecture_parser.go # YAML parser for dependency rules
│       ├── bootstrap.go      # Pristine config file template generator
│       └── constants.go      # Baseline threshold constants
├── skills/
│   ├── modularity-reviewer/  # Tier 2 AI Skill for Semantic Modularity Review
│   └── pre-flight-check/     # Tier 2 AI Skill for autonomous check-diff runs
├── tests/
│   ├── orchestrator_test.go   # Go AST & Level 0 fallback unit tests
│   ├── bootstrap_test.go      # Bootstrap template and overwrite guardrail tests
│   ├── relaxed_limits_test.go # Relaxed limit detection tests
│   ├── golden_test.go         # Golden snapshot tests for real-world repos
│   ├── architecture_test.go   # Component tests for layer dependency logic
│   └── multi_repo_test.go     # End-to-end CLI integration tests
└── docs/
    ├── GITHUB_ACTIONS_GUIDE.md    # CI/CD integration guide
    ├── AI_AGENT_FEEDBACK_LOOP.md  # Agent self-correction loop guide
    └── CASE_STUDIES.md            # Real-world code decay analysis
```

---

## 🧩 Architectural Constraints (ADR Rules)

1. **Stateless Execution:** The CLI must remain completely stateless. It reads local files and writes to stdout or stderr. No database dependencies, no filesystem caches, and no remote telemetry.
2. **Minimal External Dependencies:** The binary must have minimal external Go dependencies, strictly limited to standard config unmarshallers (like `yaml.v3` and `go-toml/v2`).
3. **Safety Guardrails:** The `bootstrap` command must **never** destructive-overwrite existing custom configuration files. If an existing config is found, skip writing, alert the user, and output recommended addition snippets.
4. **Agent-Facing Output:** All warnings and errors must output clear, actionable, and structured **Refactoring Prompts** specifically optimized for LLM coding agents to ingest and act on.

---

## 🚀 Standard Operational Commands

```bash
# Run all tests
/usr/local/go/bin/go test ./...

# Force a clean, non-cached test run
/usr/local/go/bin/go test -count=1 ./...

# Build the static binary
go build -o bin/maintainability-sensors ./cmd/maintainability-sensors

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
9. **Commit Often:** Always commit changes after each significant step, rather than waiting until the end of a long feature or refactoring session. Ensure changes are checkpoints safely along the way.

## 🔄 Iterative Subagent Development Loop

When executing new features, extensions, or major refactors, you **MUST** use the following strict iterative subagent loop. Do not attempt to implement large changes in a single monolithic step.

1.  **Break Down the Task:** Identify the first discrete step suitable for a subagent (e.g., the `generalist` agent).
2.  **Delegate via R-S-T-I:** Invoke the subagent and instruct it to follow the strict **Requirements -> Spec -> Tests -> Implementation** workflow.
3.  **Verify & Commit:** Upon the subagent's return, review the work, run the test suite, and commit the checkpoint.
4.  **Repeat:** Find the next logical step and repeat the cycle until the overarching goal is complete or user input is required.
