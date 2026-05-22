# AI Coding Agent Best Practices: A Case Study

This document is a summary of real interactions with an AI coding agent during the development and refactoring of this project. It highlights effective strategies, common pitfalls, and practical examples of how to get the most out of AI assistants.

## 🌟 What Went Well: Effective Strategies

### 1. Requesting "Radically Candid" Reviews
Instead of just asking "does this look good?", you asked for radical candor. This gives the AI permission to point out architectural flaws, security issues, and bad practices without "sugar-coating" it.
*   **Prompt Example:** *"do a radically candid review of this repo, what's left from it to be a useful tool for most companies..."*
*   **Prompt Example:** *"use another subagent to do yet another review, ask it to be radically candid..."*

### 2. Using Different "Hats" or Personas
You asked the AI to analyze the codebase from specific expert perspectives. This forces the AI to shift its context and identify different classes of problems (e.g., system-level vs. enterprise-level vs. security).
*   **Prompt Example:** *"let's go up a level, what would an architect focus on? or is there a better 'hat' to wear..."*
*   **Prompt Example:** *"Let's go up a level to see what systemic issues there are, would what a distinguished engineer note and think"*
*   **Prompt Example:** *"do another review with a subagent looking again with the enterprise viewpoint"*

### 3. Delegating to Subagents for Execution
Instead of doing everything in one massive context window, you orchestrated the workflow by having the main agent plan, and then delegating the actual coding/fixing to subagents. This keeps the main session fast and focused on strategy.
*   **Prompt Example:** *"yes, let's capture that properly then use subagents to implement if that makes sense, stage the work to what a single subagent could do"*
*   **Prompt Example:** *"use subagents to do all four phases, one at a time"*

### 4. Strategy Before Execution
You forced the AI to write down its plan and get agreement before writing code. This prevents the AI from going down a rabbit hole of incorrect assumptions.
*   **Prompt Example:** *"work out a detailed strategy for the pivot and write that out before we look at using subagents to implement"*
*   **Prompt Example:** *"write out the strategy and plan and then review it again before we look to implement it"*

### 5. "Dogfooding" and Verifying Failures
You didn't just trust that the tests worked; you actively tested the testing mechanism to ensure it would catch failures.
*   **Prompt Example:** *"let's dogfood our own software, we want our checks to run before we commit each time"*
*   **Prompt Example:** *"let's try making a change which should make the hook fail, to see what happens"*

---

## ⚠️ Areas for Improvement: What to Avoid

### 1. The "Try Again" Loop
When an agent fails, simply saying "try again" forces the AI to guess what it did wrong, often leading it to repeat the same mistake or invent a new one. 
*   **What Happened:** You had a sequence of 4 back-to-back *"try again"* prompts.
*   **Better Approach:** Provide the specific error output, or ask the AI: *"That didn't work. Before trying again, explain why it failed and how your new approach will fix it."* (Note: You actually did this brilliantly in other parts of the logs by pasting the CI test failure output!).

### 2. Over-Prompting for Problems
If you repeatedly ask an AI to find problems, it will eventually start hallucinating or nitpicking minor, irrelevant details just to fulfill the prompt.
*   **What Happened:** *"has there been a round of reviews where no further problems found?"* followed by *"find more feedback and what should be done next"*. 
*   **Better Approach:** Define a "Definition of Done". Stop asking for general reviews once the critical and high-priority issues are resolved, and switch to feature-driven or test-driven prompts.

### 3. Relying on the Agent for Git Workflows
While the AI can run Git commands, relying on it to remember to "commit as you go" can be risky, especially if a subagent makes a sweeping change that breaks the build.
*   **What Happened:** *"you didn't commit and push"* / *"remember to commit and push once it's done"*.
*   **Better Approach:** Let the AI focus on writing the code and tests. Use your own terminal to review the `git diff`, run the tests yourself, and commit the changes manually. This keeps you in the driver's seat for version control.

---

## 💡 Key Takeaway for Beginners

Think of the AI not as a junior developer who just writes code, but as a **co-architect**. Spend 80% of your time defining the strategy, setting the boundaries (e.g., *"I prefer component tests instead of having implementation details tested by unit tests"*), and asking it to review its own plans. Once the plan is solid, let the AI (or its subagents) churn out the boilerplate and implementations.
