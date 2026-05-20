# Maintainability Sensors for Coding Agents 📡

> **The Community Reference Implementation**  
> *Inspired by Birgitta Böckeler's article ["Maintainability sensors for coding agents"](https://martinfowler.com/articles/sensors-for-coding-agents.html) on MartinFowler.com.*

---

## 🛑 The "Big Ball of Mud" Problem

If you are using AI coding agents (like Cursor, Claude Code, or GitHub Copilot Workspace), you've likely noticed a dangerous trend: **Agents are incredible at writing code fast, but they can struggle to write code that is easy to maintain.**

If you just give an agent a feature request and walk away, it will almost always take the path of least resistance. It will write 200-line monolithic functions, duplicate logic instead of extracting helpers, and nest `if` statements 8 levels deep. Over a few weeks, your clean architecture rapidly degrades into a difficult-to-maintain **"Big Ball of Mud."**

In an AI-augmented software development lifecycle, **generating code is no longer the bottleneck—enforcing architectural boundaries is.**

## 🛡️ The Solution: Active Guardrails

**Maintainability Sensors** are active, automated feedback loops designed specifically to keep AI agents in check. 

Instead of passive `README` guides that agents ignore, sensors act as **hard CI/CD gatekeepers**. They measure code health (cyclomatic complexity, function length, parameter counts) in real-time. Crucially, when an agent writes a monolithic block of code, the sensor fails the build and returns a **highly structured self-correction prompt**, forcing the AI to refactor its own code *before* a human ever has to review it.

---

## 🚀 Key Features

* **Ultra-Fast Static Binary:** Built as a single, static Go binary. No databases, no telemetry, no warm-up times. Runs in microseconds (<15ms per file).
* **CI/CD Gatekeeping:** Returns a non-zero exit code (`exit 1`) when code violates maintainability baselines, ensuring unmaintainable code never merges.
* **Orchestration Architecture:** Auto-detects and orchestrates your local static analysis tools (ESLint, PyLint, Go Vet, RuboCop). It respects your project's custom linting rules rather than forcing a proprietary model.
* **Agent Self-Correction Formatter:** Converts standard linter outputs into rich, high-context prompts. When the build fails, the CLI tells the agent exactly *why* and *how* to refactor (e.g., *"Nudge coding agent to extract nested conditionals into separate, single-responsibility helper functions"*).
* **The `bootstrap` Accelerator:** If your repo has no existing rules (Level 0), `bootstrap` programmatically writes pristine linter configs for **TypeScript, Python, Go, and Ruby** enforcing strict maintainability thresholds directly to your project root.

---

## 📦 Installation

```bash
# Build the binary
go build -o bin/maintainability-sensors main.go

# Install to path
chmod +x bin/maintainability-sensors
mv bin/maintainability-sensors /usr/local/bin/
```

---

## 🚦 Usage & Commands

### 1. `run` (Scan & Gatekeep)
Scan a specific file or your entire repository. The CLI checks for local static analysis configurations. If code exceeds the complexity limits, the CLI outputs self-correction prompts and exits with code `1`.

```bash
# Scan a specific file
maintainability-sensors run src/components/MyComponent.tsx

# Scan the entire repository and fail if code is too complex
maintainability-sensors run .

# Write visual reports for humans
maintainability-sensors run . --markdown-out=report.md --html-out=report.html
```

### 2. `bootstrap` (Environment-Hardening)
Auto-detects the languages in your codebase and writes pristine, ready-to-use maintainability configurations enforcing strict thresholds to prevent agent-driven decay:
* **File Length:** max 300 lines
* **Function Length:** max 50 lines
* **Argument Count:** max 4 parameters
* **Cyclomatic Complexity:** max 8 limit

```bash
# Bootstrap local maintainability configurations for all languages in the repo
maintainability-sensors bootstrap /path/to/repo
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

MIT © 2026 Paulo Lai & Contributors.  
*This is an independent open-source community reference implementation and is not affiliated with Thoughtworks or Martin Fowler.*