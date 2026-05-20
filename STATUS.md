# Project Status 📡

**Last Updated:** 2026-05-20  
**Branch:** `main`  
**State:** 🟢 Stable (Architecture Rewrite Complete)

---

## 🏗️ Current Architecture

The codebase has recently undergone a major architectural overhaul, resulting in a highly modular, easily testable design that strictly adheres to its "Ultra-Fast Static Binary" constraint.

```
maintainability-sensors/
├── main.go                          # CLI entrypoint
├── cli/
│   ├── cmd.go                       # Subcommands (run, generate, bootstrap)
│   ├── html.go                      # HTML scorecard generator
│   ├── github.go                    # GitHub Actions & PR comment integration
│   ├── cli_test.go                  # Unit tests for CLI commands (46 tests)
│   └── templates/report.html        # Embedded dark-themed report
├── sensors/
│   ├── orchestrator.go              # Subprocess executor + linter JSON parser
│   ├── config_parsers.go            # Native YAML/INI subset parser & JS config parser
│   ├── eslint_parser.go             # ESLint config parser
│   ├── pylint_parser.go             # PyLint config parser
│   ├── golangci_parser.go           # golangci-lint config parser
│   ├── rubocop_parser.go            # RuboCop config parser
│   ├── go_ast.go                    # Native Go AST metric collector (McCabe)
│   ├── bootstrap.go                 # Pristine config template generator (Polyglot)
│   ├── constants.go                 # Baseline threshold constants
│   ├── csharp_parser.go             # Stub (external tooling required)
│   ├── parsers_test.go              # Unit tests for config parsers
│   ├── sanitize_test.go             # Unit tests for path sanitization
│   └── subprocess_test.go           # Unit tests for subprocess error branches
├── tests/
│   ├── orchestrator_test.go         # Go AST & Level 0 fallback tests
│   ├── component_test.go            # High-level polyglot e2e integration test
│   ├── relaxed_limits_test.go       # Relaxed limit detection tests
│   ├── golden_test.go               # Golden snapshot tests (Real metrics)
└── docs/
    ├── GITHUB_ACTIONS_GUIDE.md      # CI/CD integration guide
    ├── AI_AGENT_FEEDBACK_LOOP.md    # Agent self-correction loop guide
    └── CASE_STUDIES.md              # 6-repository architectural deep-dive
```

---

## ✅ Completed Work (May 2026 Sprint)

The recent sprint focused on eradicating technical debt, expanding modern framework support, and guaranteeing test reliability.

| Category | Achievements |
|---|---|
| **Architecture** | Refactored the `executeRun` monolith into a decoupled `FindFiles` -> `ScanFiles` -> `FormatResultsCLI` pipeline. |
| **Bootstrapping** | Overhauled `BootstrapRepo` to correctly support modern monorepos by configuring all detected languages. |
| **Config Parsing** | Replaced fragile regex with a robust, stack-based YAML/INI parser in `config_parsers.go`. |
| **Modernization** | Added complete parsing support for ESLint 9 Flat Configs (`eslint.config.js`, `eslint.config.mjs`). |
| **Validation** | Implemented structural JSON schema validation in the `generate` subcommand to eliminate cryptic crashes. |
| **Testing** | Expanded test suites to cover CLI commands, subprocess branches, and regenerated golden snapshots with real metrics. |
| **CI Gatekeeping** | Configured CLI to return non-zero exit codes when code violates maintainability limits, enabling strict CI/CD enforcement. |
| **AST Parsing** | Fixed Go AST native metric collector to accurately incorporate binary expressions (`&&`, `||`) into cyclomatic complexity. |

---

## 📊 Test Status

```
✅ sensors/ package:     65+ subtests (parsers, sanitization, config detection, subprocess boundaries)
✅ cli/ package:         46 unit tests (pipeline logic, GitHub integration, input validation)
✅ tests/ package:       Integration + golden snapshot tests (5 real-world repos)
✅ Race detector:        PASS
✅ go vet:               CLEAN
✅ Build:                SUCCESS
```

---

## 🚧 Known Issues & Technical Debt

All major technical debt has been resolved. The remaining items represent minor optimizations or ecosystem constraints.

### Low Priority

| Issue | Impact | Notes |
|---|---|---|
| **`ParseCSharp` always returns error** | None | Intentional stub; kept for API compatibility until native parsing is viable. |

---

## 🌐 Supported Languages

| Language | Native Parsing | Orchestrated Tool | Bootstrap Config |
|---|---|---|---|
| **Go** | ✅ AST parser | golangci-lint | `.golangci.yml` |
| **TypeScript/JS** | ❌ | ESLint, Biome | `.eslintrc.json`, `eslint.config.js`, `biome.json` |
| **Python** | ❌ | PyLint, Ruff | `.pylintrc`, `ruff.toml` |
| **Ruby** | ❌ | RuboCop, StandardRB | `.rubocop.yml`, `.standard.yml` |
| **Java** | ❌ | Checkstyle | `checkstyle.xml` |
| **C#** | ❌ | Roslyn analyzers | `.editorconfig` |

---

## 📈 Metrics

| Metric | Value |
|---|---|
| **Binary Architecture**| Static Go Binary (stdlib only) |
| **Supported Languages** | 6 |
| **Case Studies** | 6 real-world repos |
| **CLI Test Coverage** | Comprehensive (pipeline boundaries, error handling, formatting) |
| **Subprocess Coverage** | Extensive (exit codes, missing tools, parsing crashes) |

---

*This file is auto-generated. Update after significant changes.*va** | ❌ | Checkstyle | `checkstyle.xml` |
| **C#** | ❌ | Roslyn analyzers | `.editorconfig` |

---

## 📈 Metrics

| Metric | Value |
|---|---|
| **Binary Architecture**| Static Go Binary (stdlib only) |
| **Supported Languages** | 6 |
| **Case Studies** | 6 real-world repos |
| **CLI Test Coverage** | Comprehensive (pipeline boundaries, error handling, formatting) |
| **Subprocess Coverage** | Extensive (exit codes, missing tools, parsing crashes) |

---

*This file is auto-generated. Update after significant changes.*