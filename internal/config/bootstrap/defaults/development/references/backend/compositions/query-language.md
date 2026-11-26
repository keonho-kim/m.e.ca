# Bounded query and rule compilation

Choose when callers need a stable composable filter/rule language that ordinary typed parameters no longer express well. Do not create a DSL for a few fixed conditions. A Specification can be a named pure predicate without a rule framework.

## Representation and ownership

```text
untrusted filter text/JSON
  → bounded parser → typed AST → scope/type/cost checks
  → SQL/search compiler with parameters → bounded execution
  → authorized projection
```

The parser owns grammar and syntax errors; the application owns supported semantics, authorization and cost limits; the adapter owns engine syntax/operator translation. An AND/OR/group tree can use Composite structure. Ordinary traversal or a Visitor can perform validation/compilation when multiple operations over a stable AST justify it. An Interpreter is useful only if direct evaluation is actually required.

## Preserve semantics across representations

Define null/missing values, case/Unicode normalization, timezone/precision, empty sets, short-circuit behavior and rejection reasons. If the same rule runs in memory and SQL, verify equivalence on boundary cases; collations, nulls and numeric coercion can disagree. Bind tenant scope independently so user OR expressions cannot remove it.

Allowlist identifiers, operators, types and sort keys. Parameterization handles values, not arbitrary SQL identifiers or executable expressions. Compile once per immutable normalized query/version when reuse earns the cache cost; include schema/analyzer and relevant scope in cache identity.

## Bound traversal and engine work

Limit AST bytes/nodes/depth, expansion, regex/fuzzy cost, join depth, candidate size and execution time. A small serialized request can describe expensive work. Charge admission for known cost features and observe actual cost to refine the estimate; do not promise accurate cost from request length alone.

For hierarchical domain data, choose adjacency/recursive queries, materialized paths or closure relations from read/move cost, and detect cycles. Recursion depth alone does not bound fan-out. See [access models](../storage/access-models.md) and PostgreSQL's [recursive query documentation](https://www.postgresql.org/docs/current/queries-with.html).

Pure condition evaluation must not unexpectedly issue network/database calls. If a condition needs fresh external facts, the use case obtains them at an explicit boundary and binds their version/lifetime. A shared Blackboard is at most a typed bounded observation record with defined writers, not a hidden service locator.

## Cost and verification

Compare direct typed filters with AST parsing, validation, compilation, caching and diagnostic overhead. An AST can centralize semantics and avoid repeated text processing, but it also creates a language contract and another retained representation. Keep only the representations actually consumed.

Test nested negation/OR, empty/null fields, Unicode equivalents, cycles, expensive expansions, cross-tenant attempts and cancellation. Compare in-memory and database results where both are supported. Use bounded generated inputs/property checks and explicit complexity limits; do not evaluate untrusted code as a shortcut.
