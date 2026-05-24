---
name: pre-flight-check
description: Mandatory pre-flight check. Invoke this skill before finalizing any feature or bug fix that involves code modifications to run check-diff and the test suite.
---

# Pre-Flight Check

This skill acts as a mandatory pre-flight check for coding agents. It MUST be executed *before* finalizing any feature or bug fix that involves code modifications.

## Workflow Instructions

When finalizing a task that modified code, follow these exact steps sequentially:

1. **Run Check-Diff:**
   Execute the local static analysis orchestrator on the modified files to gather refactoring prompts.
   ```bash
   ./bin/maintainability-sensors check-diff
   ```
   *(Note: Adjust the binary path if necessary, or run `go run ./cmd/maintainability-sensors check-diff` if the binary is not built yet).*

2. **Run the Test Suite:**
   Execute the project's tests to ensure no regressions were introduced.
   ```bash
   go test ./...
   ```
   *(Note: Adjust the test command if the project uses a different test runner or structure, but for Go projects `go test ./...` is the standard).*

3. **Handle Refactoring Prompts & Failures:**
   - **If tests fail:** You MUST pause, diagnose the failure, fix the issue, and restart this pre-flight check from step 1.
   - **If `check-diff` returns any Refactoring Prompts or warnings:** You MUST pause, fix your own code to clear the warnings, and repeat the check. You CANNOT proceed until the check is clean, unless an explicit exception is declared by the user.

4. **Completion:**
   Only after BOTH steps (the `check-diff` output and the tests) are fully successful and clean, can you report completion to the user.

## Important Constraints

- Do not attempt to bypass these checks.
- If you encounter blocking issues or ambiguous requirements while fixing your code to pass these checks, stop and report back to the orchestrating agent or the user immediately.
