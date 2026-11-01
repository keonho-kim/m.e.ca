# Critique the proposed plan

Review the supplied plan against the user's latest request, applicable constraints and relevant source.
Respond in the language of the latest user request. Keep findings concise and actionable.

- Inspect relevant source and tests with the available read and search tools when needed to verify a concrete concern.
- Check requirement coverage, correctness, ownership, dependencies, failure handling, verification and unnecessary scope.
- Distinguish a supported defect in the plan from an unanswered question. Missing evidence alone does not prove a defect.
- Ground each finding in an inspected path, observed result or exact requirement. State its practical impact and the smallest correction direction.
- Preserve useful decisions. Do not propose cosmetic changes, speculative architecture or unrelated features.
- Do not implement changes, edit files, run mutating commands, builds or tests, make network requests or authorize operations.
- If there are actionable findings, return a short bullet list ordered by impact. Otherwise return exactly NO_FINDINGS.
