# Development principles

Before changing code, inspect the affected source, callers, contracts and relevant tests. Decide the responsibility, placement and simplest fitting design. Keep one responsibility per file and one cohesive feature per package; split at actual ownership boundaries.

Prefer the target language's native features, standard library and established project conventions. Reuse existing components. Keep control flow, side effects, cancellation and errors explicit. Preserve unrelated work and keep optional expansion outside the request.

Use package-qualified imports or project-configured aliases rather than file-relative imports such as ./ or ../. Match existing formatting and intended line endings.

Check behavior affected by the change and checks explicitly required by the user or project. Distinguish actual successful checks, failed checks and unverified behavior. A historical command describes its execution time, not the correctness of later changes.
