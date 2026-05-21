# Delta Mode & Agent Skill Implementation Strategy

This document outlines the comprehensive technical strategy for shifting `maintainability-sensors` to a "Delta Mode" capability for AI agents. 

## The Core Concept
Instead of passing/failing the entire repository based on legacy debt, the CLI will parse the active workspace changes (`git diff HEAD`) and cross-reference them with linter violations. If an AI agent modifies code that intersects with a maintainability violation, the tool flags it, allowing for real-time self-correction without punishing the agent for pre-existing rot.

---

## Phase 1: Boundary-Aware Plugin Interface
**The Flaw Prevented:** The "Function Boundary Problem." If an agent modifies Line 50, but the linter reports the complexity violation at the function signature on Line 10, a naive `line == line` filter will drop the violation.
**The Solution:** The `Plugin` interface must return a `Violation` struct that defines the entire AST block boundary (`StartLine` and `EndLine`).

1. **Update Data Structures:**
   ```go
   type Violation struct {
       RuleName  string
       Value     int
       StartLine int
       EndLine   int
       Message   string
   }
   type Plugin interface {
       Name() string
       Analyze(filePaths []string) (map[string][]Violation, error)
   }
   ```
2. **Refactor Native Plugins (Go, C#, Java):** Update the `go-tree-sitter` logic to capture both the start and end rows of the offending AST nodes.
3. **Refactor Ecosystem Plugins (ESLint, PyLint, Ruff, RuboCop, Biome):** 
   - Update the JSON decoding logic. All these modern linters provide AST location boundaries in their JSON outputs (e.g., `endLine` in ESLint, `end_location.row` in Ruff, `last_line` in RuboCop). Capture these into `EndLine`.

---

## Phase 2: Comprehensive Git Diff Parsing
**The Flaw Prevented:** The "Untracked File" Blindspot. `git diff` ignores brand new files created by the AI until they are staged or tracked.
**The Solution:** Build a robust `sensors/git_diff.go` module that handles both diffs and untracked files.

1. **Tracked Files Diff:**
   - Execute `git diff HEAD --unified=0`.
   - Parse the unified diff to extract ranges of *added/modified* lines (ignore deleted lines).
   - Produce a map: `map[string][]LineRange{Start, End}`.
2. **Untracked Files:**
   - Execute `git ls-files --others --exclude-standard`.
   - For every untracked file, create a special `LineRange` of `[1, Infinity]` to ensure *all* violations in new files are caught.

---

## Phase 3: The `check-diff` Delta Filter
**The Flaw Prevented:** The "Tipping Point" Frustration. We only want to alert if the agent's work directly overlaps with the violation.

1. **Execution Pipeline:**
   - Run the Git Diff parser to get the map of modified files.
   - Run the `PluginRegistry` *only* on the files present in the map.
2. **The Intersection Filter:**
   - For each `Violation` returned by the plugin, check if its boundary (`[StartLine, EndLine]`) *intersects* with any of the `LineRange`s from the `git diff` for that file.
   - If it intersects, keep the violation. If it doesn't, discard it.
3. **Output:** 
   - Print the filtered list as structured self-correction prompts.

---

## Phase 4: AI Skill Packaging (`SKILL.md`)
**The Flaw Prevented:** The agent won't know how to use the tool unless taught.
**The Solution:** Create a formal Gemini CLI Skill payload.

1. Author a `SKILL.md` that instructs the agent:
   - "Before finalizing your task, run `maintainability-sensors check-diff`."
   - "If it returns violations, you MUST attempt to refactor the code."
   - "If refactoring is impossible due to legacy coupling, add a suppression comment (e.g. `//nolint`) and explain why in your PR comment."