# Governing the AI Agent Loop: Automated Self-Correction 🤖

This guide explains how to establish the **deterministic feedback loop** between your AI Coding Agents (such as Gemini CLI, Cursor, Claude Code, or autonomous agents) and the `maintainability-sensors` CLI to ensure unconstrained AI code generation does not degrade your codebase.

---

## 1. The Challenge of Unconstrained AI Generation

AI coding agents are highly productive at writing boilerplate and implementing features. However, they lack context regarding long-term codebase maintenance. Left ungoverned:
*   They generate monolithic, nested, and tangled code structures because they are optimized for "immediate functional passing" over "clean architecture."
*   They are highly prone to **regression-stacking** (introducing micro-refactorings that break adjacent modules).

By integrating **Maintainability Sensors** directly into their workspace, you turn passive rules (guides) into **active, automated gatekeepers**.

---

## 2. The Solution: Autonomous `check-diff` Validation

Instead of scanning the entire codebase (which might have existing legacy debt), the AI agent must validate its *own* specific changes using `check-diff`. This ensures the agent is only held responsible for the code it just generated or modified.

```
┌─────────────────────────────────────────────────────────────┐
│              1. Agent Generates Code                        │
└──────────────────────────────┬──────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│    2. Agent Autonomously Runs `check-diff` on Delta         │
└──────────────────────────────┬──────────────────────────────┘
                               │
                    (Sensor Violations Found?)
                               │
               ┌───────────────┴───────────────┐
              YES                              NO
               │                               │
               ▼                               ▼
┌──────────────────────────────┐              ┌────────────────┐
│ 3. CLI Outputs Structured    │              │ 4. Code Ready  │
│    Correction Prompt Blocks  │              │    for PR      │
└──────────────┬───────────────┘              └────────────────┘
               │
               ▼
┌──────────────────────────────┐
│ 4. Agent Ingests Prompt &    │
│    Refactors Its Own Code    │
└──────────────────────────────┘
```

### Step 1: Bootstrap the Environment
First, ensure that your repository has active maintainability sensors configured. Ask the agent or run:
```bash
maintainability-sensors bootstrap .
```
This writes pristine configurations with strict, deterministic thresholds (e.g., cyclomatic complexity limit 8, max function lines 50, max params 4).

### Step 2: The Agent's Internal Verification Loop
When the AI agent finishes generating a feature or a fix, it must intrinsically run the following command before presenting the code to the user or opening a PR:

```bash
maintainability-sensors check-diff origin/main
```
*(Or diffing against the current HEAD if working entirely locally with uncommitted changes)*

### Step 3: Self-Correction via Prompts
If the `check-diff` command fails, the tool outputs structured, machine-optimized **Refactoring Instructions**:

```
  * Complexity is 12 (Max 8) on lines 45-80. Nudge coding agent to extract nested conditionals into separate, single-responsibility helper functions.
```

The AI agent will parse this instruction, implement the modular refactoring on the lines it just wrote, run the tests, and loop back to Step 2 until the sensor gate passes.

---

## ⚖️ Elastic Thresholds: Slightly Relaxing the Limits

A key finding is that we should **not** force a binary "suppress-or-comply" choice. Some refactorings are truly impossible or counterproductive.

If the AI agent believes that a threshold should be slightly relaxed for a specific file:
1.  It edits the local configuration file (e.g., increasing `max-params` from 4 to 5).
2.  It **must append an inline comment or configuration reason** explaining why.
3.  Our `maintainability-sensors` scanner will automatically detect this change and flag it under **"Exceptions Created by AI (Relaxed Constraints)"** in the PR scorecard.
4.  This isolates the exact places where the AI struggled to refactor, providing the **perfect starting point for human code reviews**.
