# Reporting implementation Q&A

## What should happen when cards update separately?

Show meaningful card-level status. If the task needs a consistent comparison, use a shared dataset snapshot or wait for results that correspond to the same accepted scope. For retained values, show the query or freshness that produced them.

## Which time zone governs a period?

Choose the organization's reporting zone, the viewer's zone or another explicit definition according to the metric. Use the same boundary for cards, drill-down and export. Exercise the end of a day and offset changes for relevant reports.

## What does export include?

State whether the action exports visible rows, selected IDs or all matches. Capture its scope and revision when needed. Recheck authorization before generating or delivering it and expose a supported state for long-running exports.

## When is a rollup worth adding?

Establish the cost of the actual query and the acceptable delay for the decision. A rollup earns its place when repeated work is expensive and the team can explain freshness, scope, repair and removal. Verify the aggregate against representative underlying records.
