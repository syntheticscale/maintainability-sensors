# Maintainability Sensors for Coding Agents 📡

> **The Community Reference Implementation**  
> *Inspired by Birgitta Böckeler's article ["Maintainability sensors for coding agents"](https://martinfowler.com/articles/sensors-for-coding-agents.html) on MartinFowler.com.*

---

## 🛑 The "Big Ball of Mud" Problem

If you are using AI coding agents (like Cursor, Claude Code, or GitHub Copilot Workspace), you've likely noticed a dangerous trend: **Agents are incredible at writing code fast, but they can struggle to write code that is easy to maintain.**

If you just give an agent a feature request and walk away, it will almost always take the path of least resistance. It will write 200-line monolithic functions, duplicate logic instead of extracting helpers, and nest `if` statements 8 levels deep. Over a few weeks, your clean architecture rapidly degrades into a difficult-to-maintain **"Big Ball of Mud."**

In an AI-augmented software development lifecycle, **generating code is no longer the bottleneck—enforcing architectural boundaries is.**

## 🛡️ The Solution: Active Guardrails & Skills

**Maintainability Sensors** are active, automated feedback loops designed specifically to keep AI agents in check. 

Instead of passive `README` guides that agents ignore, or blunt CI pipelines that frustrate humans, sensors act as **real-time AI Skills**. They measure code health (cyclomatic complexity, function length, parameter counts) during the active coding phase. Crucially, when an agent writes a monolithic block of code, the sensor returns a **highly structured self-correction prompt**, nudging the AI to refactor its own code or explicitly declare an exception *before* a human ever has to review it.

---

## 🚀 Key Features

* **Ultra-Fast Static Binary:** Built as a single, static Go binary. No databases, no telemetry, no warm-up times. 
* **Dual-Purpose Architecture:** Operates in *Audit Mode* to map legacy debt, and *Delta Mode* to act as an active, localized Skill for AI coding assistants.
* **Orchestration Architecture:** Auto-detects and orchestrates your local static analysis tools (ESLint, Biome, PyLint, Ruff, Go Vet, RuboCop, StandardRB) alongside native `tree-sitter` parsing for compiled languages (Java, C#, Go).
* **Agent Self-Correction Formatter:** Converts standard linter outputs into rich, high-context prompts. It tells the agent exactly *why* and *how* to refactor (e.g., *"Nudge coding agent to extract nested conditionals into separate, single-responsibility helper functions"*).
* **The `bootstrap` Accelerator:** If your repo has no existing rules (Level 0), `bootstrap` programmatically writes pristine linter configs for **TypeScript, Python, Go, Java, C#, and Ruby** enforcing strict maintainability thresholds directly to your project root.

---

## 📦 Installation

```bash
# Build the binary
go build -o bin/maintainability-sensors main.go

# Install to path
chmod +x bin/maintainability-sensors
mv bin/maintainability-sensors /usr/local/bin/
```

### Pre-Commit Hook (Dogfooding)
To ensure agents and developers self-correct before pushing, you can set up a pre-commit hook that runs the same test suite as your CI pipeline alongside `check-diff`:

```bash
cat << 'EOF' > .git/hooks/pre-commit
#!/bin/sh
go test ./...
TEST_STATUS=$?

go run main.go check-diff
DIFF_STATUS=$?

if [ $TEST_STATUS -ne 0 ] || [ $DIFF_STATUS -ne 0 ]; then
  echo "Pre-commit checks failed."
  exit 1
fi

exit 0
EOF
chmod +x .git/hooks/pre-commit
```

---

## 🚦 Usage & Commands

### Global Flags
* `-q`, `--quiet`: Suppress non-critical diagnostic output (stderr). Useful for CI/CD pipelines to reduce noise.

### 1. `check-diff` (Delta Mode: The Agent Skill)
The primary operational mode for AI agents. It analyzes `git diff HEAD` (or a specific branch) and cross-references it with maintainability violations, only alerting on code the agent actively modified. This catches new architectural rot instantly without punishing the agent for legacy debt.

```bash
# Analyze uncommitted changes in the working tree
maintainability-sensors check-diff

# Analyze changes against a specific base branch
maintainability-sensors check-diff origin/main
```

### 2. `run` (Audit Mode: Legacy Analysis)
Scan a specific file or your entire repository. Used by Tech Leads to generate a one-time scorecard or map out the existing legacy debt of the codebase.

Audit Mode will exit with a non-zero status (`Exit Code: 1`) if it finds files without static analysis tooling (Running "Blind") or if files violate the configured thresholds.

**How to run a complete Audit:**
```bash
# 1. Build the binary (if not already built)
go build -o bin/maintainability-sensors main.go

# 2. Run the audit across the repository
./bin/maintainability-sensors run .

# 3. If files are running blind, bootstrap configuration files for those languages
./bin/maintainability-sensors bootstrap .

# 4. Write visual reports to investigate violations
./bin/maintainability-sensors run . --markdown-out=report.md --html-out=report.html --json-out=report.json
```

**Fixing Audit Failures:**
To get a clean Audit (Exit Code 0), you must either refactor the failing code to pass the strict baseline thresholds (Complexity <= 8, Length <= 50, Params <= 4) or use **Elastic Thresholds** to relax the limits in the corresponding configuration file (e.g., `.golangci.yml`, `.eslintrc.json`, `.pylintrc`). The CLI parser will automatically read these relaxed config values as exceptions and allow the audit to pass, while flagging them for human review.


### 3. `generate` (Report Reconstruction)
Reconstruct visual reports from a saved JSON scorecard.

```bash
maintainability-sensors generate report.json --html-out=report.html --markdown-out=report.md
```

### 4. `bootstrap` (Environment-Hardening)
Auto-detects the languages in your codebase and writes pristine, ready-to-use maintainability configurations enforcing strict thresholds:
* **File Length:** max 300 lines
* **Function Length:** max 50 lines
* **Argument Count:** max 4 parameters
* **Cyclomatic Complexity:** max 8 limit

```bash
# Bootstrap strict thresholds
maintainability-sensors bootstrap /path/to/repo

# Bootstrap with a permissive warn-only policy for check-diff
maintainability-sensors bootstrap /path/to/repo --with-warn-policy
```

---

## 🔬 The AI-Trap Matrix (Case Studies)

Is cyclomatic complexity really that big of a deal for AI? Yes. 

To prove the necessity of maintainability sensors, we audited six high-profile, production-grade repositories. We found that when human developers write highly complex, monolithic functions, they inadvertently create **"AI Traps"**—areas of code where coding agents experience reasoning freezes or introduce catastrophic bugs.

**The Production Complexity Validation Matrix** below remains highly accurate. It illustrates exactly *why* you must prevent your agents from writing code that looks like this:

| Repository | Target Function | The Engineering Trade-off | The AI Agent Risk (Why it fails) |
|---|---|---|---|
| **`go-chi/chi`** | `tree.go` -> `findRoute` | Extreme execution speed & zero heap-allocations | High risk of introducing concurrency or path matching regressions due to massive variable scope. |
| **`psf/requests`** | `adapters.py` -> `HTTPAdapter.send` | Evolutionary drag & accumulating OS exceptions | Fragmented error handling; agents break exception mappings because the context window is overwhelmed. |
| **Go Std Library** | `net/http/server.go` -> `serve` | Inherent complexity of stateful network protocols | The state machine is too deeply nested. Agents cannot safely modify it without breaking HTTP/1.x invariants. |
| **`tiangolo/fastapi`** | `dependencies/utils.py` | High-throughput polymorphic request parsing | Agents easily break type-casting or security parameter validation when modifying the massive 100+ line parser. |
| **`nestjs/nest`** | `packages/core/injector` | Runtime reflection & complex DAG DI resolution | AI recursion traps, stack overflows, or memory leaks when trying to trace circular dependencies across scopes. |

### 📖 Read the Full Deep-Dive
For a detailed analysis of how these systems act as AI traps and exactly how they should be refactored to be "agent-friendly", read our:
👉 **[Architectural Case Studies deep-dive (docs/CASE_STUDIES.md)](docs/CASE_STUDIES.md)**

---

## 🧩 Architectural Philosophy

This CLI is designed around three strict architectural rules:
1. **Statelessness (Local-First):** Inputs are your local files and git; outputs are stdout. No external telemetry or remote databases.
2. **Orchestration over Re-implementation:** Leverage standard local compilers and AST tools rather than re-writing syntax parsers.
3. **Deterministic Feedback Loop:** Errors and warnings are structured to provide explicit refactoring guidance specifically formatted for LLM coding agents.

---

## 📄 License

Created by [Paulo Lai](https://github.com/paulolai) for Synthetic Scale.

MIT © 2026 Synthetic Scale & Contributors.  
*This is an independent open-source community reference implementation and is not affiliated with Thoughtworks or Martin Fowler.*Thoughtworks or Martin Fowler.*