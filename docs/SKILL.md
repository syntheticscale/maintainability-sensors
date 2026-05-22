# Maintainability Sensors Skill

<activated_skill>
<instructions>
You are now equipped with the **Maintainability Sensors** skill. This skill provides you with a crucial "Delta Mode" capability to ensure that the code you write is structurally sound and does not contribute to the "Big Ball of Mud" anti-pattern.

### Your Mandate
Before you finalize any coding task, state that you are finished, or attempt to commit code, you MUST autonomously verify your changes using the `check-diff` command. 

This tool will analyze your `git diff` and cross-reference the lines you modified against strict maintainability thresholds (Cyclomatic Complexity, Function Length, and Argument Count).

### Workflow
1. **Execute the Check:**
   Run the following command in the repository root:
   ```bash
   maintainability-sensors check-diff
   ```
   *(Optional: If your changes are already committed locally but not pushed, you can compare against the main branch: `maintainability-sensors check-diff --target-branch=origin/main`)*

2. **Interpret the Output:**
   - **"Delta clean."**: Your code changes are structurally sound. You may proceed.
   - **"AI WARNING: <File>:<Line> - <Rule> - <Message>"**: Your modifications have triggered a maintainability violation (e.g., you added a deeply nested `if` statement to a legacy function, pushing its complexity over the limit).

3. **Self-Correction (The Nudge):**
   If you receive an AI WARNING, you must immediately pause and reconsider your approach.
   - **Attempt to Refactor:** Can you extract a helper function? Can you simplify the boolean logic? Can you use early returns? Refactor your code and run `check-diff` again.
   - **Declare an Exception (The Honest Exception Protocol):** If refactoring is impossible due to tight legacy coupling, strict domain requirements, or because the logic is **highly cohesive** (e.g., a centralized router, a massive `switch` statement mapping values, or deeply coupled network protocol logic), DO NOT fragment the code into unreadable helpers just to appease the metric. Instead, add a standard linter suppression comment (e.g., `//nolint:gocognit` for Go, `// eslint-disable-next-line complexity` for JS, `# pylint: disable=too-many-branches` for Python) right above the offending code. You MUST also add a brief comment explaining *why* you had to bypass the limit so the human reviewer understands your decision.

### AI Engineering Best Practices (When Refactoring)
*   **The Parameter Object Pattern:** If a function exceeds the Argument Count threshold (usually 4), do NOT attempt to shuffle data awkwardly. Always introduce a **Parameter Object** (a dedicated struct/interface) to encapsulate the arguments. This is an AI "Super-Pattern" that drastically reduces parameter hallucination and keeps code robust against future API expansions.

### Golden Rule
Never ignore an AI WARNING. You must either refactor until `check-diff` is clean, or explicitly suppress the warning with a documented inline comment.

**CRITICAL:** Your behavior regarding global thresholds depends entirely on your current operational mode:
- **During Routine Coding (Delta Mode):** NEVER relax or increase the global thresholds in the repository's configuration files (e.g., `.golangci.yml`, `.eslintrc.json`, `.pylintrc`) to bypass a violation you just introduced. You must fix the code or use targeted inline suppressions (The Honest Exception Protocol).
- **During Legacy Audits (Audit Mode):** If you are explicitly tasked by a human to audit an existing, legacy codebase, the goal is to establish a truthful baseline. In this scenario, you **MAY** relax the global thresholds in the configuration files to match the existing legacy debt, **BUT ONLY IF** you have explicitly proposed this to the human engineer and received their approval. Human judgment is required to ensure we do not overfit to the metric.

> **Note:** If your repository has `.maintainability-sensors.yml` configured with `default-severity: warn`, warnings are expected and you should not try to fix them unless they indicate actual issues. The `check-diff` command will exit cleanly (0) for warnings.

> **To confirm your repo's policy:** Check if `.maintainability-sensors.yml` exists and what `default-severity` it sets.
</instructions>
</activated_skill>