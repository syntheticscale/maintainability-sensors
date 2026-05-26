# Delta Mode Implementation Strategy

To successfully transition from a repository-wide CI gatekeeper (Audit Mode) to a surgical, AI-assisting skill (Delta Mode), we must fundamentally change how we process and filter metrics. 

Currently, the tool calculates and returns the *absolute maximum* complexity or function length for a whole file. To evaluate only the "delta" (the code actively written by the agent), we need line-level granularity.

Here is the detailed, 4-phase technical strategy to implement this pivot.

---

## Phase 1: Interface Evolution (Line-Level Granularity)
**The Problem:** The current `Plugin` interface (`Analyze(filePaths []string) (map[string]MaintainabilityMetrics, error)`) collapses all linter violations into a single, file-wide `MaintainabilityMetrics` struct. We lose the line numbers of where the violations actually occurred.
**The Fix:** Redefine the `Plugin` interface to return an array of line-specific violations for each file.

1. **Update Data Structures:**
   ```go
   type Violation struct {
       RuleName string // e.g., "Cyclomatic Complexity"
       Value    int    // e.g., 15
       Line     int    // e.g., 42
       Message  string // e.g., "complexity of 15 (max 8)"
   }
   // The new Plugin Interface
   type Plugin interface {
       Name() string
       Analyze(filePaths []string) (map[string][]Violation, error)
   }
   ```
2. **Refactor Tier 1 (Native) Plugins:**
   - Update `ParseGoAST`, `ParseJava`, and `ParseCSharp` in `go-tree-sitter` to capture and return the starting line number of the offending functions/methods.
3. **Refactor Tier 2 (Orchestrated) Plugins:**
   - Update `ESLintPlugin`, `PyLintPlugin`, `RuffPlugin`, etc. Most linter JSON outputs already include a `"line"` property. Extract this property into the `Violation` struct instead of just blindly tracking the `max` value.

---

## Phase 2: Git Diff Parsing (`sensors/git_diff.go`)
**The Problem:** The tool needs to know exactly which lines the AI agent or developer modified.
**The Fix:** We need a robust parser for `git diff` output.

1. **Execute Diff:** Run `git diff HEAD --unified=0` (or `git diff main` depending on CI context). `--unified=0` removes unchanged context lines, making it easier to parse.
2. **Parse Unified Diff:**
   - Look for file headers: `+++ b/src/main.go`
   - Look for chunk headers: `@@ -50,0 +51,10 @@`
   - Calculate the range of *added or modified* lines in the new file (e.g., lines 51 through 60).
3. **Data Model:** Return a map of changed files to their modified line ranges: `map[string][]LineRange{Start, End}`.

---

## Phase 3: The `check-diff` Command (`cli/cmd.go`)
**The Problem:** We need a new entrypoint that ties the git diff and the line-level violations together.
**The Fix:** Implement the `check-diff` CLI command.

1. **Execution Flow:**
   - Call the new Git Diff parser to get `map[string][]LineRange`.
   - Extract the keys (file paths) and pass them to `FindFiles` and the `Orchestrator` to run the plugins.
   - The plugins will return `map[string][]Violation` (all violations in those files).
2. **The Delta Filter:**
   - For each file, iterate through its `[]Violation`.
   - Check if `Violation.Line` falls within any of the `LineRange`s for that file.
   - Discard any violation that does *not* fall on a modified line.
3. **Reporting:**
   - Output the filtered violations strictly as **Agent Self-Correction Prompts**. If the list is empty, return `exit 0` (even if the rest of the file is a legacy nightmare).

---

## Phase 4: AI Agent Skill Packaging
**The Problem:** An AI agent doesn't intrinsically know how to use a CLI tool to self-correct.
**The Fix:** Formalize the tool as an Agent Skill.

1. **Skill Definition (`docs/SKILL.md`):**
   - Write a prompt-optimized markdown file defining the "Maintainability Sensor Skill".
   - It will instruct agents (like Cursor or the Gemini CLI) to autonomously execute `maintainability-sensors check-diff` after modifying code but *before* calling their work complete.
   - Teach the agent how to read the output and how to either refactor its code or insert a standard suppression comment (e.g., `//nolint`) if the complexity is unavoidable.
