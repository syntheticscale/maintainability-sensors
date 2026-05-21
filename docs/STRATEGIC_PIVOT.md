# Strategic Pivot: From CI Gatekeeper to AI Agent Skill

## The Problem with "Hard Gates"
Early iterations of `maintainability-sensors` operated as a rigid CI/CD gatekeeper. It would scan an entire repository and block PRs if any code breached cyclomatic complexity or length thresholds. 

While theoretically sound, this approach revealed severe systemic flaws:
1. **The Baseline Paradox:** To allow legacy code to pass, the tool recorded the historical *maximum* complexity of a file. This inadvertently gave developers and AI agents massive "headroom" to write terrible new functions in historically complex files without triggering a failure.
2. **Punishing the Human:** AI agents lack global architectural context. Punishing the human developer with a CI failure 10 minutes after the AI pushed the code creates immense sociotechnical friction.
3. **Misalignment with the Original Vision:** Birgitta Böckeler’s original concept advocated for *sensors and nudges* during the active coding phase, enabling human-in-the-loop decision-making—not a blunt CI sledgehammer.

## The Dual-Purpose Architecture
To solve these flaws, `maintainability-sensors` is pivoting to a dual-purpose architecture, specifically optimized to run as a **Skill** for AI coding assistants (like Cursor, Copilot, or Claude Code).

### 1. Audit Mode (Day 1 Legacy Analysis)
- **Command:** `maintainability-sensors run .`
- **Purpose:** Scans the entire repository.
- **Use Case:** Used by Tech Leads to generate a one-time HTML scorecard or JSON baseline of existing legacy debt. It establishes the current state of the codebase but is *not* used to blindly block ongoing work.

### 2. Delta Mode (The Agent Skill)
- **Command:** `maintainability-sensors check-diff` (or triggered via LSP)
- **Purpose:** Analyzes only the lines of code actively modified in the current working tree or `git diff`.
- **Use Case:** This is the core operational mode. It is designed to be invoked autonomously by AI coding agents *before* they finalize a task.

#### Why Delta Mode is the Holy Grail:
- **No Punishment for Legacy Debt:** If an agent adds a safe 3-line patch to a historically 500-line legacy function, it passes. The AI is not penalized for sins of the past.
- **Instant Rot Detection:** If the agent generates a brand new, overly complex monolithic function, the Delta Mode catches it instantly in the editor.
- **The Nudge:** When flagged, the AI can pause its thought loop to refactor its own work, or it can insert an explicit exception comment (e.g., `// TODO(AI): High complexity required due to legacy API structure`). This explicitly fulfills Birgitta's vision: forcing the AI to highlight the exception for the human reviewer.

## Next Steps for Implementation
1. **Develop `check-diff`:** Implement `git diff` parsing to filter the orchestrated plugin metrics down to only modified line ranges.
2. **Package as a Skill:** Author the tool's interaction model as a formal "Skill" that AI assistants can ingest and utilize autonomously during their coding loops.