# Maintainability Sensors — Agent Operational Protocol 📡

This document defines the high-assurance operational standards and workflows for the `maintainability-sensors` CLI repository. 

Every AI assistant or developer working on this codebase must respect these rules.

---

## 🏛️ Repository Architecture

This tool is designed to be a lightweight, ultra-fast Go CLI utility that orchestrates local static analysis and parses ASTs natively.

```
/
├── main.go               # CLI entrypoint
├── cli/
│   ├── cmd.go            # Subcommands (run, generate, bootstrap) & flag parsing
│   ├── html.go           # HTML scorecard generator (embeds report.html template)
│   ├── github.go         # GitHub Actions step summary & PR comment poster
│   ├── cli_test.go       # Unit tests for CLI commands (44 tests)
│   └── templates/
│       └── report.html   # Dark-themed HTML scorecard template
├── sensors/
│   ├── orchestrator.go   # Subprocess executor and linter JSON parser
│   ├── config_parsers.go # ConfigParser interface + shared utilities
│   ├── eslint_parser.go  # ESLint config parser
│   ├── pylint_parser.go  # PyLint config parser
│   ├── golangci_parser.go # golangci-lint config parser
│   ├── rubocop_parser.go # RuboCop config parser
│   ├── go_ast.go         # Native Go AST metric collector
│   ├── bootstrap.go      # Pristine config file template generator
│   ├── constants.go      # Baseline threshold constants (complexity, length, params)
│   ├── csharp_parser.go  # Stub for C# metrics (external tooling required)
│   ├── parsers_test.go   # Unit tests for config parsers (30+ tests)
│   ├── sanitize_test.go  # Unit tests for path sanitization
│   └── subprocess_test.go # Unit tests for subprocess error branches
├── tests/
│   ├── orchestrator_test.go   # Go AST & Level 0 fallback unit tests
│   ├── bootstrap_test.go      # Bootstrap template and overwrite guardrail tests
│   ├── relaxed_limits_test.go # Relaxed limit detection tests
│   ├── golden_test.go         # Golden snapshot tests for real-world repos
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
go build -o bin/maintainability-sensors main.go

# Scan the current folder
./bin/maintainability-sensors run .

# Bootstrap a directory
./bin/maintainability-sensors bootstrap /path/to/repo
```

---

## 📋 Standard PR Checklists for AI Agents

When modifying existing sensors or adding a new language bootstrap:
1. **Spec First:** Define the language limits and expected linter patterns.
2. **Test First (TDD):** Implement table-driven tests inside the `tests/` directory and ensure they fail. **Testing Policy:** Prefer component/integration tests over testing implementation details with unit tests. Only add unit tests for highly complex, isolated logic (e.g., metric extraction from ASTs).
3. **Implement Cleanly:** Write the minimum code inside the `sensors/` package to pass the tests.
4. **Compile & Verify:** Confirm that `go test ./...` passes beautifully in milliseconds and the compiled binary functions as expected.
5. **Subagent Protocol (Stop & Report):** If you encounter any blocking issues, ambiguous requirements, or areas that warrant architectural questions during execution, you MUST stop and report back to the orchestrating agent immediately. Do not guess or force a fragile solution.
6. **Multi-Persona Self-Review:** For any significant architectural changes or large features, you MUST pause and conduct a rigorous self-review using the multi-persona protocols defined in `docs/AI_REVIEW_PROTOCOLS.md`. Do not simply accept the "happy path" completion.
7. **Commit Often:** Always commit changes after each significant step, rather than waiting until the end of a long feature or refactoring session. Ensure changes are checkpoints safely along the way.
