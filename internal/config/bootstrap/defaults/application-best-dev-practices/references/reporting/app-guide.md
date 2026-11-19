# Reporting, dashboards and analysis

Use this when people inspect metrics, change scope, drill into records and share a result. Start from the decision and the data needed to support it.

[Do & Don’t](do-and-dont.md) · [Implementation Q&A](implementation-qa.md)

## Main flow

1. Open a view and show the metric, time range, unit, source scope and applicable freshness. Explain the comparison period or grouping where needed.
2. Change filters and render results for the accepted query. Keep retained results associated with the query that produced them while an update is pending.
3. Inspect records or export using the declared scope. Distinguish a displayed page, all matches and a snapshot used for reconciliation.
4. Handle no matches, unavailable data and failures. Offer the next supported action and preserve the meaning of an unknown measurement.

## Implementation boundaries

- Define metrics from actual dimensions, units, aggregation and denominator. Bind intervals to the relevant time zone and boundary semantics.
- Apply row and column access in the underlying query, drill-down and export. Account for the exposure of derived results when access is restricted.
- For consistent comparisons, name the dataset or revision and define acceptable staleness across cards. Measure costly reads before adding preaggregation or caching.
- Keep applied scope, pending query and results correlated. Reject stale results after a scope, unit or date range change.

## Borrow for this workflow

Use [Forms](../forms/app-guide.md) for versioned responses, [Administrative workspace](../admin-workspace/app-guide.md) for permission and bulk scope, and [Offline fieldwork](../offline-fieldwork/app-guide.md) for queued and accepted records.

Metabase Documentation “Dashboard filters” describes connecting filters to dashboard cards.

## Acceptance cases

- Change the report range twice while a slow query runs. The final view and export use the intended scope.
- Compare an actual zero with empty, missing and delayed data. The displayed states preserve their meaning.
- Remove access and verify dashboard cards, drill-down and requested exports against current permissions.
