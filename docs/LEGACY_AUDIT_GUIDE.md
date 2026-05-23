# Legacy Audit Guide: Mastering Maintainability Sensors

When introducing `maintainability-sensors` into a mature, battle-tested codebase, you will almost certainly be greeted by a sea of red violations. This is expected. Legacy codebases grow organically, and structural metrics (like cyclomatic complexity) are uncompromising.

This guide outlines the definitive strategy for auditing legacy systems, establishing a safe baseline, and setting the repository on a path to improvement—without breaking the build or overwhelming your engineering team (human or AI).

---

## The Philosophy: Incremental "Clean House" Maintenance

As Birgitta Böckeler notes in her foundational article *Maintainability Sensors for Coding Agents*, the goal is not to rewrite the entire legacy system on day one. The goal is to establish **hygiene guardrails** that prevent the codebase from decaying *further* while using AI to perform gradual "garbage collection" on technical debt.

### Core Tenets of a Legacy Audit:
1.  **Do not overfit to the metric.** A 100-line routing switch statement might fail a complexity check, but breaking it into 10 disconnected files makes it harder to read. Protect semantic cohesion.
2.  **Ratchet, don't relax globally.** Never lower the global standards of the repository just to pass the build. Quarantine the legacy debt using file-specific rules.
3.  **Human Partnership is Required.** Sensors catch structural hygiene issues, but humans (and high-level AI inferential analysis) are required to judge architectural trade-offs.

---

## 🛠️ Step-by-Step Audit Playbook

### Step 1: Run the Baseline Audit
Execute the sensor in "Audit Mode" to map the entire repository:
```bash
maintainability-sensors run .
```
Review the terminal output or the generated HTML scorecard. Identify the primary offenders.

### Step 2: Triage the Violations
Sort the failing files into two categories:
*   **Category A (The Spaghetti):** Deeply nested `if/for` loops, confusing boolean logic, and monolithic functions that clearly violate clean code principles.
*   **Category B (The Cohesive Monoliths):** Large, flat `switch` statements, complex dictionary mappings, or deeply coupled network protocol logic that is battle-tested and structurally sound despite failing the metric.

### Step 3: Action Plan for Category A (Refactor)
For poorly structured legacy code, use an AI coding agent to break down the monoliths.

*   **The Prompt:** Ask the agent to "Refactor this function to reduce Cognitive Complexity below 8, ensuring no function exceeds 50 lines."
*   **The Nudge:** The sensor output will provide explicit refactoring prompts (e.g., *"Extract nested conditionals into separate, single-responsibility helper functions"*).
*   **The AI Super-Pattern:** If the agent struggles with argument counts, instruct it to use the **Parameter Object Pattern** to encapsulate data passing.

### Step 4: Action Plan for Category B (The Honest Exception & Ratchet)
Do not fracture cohesive monoliths. Instead, document the architectural trade-off using one of the following methods.

#### Option 1: The Honest Exception (Inline Suppression)
If a specific function is highly cohesive (e.g., a massive JSON parser), leave the code intact and add a localized linter suppression comment.
*   **Requirement:** You *must* include a justification comment so human reviewers understand the exception.
*   **Example (Go):**
    ```go
    //nolint:gocognit,cyclop // Highly cohesive mapping logic for linters. Splitting this hurts readability.
    func parseMessages(list []Message) []Violation { ... }
    ```
*   **File-Level Suppressions:** If an entire file is a legacy nightmare that cannot be touched, you can suppress our specific structural metrics for the whole file by adding `//nolint:maintainability // Legacy Audit Exception` to the top of the file. This crucially leaves security and bug linters active.

#### Option 2: "Ratchet B" (File-Specific Configuration Overrides)
If you want to prevent a legacy file from decaying *further* without silencing the linter entirely, use **File-Specific Ratcheting**.
*   Do **not** relax the global threshold in your `.golangci.yml` or `.eslintrc.json`.
*   Instead, add a per-path override that locks the legacy file to its *current* complexity score.
*   **Example (.golangci.yml):**
    ```yaml
    issues:
      exclude-rules:
        - path: 'legacy_router\.go'
          linters:
            - gocognit
          text: "cognitive complexity 70 of func `.*` is high" 
    ```
*   **The Result:** The build passes today. If someone adds more complexity tomorrow (pushing it to 71), the build fails. The technical debt is quarantined and ratcheted.

---

## 🤖 Guidance for AI Agents

If you are an AI agent executing an audit:
1.  **Do not act unilaterally.** If you believe a file belongs in Category B (Cohesive Monolith), you **must stop and ask the human user for explicit permission** before applying an Honest Exception or a Ratchet.
2.  **Never touch global configs.** You are strictly forbidden from relaxing global thresholds (e.g., changing `max-complexity: 8` to `max-complexity: 50` for the whole repo).
3.  **Prefer Ratchet B.** When granted permission to bypass a check for a legacy file, default to configuring a file-specific override (Ratchet B) in the configuration file rather than using a blanket `//nolint` comment, as this preserves the ratchet effect.

---

## 📖 Case Study: Self-Dogfooding `maintainability-sensors`

When we audited the `maintainability-sensors` repository itself, we encountered massive switch statements mapping AST nodes to metrics.
*   Instead of relaxing the global limit, we built a **3-Layer Defense System** natively into the AST parsing:
    1.  **Cyclomatic Complexity:** Catches raw path explosion. (Suppressed via inline comments for flat switches).
    2.  **Cognitive Complexity:** Catches nested spaghetti code hidden *inside* those switches.
    3.  **Max Case Length:** Prevents developers (and agents) from hiding massive linear code blocks inside a single `case`.
*   By layering these metrics and applying Honest Exceptions precisely where needed, we successfully forced the entire codebase into strict compliance without sacrificing semantic readability. (See [Case Studies](CASE_STUDIES.md) for full details).