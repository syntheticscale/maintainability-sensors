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
   *(Optional: If your changes are already committed locally but not pushed, you can compare against the main branch: `maintainability-sensors check-diff origin/main`)*

2. **Interpret the Output:**
   - **"Delta clean."**: Your code changes are structurally sound. You may proceed.
   - **"AI WARNING: <File>:<Line> - <Rule> - <Message>"**: Your modifications have triggered a maintainability violation (e.g., you added a deeply nested `if` statement to a legacy function, pushing its complexity over the limit).

3. **Self-Correction (The Nudge):**
   If you receive an AI WARNING, you must immediately pause and reconsider your approach.
   - **Attempt to Refactor:** Can you extract a helper function? Can you simplify the boolean logic? Can you use early returns? Refactor your code and run `check-diff` again.
   - **Declare an Exception:** If refactoring is impossible due to tight legacy coupling or strict domain requirements, you must explicitly declare an exception. Add a standard linter suppression comment (e.g., `//nolint:gocognit` for Go, `// eslint-disable-next-line complexity` for JS, `# pylint: disable=too-many-branches` for Python) right above the offending code. You MUST also add a brief comment explaining *why* you had to bypass the limit so the human reviewer understands your decision.

### Golden Rule
Never ignore an AI WARNING. You must either refactor until `check-diff` is clean, or explicitly suppress the warning with a documented inline comment.
</instructions>
</activated_skill>