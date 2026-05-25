# Maintainability Sensors for Coding Agents 📡

> **The Community Reference Implementation**  
> *Inspired by Birgitta Böckeler's article ["Maintainability sensors for coding agents"](https://martinfowler.com/articles/sensors-for-coding-agents.html) on MartinFowler.com.*

---

## 🛑 The "Big Ball of Mud" Problem

If you are using AI coding agents (like Cursor, Claude Code, or GitHub Copilot Workspace), you've likely noticed a dangerous trend: **Agents are incredible at writing code fast, but they can struggle to write code that is easy to maintain.**

If you just give an agent a feature request and walk away, it will almost always take the path of least resistance. It will write 200-line monolithic functions, duplicate logic instead of extracting helpers, and nest `if` statements 8 levels deep. Over a few weeks, your clean architecture rapidly degrades into a difficult-to-maintain **"Big Ball of Mud."**

In an AI-augmented software development lifecycle, **generating code is no longer the bottleneck—enforcing architectural boundaries is.**

---

## 🛡️ The Solution: A Two-Tier AI Defense Ecosystem

**Maintainability Sensors** is not just a linter; it is a complete ecosystem of active, automated feedback loops designed specifically to keep AI agents in check. 

Instead of passive `README` guides that agents ignore, or blunt CI pipelines that frustrate humans, we employ a **Two-Tier Architecture** that catches code decay before a human ever has to review it.

### Tier 1: The Guardrails (Syntactic Sensors)
A lightning-fast, stateless Go utility that provides sub-millisecond feedback on structural boundaries.
*   **Native Polyglot ASTs:** Deterministically calculates Cyclomatic Complexity (Max 8), Function Length (Max 50), and Parameter Counts (Max 4) entirely in memory for **Go, Python, TypeScript/JavaScript, C#, and Java** using Tree-sitter. Zero external toolchains required.
*   **Real-Time LSP Server:** Runs natively in the background of any modern IDE (VS Code, Cursor, Neovim) providing instant red squiggles (`textDocument/didChange`) to developers and LSP-aware agents as they type.
*   **Macro-Coupling Dependency Rules:** Enforces layered architecture boundaries (e.g., stopping the `domain` layer from importing the `api` layer) natively via AST `import` extraction.
*   **Agent Self-Correction Formatter:** Converts violations into rich, high-context **Refactoring Prompts** (e.g., *"Nudge coding agent to extract nested conditionals"*).

### Tier 2: The Intelligence (Semantic AI Skills & CI)
Slower, context-heavy operations deferred to autonomous AI Agent Skills and asynchronous CI pipelines.
*   **`pre-flight-check` Skill:** A bundled AI skill that intercepts an agent's "I am finished" state. It forces the agent to autonomously run the Tier 1 `check-diff` tools and the test suite, making it self-correct any structural decay before handing the code back to you.
*   **`modularity-reviewer` Skill:** A bundled AI skill that performs LLM-as-a-judge inferential reviews. It evaluates code for "Semantic Duplication" and "Misplaced Responsibilities" based on Vlad Khononov's modularity principles.
*   **CI Mutation Testing (Deferred):** GitHub Actions workflows that run `go-mutesting` strictly against the `git diff` to ensure AI-generated tests actually assert value.

---

## 📦 Getting Started & Installation

### 1. The Go CLI (Tier 1)
```bash
# Build the binary
go build -o bin/maintainability-sensors ./cmd/maintainability-sensors

# Install to path
chmod +x bin/maintainability-sensors
mv bin/maintainability-sensors /usr/local/bin/
```

### 3. Cross-Compilation (CGO Dependency)
Because this tool uses `go-tree-sitter` for native AST parsing, it requires `cgo` and a C compiler. To cross-compile for different operating systems or architectures, you must provide the appropriate cross-compiler via the `CC` environment variable and ensure `CGO_ENABLED=1`.

Example (Linux to Windows):
```bash
sudo apt-get install mingw-w64
CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -o bin/maintainability-sensors.exe ./cmd/maintainability-sensors
```

Example (macOS to Linux):
```bash
brew install FiloSottile/musl-cross/musl-cross
CGO_ENABLED=1 CC=x86_64-linux-musl-gcc go build -o bin/maintainability-sensors-linux ./cmd/maintainability-sensors
```

### 2. The AI Skills (Tier 2)
The repository includes `.skill` files ready for installation into compatible agents (like Gemini CLI).
```bash
# Install the autonomous pre-flight guardrail
gemini skills install pre-flight-check.skill --scope workspace

# Install the LLM-as-a-judge modularity reviewer
gemini skills install modularity-reviewer.skill --scope workspace

# Install the empirical performance benchmarker
gemini skills install performance-benchmarker.skill --scope workspace
```

---

## 🚦 CLI Reference

### 1. `check-diff` (Delta Mode: The Core Agent Loop)
The primary operational mode for AI agents. It analyzes `git diff HEAD` (or a specific branch) and cross-references it with maintainability violations, only alerting on code the agent actively modified. This catches new architectural rot instantly without punishing the agent for legacy debt.

```bash
maintainability-sensors check-diff
```

### 2. `lsp` (Real-Time Editor Squiggles)
Launch the Language Server Protocol wrapper to communicate with your IDE via `stdio`.
```bash
maintainability-sensors lsp
```

### 3. `run` (Audit Mode: Legacy Analysis)
Scan a specific file or your entire repository. Used by Tech Leads to generate a one-time scorecard or map out the existing legacy debt of the codebase.

```bash
# Run the audit across the repository
maintainability-sensors run .

# Write visual reports to investigate violations
maintainability-sensors run . --markdown-out=report.md --html-out=report.html --json-out=report.json
```

### 4. `bootstrap` (Environment-Hardening)
Auto-detects the languages in your codebase and writes pristine, ready-to-use maintainability configurations enforcing strict thresholds.

```bash
maintainability-sensors bootstrap /path/to/repo
```

---

## 🤝 Guidance for Human Developers: The "Honest Exception"

While this tool enforces strict metrics (like Cyclomatic Complexity <= 8), it is crucial to recognize that **classic metrics heavily penalize highly cohesive structures like `switch` statements**. A flat `switch` mapping 20 JSON codes is highly readable to humans but generates a complexity score of 20+.

We do NOT want you to fracture readable code into disjointed pieces just to appease a metric.

If you encounter a violation caused by highly cohesive logic (or an unavoidable legacy integration), follow the **Honest Exception Protocol**:
1. Do not fragment the code.
2. Add a standard suppression comment (e.g., `//nolint:gocognit` or `# pylint: disable=too-many-branches`) right above the offending logic.
3. **Crucially:** Add an inline comment briefly explaining *why* the suppression exists (e.g., `//nolint:gocognit // Highly cohesive mapping logic, splitting hurts readability`).

### 🔒 Legacy Audits & "Ratchet B"
If you are auditing a massive legacy file, do **not** relax the global repository thresholds. Instead, use **File-Specific Configuration Overrides ("Ratchet B")**. By adding a per-path override in your config file, you lock a specific legacy file to its current score. If a developer or AI adds even one more point of complexity tomorrow, the build will fail.

---

## 🔬 The AI-Trap Matrix (Case Studies)

To prove the necessity of maintainability sensors, we audited six high-profile, production-grade repositories (including FastAPI, NestJS, and the Go Std Lib). We found that when human developers write highly complex, monolithic functions, they inadvertently create **"AI Traps"**—areas of code where coding agents experience reasoning freezes or introduce catastrophic bugs.

👉 **[Read the Architectural Case Studies deep-dive (docs/CASE_STUDIES.md)](docs/CASE_STUDIES.md)**

👉 **[Read the Legacy Audit Guide (docs/LEGACY_AUDIT_GUIDE.md)](docs/LEGACY_AUDIT_GUIDE.md)**

---

## 📄 License

Created by [Paulo Lai](https://github.com/paulolai) for Synthetic Scale.

MIT © 2026 Synthetic Scale & Contributors.  
*This is an independent open-source community reference implementation and is not affiliated with Thoughtworks or Martin Fowler.*nce implementation and is not affiliated with Thoughtworks or Martin Fowler.*