# Project Status 📡

**Last Updated:** 2026-05-21  
**Branch:** `main`  
**State:** 🟢 Stable (Enterprise Hardening & Hybrid Plugin Architecture Complete)

---

## 🏗️ Current Architecture (Hybrid Plugin Model)

The codebase has transitioned to a highly scalable **Hybrid Plugin Architecture**. The CLI acts purely as the gatekeeping framework, while metrics extraction is delegated through a strict Plugin Interface (`Analyze(filePaths []string)`).

```
maintainability-sensors/
├── cmd/
│   └── maintainability-sensors/
│       └── main.go                  # CLI entrypoint
├── internal/
│   ├── cli/
│   │   ├── cmd.go                   # Subcommands & Workspace Jailing
│   │   ├── html.go                  # Statically cached HTML scorecard generator
│   │   └── github.go                # Enterprise GitHub integration
│   ├── lsp/
│   │   └── server.go                # Real-time IDE feedback server
│   └── sensors/
│       ├── plugin.go                # Core Plugin Interface & Registry
│       ├── orchestrator.go          # Argument Chunking & Plugin Invocation
│       ├── go_ast.go                # Native Go Plugin 
│       ├── tree_sitter_python.go    # Native Python Plugin (Tree-sitter)
│       ├── tree_sitter_typescript.go # Native TS/JS Plugin (Tree-sitter)
│       ├── csharp_parser.go         # Native C# Plugin (Tree-sitter)
│       ├── java_parser.go           # Native Java Plugin (Tree-sitter)
│       └── bootstrap.go             # Enterprise-safe config generator
├── skills/
│   ├── modularity-reviewer/         # Tier 2 Agent Skill (Semantic Review)
│   └── pre-flight-check/            # Tier 2 Agent Skill (Enforces Checks)
└── tests/
    └── golden_test.go               # Validates formatted LLM prompts
```

---

## ✅ Completed Work (Enterprise Hardening Sprint)

This sprint focused on making the tool robust enough for strictly regulated, massive-scale CI/CD pipelines.

| Category | Achievements |
|---|---|
| **Hybrid Plugins** | Replaced brittle monolithic switch statements with a scalable `Plugin` interface and global registry. |
| **Native Execution** | Built native AST parsers (`go-tree-sitter`) for C# and Java. No JVM or .NET SDK required. |
| **OOM Protection** | Implemented streaming JSON parsing and 2MB file-size limits. Replaced `CombinedOutput()` buffers. |
| **Scale & Speed** | Upgraded to `filepath.WalkDir` and implemented 300-file argument chunking to prevent OS `ARG_MAX` panics on huge monorepos. |
| **Security Bounding**| Enforced a strict Workspace Jail to prevent absolute path traversal and added POSIX `--` delimiters to prevent command injection. |
| **Compliance** | Added GitHub API URL overrides (`GITHUB_API_URL`) and 10-second HTTP timeouts for air-gapped CI proxies. |
| **Correctness** | Fixed Go AST metrics leaking complexity from inner closures and fixed error swallowing during initializations. |

---

## 🚧 Known Issues & Technical Debt

Despite recent enterprise hardening, several deep systemic flaws were identified during a multi-persona meta-review that require future attention:

### Security & Resource Governance
| Issue | Impact | Notes |
|---|---|---|
| **Missing POSIX `--` Delimiters** | High | Only implemented for StandardRB; missing in ESLint, PyLint, Ruff, etc., leaving them vulnerable to command/flag injection. |
| **Missing Subprocess Timeouts** | High | Linter executions (`runLintCommandJSON`) lack `context.WithTimeout`, meaning stalled linters will hang the CI runner indefinitely. |
| **Untracked File OOM Risk** | High | `git ls-files` bypasses the `WalkDir` 2MB file size limit, meaning massive untracked generated files can still cause OOM crashes. |

### Architecture & Agent-UX (DevEx)
| Issue | Impact | Notes |
|---|---|---|
| **Serial Chunking Bottleneck** | Medium | The 300-file chunking processes sequentially. On massive monorepos, this severely impacts execution time. |
| **Stdout Pollution / JSON Corruption** | High | Human-readable diagnostics (`fmt.Println`) are mixed into standard output, instantly corrupting the payload if an agent expects pure JSON. |
| **Persona Mismatch & Context Bloat** | Medium | Hardcoded "Tell your agent..." messages confuse autonomous tools, and leftover `DEBUG:` prints in `check-diff` aggressively bloat LLM context windows. |
| **Regex String Parsing** | Low | Tier 2 wrappers still rely on regex against English message strings (e.g., "complexity of 15"). |

---

## 🌐 Supported Languages

| Language | Native Parsing (Tier 1) | Orchestrated Plugin (Tier 2) | Bootstrap Config |
|---|---|---|---|
| **Go** | ✅ AST parser (`go-tree-sitter`) | - | `.golangci.yml` (with revive) |
| **Java** | ✅ AST parser (`go-tree-sitter`) | - | `checkstyle.xml` |
| **C#** | ✅ AST parser (`go-tree-sitter`) | - | `.editorconfig` |
| **TypeScript/JS** | ❌ | ESLint, Biome | `eslint.config.js`, `biome.json` |
| **Python** | ❌ | PyLint, Ruff | `.pylintrc`, `ruff.toml` |
| **Ruby** | ❌ | RuboCop, StandardRB | `.rubocop.yml`, `.standard.yml` |

---

## 📈 Metrics

| Metric | Value |
|---|---|
| **Binary Architecture**| Static Go Binary (Hybrid Tier 1/2 Plugins) |
| **Supported Languages** | 6 |
| **Security Bounding**  | Workspace Jail + POSIX Injection Blocks |
| **Test Coverage**      | 100% Green (Unit, E2E Subprocess, Golden Snapshots) |

---
