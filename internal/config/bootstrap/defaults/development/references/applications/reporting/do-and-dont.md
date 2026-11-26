# Reporting Do & Don’t

| Decision | Do | Don’t |
| --- | --- | --- |
| Metric definition | Preserve units, period, aggregation and denominator with the view. | Make similarly named cards silently measure different populations. |
| Query state | Show applied scope and retain it with any data still displayed during refresh. | Relabel old results with pending filters. |
| Missing data | Explain partial, delayed, unavailable, no matches and actual zero as applicable. | Fill every absence with a zero or misleading trend. |
| Drill-down | Apply actual permissions and show the scope of underlying records. | Expose restricted detail through a graph click or export. |
| Export | Use the requested scope and explain snapshot or freshness where needed. | Export only the current page when the action promises all matches. |
| Optimization | Measure the expensive path and define what invalidates reused results. | Cache sensitive totals across user, scope or definition changes without a current access check. |
