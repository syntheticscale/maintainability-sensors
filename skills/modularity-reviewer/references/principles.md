# Vlad Khononov's Modularity Principles

When reviewing code for semantic modularity, base your evaluation on the following principles:

## 1. Semantic Duplication
Code duplication is not just about identical strings of text; it's about duplicating *knowledge* or *intent*. 
- Two functions with completely different syntax but identical business rules represent semantic duplication.
- Look for copy-pasted patterns where only the data types or superficial variable names have changed.
- **Rule:** Isolate and centralize shared business knowledge to prevent drift.

## 2. Inefficient Arguments (Primitive Obsession & Stamp Coupling)
How data flows into a module dictates its coupling.
- Passing long lists of primitive arguments (`string id, string name, string email, string phone`) deep into call stacks creates brittle code.
- **Rule:** Group related data into cohesive Parameter Objects (e.g., a `User` or `ContactInfo` object).

## 3. Misplaced Responsibilities (Cohesion)
A module should do one thing, and the things inside a module should belong together.
- Look for "god modules" that try to orchestrate, apply business rules, and save to the database all at once.
- Look for UI components that make raw API calls or process complex business logic.
- **Rule:** Separate concerns strictly. Keep business logic pure and push I/O, database access, and UI rendering to the boundaries.