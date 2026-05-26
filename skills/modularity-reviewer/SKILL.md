---
name: modularity-reviewer
description: Performs a Semantic Modularity Review of code changes using LLM-as-a-judge based on Vlad Khononov's modularity principles. Use when requested to review modularity, evaluate a completed feature, or analyze semantic duplication, inefficient arguments, and misplaced responsibilities.
---

# Modularity Reviewer

This skill enables Gemini CLI to perform a semantic "inferential" modularity review of code.
Mathematical linters lack semantic context. An AI agent might accidentally duplicate the exact same business logic in two different files, just using different variable names. Our tools will report both files as clean. We need an inferential sensor to evaluate the *meaning* of the code, not just its syntax.

## Workflow

Invoke this skill after a major feature implementation, when requested to review code modularity, or as part of a post-coding validation step.

When executing a modularity review, follow these steps:

1. **Gather Context:** Analyze the recent code changes (e.g., via `git diff HEAD` or by reading the newly created files).
2. **Review Principles:** Read and understand the modularity principles defined in `references/principles.md`.
3. **Evaluate Code:** Evaluate the code changes against the three core areas:
   - **Semantic Duplication:** Does this business logic or semantic meaning exist elsewhere in the codebase?
   - **Inefficient Arguments:** Are primitive arguments passed too deeply instead of using a Parameter Object?
   - **Misplaced Responsibilities:** Is data access mixed with UI/business logic?
4. **Report:** Return a structured evaluation providing clear, actionable feedback for the user or the acting agent on how to refactor the code to improve semantic modularity.

If no issues are found, explicitly state that the code aligns with the modularity principles.