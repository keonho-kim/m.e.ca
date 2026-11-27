# Team development workflow

Use this guide when the requested engineering work needs coordinated contributions. Derive roles and shared decisions from the outcome the user wants. For coordinated implementation, use domain responsibilities and a shared Spec to agree behavior, then the development skill's test cycle for each agreed change. Keep discussion and investigation scoped to the evidence or decision requested, and reuse existing agreements.

## Agree on domain responsibilities

- Inspect the current code, documentation and workflows. Agree on vocabulary, use cases, invariants, state ownership and consistency boundaries.
- Divide work into cohesive domain slices. Give each shared interface, prerequisite or file an owner and identify who depends on it. Each owner is responsible for the observable behavior and tests in their scope.
- Choose entities, value objects, aggregates or bounded contexts only where they clarify the domain. An existing module can express the boundary.

## Agree on acceptance before implementation

- Reuse one concise project-local Spec for substantial work. Existing acceptance criteria and a brief public agreement are sufficient for a small change.
- Record only relevant outcomes, scope, domain rules, inputs and outputs, failures, state changes, shared interfaces and acceptance cases. Assign one editor and get the affected owners' acceptance in the public discussion.
- Reference the agreed Spec and contracts in the work declaration. The latest user request governs scope. When evidence changes the contract, agree on the affected part with its owners.

## Integrate the changes

- Start independent work once its contracts and prerequisites are ready. Share useful fixtures and verification evidence. Inspect current files and confirmed prerequisite results before integrating.
- Map acceptance cases to behavior tests at the owning boundary. Use consumer and provider contract checks for important handoffs. Exercise complete workflows across the changed domains and resolve mismatches with their owners.
- Report fulfilled acceptance cases, actual integration checks and remaining uncertainty.
