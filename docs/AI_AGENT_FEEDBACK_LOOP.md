# Governing the AI Agent Loop: Automated Self-Correction 🤖

This guide explains how to establish the **deterministic feedback loop** between your AI Coding Agents (such as Cursor, Claude Code, Copilot, or customized autonomous coding agents) and the `maintainability-sensors` CLI to ensure unconstrained AI code generation does not degrade your codebase.

---

## 1. The Challenge of Unconstrained AI Generation

AI coding agents are highly productive at writing boilerplate and implementing features. However, they lack context regarding long-term codebase maintenance. Left ungoverned:
*   They generate monolithic, nested, and tangled code structures because they are optimized for "immediate functional passing" over "clean architecture."
*   They are highly prone to **regression-stacking** (introducing micro-refactorings that break adjacent modules).

By integrating **Maintainability Sensors** directly into their workspace, you turn passive rules (guides) into **active, automated gatekeepers**.

---

## 2. Setting Up the Feedback Loop (Step-by-Step)

```
┌─────────────────────────────────────────────────────────────┐
│              1. Agent Generates Code (Cursor)               │
└──────────────────────────────┬──────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│        2. Local Pre-commit / Terminal Scan Fires            │
└──────────────────────────────┬──────────────────────────────┘
                               │
                    (Sensor Violations Found?)
                               │
               ┌───────────────┴───────────────┐
              YES                              NO
               │                               │
               ▼                               ▼
┌──────────────────────────────┐              ┌────────────────┐
│ 3. CLI Outputs Self-         │              │ 4. PR Merged   │
│    Correction Prompt Blocks  │              │    Successfully│
└──────────────┬───────────────┘              └────────────────┘
               │
               ▼
┌──────────────────────────────┐
│ 4. Agent Ingests Prompt &    │
│    Refactors Itself          │
└──────────────────────────────┘
```

### Step 1: Bootstrap the Environment
First, ensure that your repository has active maintainability sensors configured. Run:
```bash
maintainability-sensors bootstrap .
```
This writes pristine `.eslintrc.json`, `.pylintrc`, or `.golangci.yml` configurations with strict, deterministic thresholds (e.g. cyclomatic complexity limit 8, max function lines 50, max params 4).

### Step 2: Configure as a Local git Pre-commit Hook
To ensure that *no* developer or AI agent can commit code that violates the maintainability rules, configure the sensors to run as a **git pre-commit hook**.

Create or update `.git/hooks/pre-commit` in your repository:
```bash
#!/bin/bash

# Run maintainability sensors on the staged files
staged_files=$(git diff --cached --name-only --diff-filter=ACM | grep -E '\.(ts|tsx|js|jsx|py|go)$')

if [ -z "$staged_files" ]; then
    exit 0
fi

# Run scan. If there are violations, print recommendations and exit 1
maintainability-sensors run $staged_files

if [ $? -ne 0 ]; then
    echo "========================================================="
    echo " [ERROR] Maintainability Sensor Violations Detected!"
    echo " Feed the prompts above to your AI Agent to refactor."
    echo "========================================================="
    exit 1
fi
```
Make the hook executable:
```bash
chmod +x .git/hooks/pre-commit
```

### Step 3: Feeding Prompt Blocks back to Cursor / Claude Code
When the pre-commit hook or manual scan fails, the tool outputs structured, machine-optimized **Refactoring Instructions**:

```
  * Complexity is 12 (Max 8). Nudge coding agent to extract nested conditionals into separate, single-responsibility helper functions.
```

When using **Cursor (composer/agent mode)** or **Claude Code**, simply paste this block or tell the agent:
> *"The maintainability sensors blocked my commit because `Complexity is 12 (Max 8)`. Read the error output and refactor the nested conditionals in `MyComponent.tsx` into separate helper functions to satisfy the sensor. Do not alter external behavior."*

The agent will parse the instruction, implement the modular refactoring, run the test, and pass the sensor gate.

---

## ⚖️ Elastic Thresholds: Slightly Relaxing the Limits

A key finding in Birgitta Böckeler's blog post is that we should **not** force a binary "suppress-or-comply" choice. Some refactorings are truly impossible or counterproductive.

If the AI agent or developer believes that a threshold should be slightly relaxed:
1.  They edit the local configuration file (e.g., increasing `max-params` in `.eslintrc.json` from 4 to 5).
2.  They **must append an inline comment or configuration reason** explaining why.
3.  Our `maintainability-sensors` scanner will automatically detect this change and flag it under **"Exceptions Created by AI (Relaxed Constraints)"** in the PR scorecard.
4.  This isolates the exact places where the AI struggled to refactor, providing the **perfect starting point for human code reviews**.
